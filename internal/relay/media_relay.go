package relay

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/helper"
	dbmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/relay/balancer"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/utils/log"
)

func mediaEndpointTypeToGroupEndpointType(endpointType MediaEndpointType) string {
	switch endpointType {
	case MediaEndpointImageGeneration:
		return dbmodel.EndpointTypeImageGeneration
	case MediaEndpointImageEdit:
		return dbmodel.EndpointTypeImageGeneration
	case MediaEndpointImageVariation:
		return dbmodel.EndpointTypeImageGeneration
	case MediaEndpointAudioSpeech:
		return dbmodel.EndpointTypeAudioSpeech
	case MediaEndpointAudioTranscription:
		return dbmodel.EndpointTypeAudioTranscription
	case MediaEndpointVideoGeneration:
		return dbmodel.EndpointTypeVideoGeneration
	case MediaEndpointMusicGeneration:
		return dbmodel.EndpointTypeMusicGeneration
	case MediaEndpointSearch:
		return dbmodel.EndpointTypeSearch
	case MediaEndpointRerank:
		return dbmodel.EndpointTypeRerank
	case MediaEndpointModeration:
		return dbmodel.EndpointTypeModerations
	default:
		return dbmodel.EndpointTypeAll
	}
}

// MediaHandler handles non-LLLM media/utility endpoints by forwarding requests
// directly to upstream channels, reusing the existing channel/group/balancer/circuit-breaker
// infrastructure without going through the Inbound/Outbound transformer pipeline.
func MediaHandler(endpointType MediaEndpointType, c *gin.Context) {
	cfg := getMediaEndpointConfig(endpointType)

	// 1. Extract model name from the request
	requestModel, bodyBytes, streamRequested, err := extractModelFromRequest(c, cfg)
	if err != nil {
		resp.Error(c, relayRequestBodyErrorStatus(err), err.Error())
		return
	}
	if cfg.MultipartInput && c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if requestModel == "" {
		resp.Error(c, http.StatusBadRequest, "model is required")
		return
	}

	apiKeyID := c.GetInt("api_key_id")

	// 2. Resolve channel group
	groupEndpointType := mediaEndpointTypeToGroupEndpointType(endpointType)
	group, err := op.GroupGetEnabledMapByEndpoint(groupEndpointType, requestModel, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusNotFound, "model not found")
		return
	}

	// 3. Create load balancer iterator
	iter := balancer.NewIterator(group, apiKeyID, requestModel)
	if iter.Len() == 0 {
		resp.Error(c, http.StatusServiceUnavailable, "no available channel")
		return
	}

	ratelimitCooldown := getRatelimitCooldown()
	maxAttemptsPerCandidate := getMaxAttemptsPerCandidate()
	maxTotalAttempts := getMaxTotalAttempts()

	var lastErr error

outer:
	for iter.Next() {
		if maxTotalAttempts > 0 && iter.ForwardedAttempts() >= maxTotalAttempts {
			lastErr = fmt.Errorf("reached relay max total attempts: %d", maxTotalAttempts)
			break
		}
		select {
		case <-c.Request.Context().Done():
			log.Infof("request context canceled, stopping media retry")
			return
		default:
		}

		item := iter.Item()

		prepare := PrepareCandidate(c.Request.Context(), item, iter, ratelimitCooldown, requestModel, nil)
		if prepare.SkipReason != "" {
			recordPreparedCandidateSkip(iter, item, prepare)
			continue
		}

		channel := prepare.Channel
		usedKey := prepare.UsedKey
		resolvedModel := prepare.ResolvedModel

		var failedKeyIDs []int
	innerRetry:
		for tryIndex := 1; tryIndex <= maxAttemptsPerCandidate; tryIndex++ {
			if maxTotalAttempts > 0 && iter.ForwardedAttempts() >= maxTotalAttempts {
				lastErr = fmt.Errorf("reached relay max total attempts: %d", maxTotalAttempts)
				break outer
			}
			select {
			case <-c.Request.Context().Done():
				log.Infof("request context canceled, stopping media retry")
				return
			default:
			}

			if tryIndex > 1 {
				usedKey = channel.GetChannelKeyExcludingWithCooldown(failedKeyIDs, ratelimitCooldown)
				if usedKey.ChannelKey == "" {
					log.Infof("channel %s has no more keys to retry, moving to next channel", channel.Name)
					break innerRetry
				}
				if iter.SkipCircuitBreak(channel.ID, usedKey.ID, channel.Name, resolvedModel) {
					failedKeyIDs = append(failedKeyIDs, usedKey.ID)
					tryIndex--
					continue
				}
			}

			log.Infof("media relay: endpoint=%d, model=%s, forwarding to channel: %s model: %s key_id: %d (candidate %d/%d, attempt %d/%d)",
				endpointType, requestModel, channel.Name, resolvedModel, usedKey.ID,
				iter.Index()+1, iter.Len(), tryIndex, maxAttemptsPerCandidate)

			span := iter.StartAttempt(channel.ID, usedKey.ID, channel.Name, resolvedModel)

			// Build and send upstream request
			statusCode, fwdErr := forwardMediaRequest(c, cfg, channel, usedKey.ChannelKey, bodyBytes, requestModel, resolvedModel, streamRequested)

			// 检查是否已写入响应（媒体端点可能是流式）
			written := c.Writer.Written()

			// 使用错误分类驱动决策
			decision := ClassifyRelayError(statusCode, fwdErr, written)

			usedKey.StatusCode = statusCode
			usedKey.LastUseTimeStamp = time.Now().Unix()

			if decision.Scope == ScopeNone && !decision.IsError {
				// Success
				// Media endpoints don't have token-based cost, so TotalCost is left unchanged.
				op.ChannelKeyUpdate(usedKey)
				span.End(dbmodel.AttemptSuccess, statusCode, "")
				op.StatsChannelUpdate(channel.ID, dbmodel.StatsMetrics{
					WaitTime:       span.Duration().Milliseconds(),
					RequestSuccess: 1,
				})
				balancer.RecordSuccess(channel.ID, usedKey.ID, resolvedModel)
				balancer.RecordAutoSuccess(channel.ID, resolvedModel)
				balancer.SetSticky(apiKeyID, requestModel, channel.ID, usedKey.ID)
				return
			}

			// Failure
			op.ChannelKeyUpdate(usedKey)

			// 构造日志消息
			msg := decision.String()
			if maxAttemptsPerCandidate > 1 {
				msg = fmt.Sprintf("attempt %d/%d: %s", tryIndex, maxAttemptsPerCandidate, msg)
			}
			span.End(dbmodel.AttemptFailed, statusCode, msg)
			op.StatsChannelUpdate(channel.ID, dbmodel.StatsMetrics{
				WaitTime:      span.Duration().Milliseconds(),
				RequestFailed: 1,
			})

			// 熔断器和 Auto 策略：只在换候选或停止时记录失败
			if decision.Scope == ScopeNextChannel || decision.Scope == ScopeAbortAll {
				balancer.RecordFailure(channel.ID, usedKey.ID, resolvedModel)
				balancer.RecordAutoFailure(channel.ID, resolvedModel)
			}

			if decision.IsError {
				log.Warnf("media relay: channel %s failed on attempt %d/%d: %v (decision: %s)",
					channel.Name, tryIndex, maxAttemptsPerCandidate, fwdErr, decision.Scope.String())
			}

			// 根据错误分类决策进行重试控制
			switch decision.Scope {
			case ScopeNone:
				// 不重试，直接失败
				lastErr = fwdErr
				resp.Error(c, http.StatusBadGateway, lastErr.Error())
				return

			case ScopeAbortAll:
				// 停止所有重试（流式响应已写入）
				return

			case ScopeSameChannel:
				// 同候选换 Key 重试
				lastErr = fwdErr
				failedKeyIDs = append(failedKeyIDs, usedKey.ID)
				continue innerRetry

			case ScopeNextChannel:
				// 换下一个候选重试
				lastErr = fwdErr
				failedKeyIDs = append(failedKeyIDs, usedKey.ID)
				break innerRetry

			default:
				// 未知决策，保守停止
				lastErr = fwdErr
				resp.Error(c, http.StatusBadGateway, lastErr.Error())
				return
			}
		}
	}

	// All channels failed
	resp.Error(c, http.StatusBadGateway, fmt.Sprintf("all channels failed: %v", lastErr))
}

func recordPreparedCandidateSkip(iter *balancer.Iterator, item dbmodel.GroupItem, prepare PrepareCandidateResult) {
	if prepare.SkipReason == "" {
		return
	}
	// PrepareCandidate already records circuit-break rejections with cooldown details.
	if prepare.SkipStatus == dbmodel.AttemptCircuitBreak {
		return
	}

	channelID := item.ChannelID
	channelName := fmt.Sprintf("channel_%d", item.ChannelID)
	keyID := 0
	if prepare.Channel != nil {
		channelID = prepare.Channel.ID
		channelName = prepare.Channel.Name
	}
	if prepare.UsedKey.ID != 0 {
		keyID = prepare.UsedKey.ID
	}
	iter.Skip(channelID, keyID, channelName, prepare.SkipReason)
}

// extractModelFromRequest extracts the model name from the request body.
// For JSON endpoints, it parses the body into a generic map.
// For multipart endpoints, it reads the form field.
func extractModelFromRequest(c *gin.Context, cfg mediaEndpointConfig) (string, []byte, bool, error) {
	if cfg.MultipartInput {
		return extractModelFromMultipart(c)
	}
	return extractModelFromJSON(c)
}

// extractModelFromJSON reads the JSON body and extracts the "model" field.
func extractModelFromJSON(c *gin.Context) (string, []byte, bool, error) {
	body, err := readLimitedRequestBody(c, maxRelayJSONBodyBytes)
	if err != nil {
		return "", nil, false, err
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", nil, false, fmt.Errorf("invalid JSON body: %w", err)
	}

	model, _ := raw["model"].(string)
	streamRequested := parseMediaStreamFlag(raw["stream"])
	return model, body, streamRequested, nil
}

// extractModelFromMultipart extracts the model from a multipart/form-data request.
func extractModelFromMultipart(c *gin.Context) (string, []byte, bool, error) {
	limitRequestBody(c, maxRelayMultipartBodyBytes)

	// Parse the multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return "", nil, false, normalizeRelayRequestBodyError(err)
	}

	model := c.Request.FormValue("model")
	streamRequested := strings.EqualFold(strings.TrimSpace(c.Request.FormValue("stream")), "true")
	// We'll re-read the full multipart body in forwardMediaRequestMultipart
	return model, nil, streamRequested, nil
}

// forwardMediaRequest builds and sends the upstream request, then streams the response back.
func forwardMediaRequest(
	c *gin.Context,
	cfg mediaEndpointConfig,
	channel *dbmodel.Channel,
	key string,
	bodyBytes []byte,
	requestModel string,
	resolvedModel string,
	streamRequested bool,
) (int, error) {
	if cfg.MultipartInput {
		return forwardMediaRequestMultipart(c, cfg, channel, key, requestModel, resolvedModel, streamRequested)
	}
	return forwardMediaRequestJSON(c, cfg, channel, key, bodyBytes, requestModel, resolvedModel, streamRequested)
}

// forwardMediaRequestJSON handles JSON-based media endpoint forwarding.
func forwardMediaRequestJSON(
	c *gin.Context,
	cfg mediaEndpointConfig,
	channel *dbmodel.Channel,
	key string,
	bodyBytes []byte,
	requestModel string,
	resolvedModel string,
	streamRequested bool,
) (int, error) {
	ctx := c.Request.Context()

	// Replace model name in the JSON body
	modifiedBody, err := replaceModelInJSON(bodyBytes, requestModel, resolvedModel)
	if err != nil {
		return 0, fmt.Errorf("failed to replace model in request: %w", err)
	}

	// Build upstream URL
	upstreamURL, err := buildMediaUpstreamURL(channel.GetBaseUrl(), cfg.UpstreamPath)
	if err != nil {
		return 0, fmt.Errorf("failed to build upstream URL: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(modifiedBody))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	copyMediaForwardHeaders(req, c, channel, key, "application/json", streamRequested)

	// Send request
	httpClient, err := helper.ChannelHttpClient(channel)
	if err != nil {
		return 0, fmt.Errorf("failed to get http client: %w", err)
	}

	response, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(response.Body, 4*1024))
		return response.StatusCode, fmt.Errorf("upstream error: %d: %s", response.StatusCode, string(respBody))
	}

	// Stream response back to client
	if cfg.BinaryResponse {
		return handleBinaryResponse(c, response)
	}
	if isMediaSSEResponse(response) {
		return handleSSEResponse(c, response)
	}
	return handleJSONResponse(c, response)
}

// forwardMediaRequestMultipart handles multipart/form-data media endpoint forwarding.
func forwardMediaRequestMultipart(
	c *gin.Context,
	cfg mediaEndpointConfig,
	channel *dbmodel.Channel,
	key string,
	requestModel string,
	resolvedModel string,
	streamRequested bool,
) (int, error) {
	ctx := c.Request.Context()

	// Build upstream URL
	upstreamURL, err := buildMediaUpstreamURL(channel.GetBaseUrl(), cfg.UpstreamPath)
	if err != nil {
		return 0, fmt.Errorf("failed to build upstream URL: %w", err)
	}

	bodyReader, contentType := buildMultipartForwardBody(c.Request.MultipartForm, resolvedModel)

	// Create upstream request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bodyReader)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	copyMediaForwardHeaders(req, c, channel, key, contentType, streamRequested)

	// Send request
	httpClient, err := helper.ChannelHttpClient(channel)
	if err != nil {
		return 0, fmt.Errorf("failed to get http client: %w", err)
	}

	response, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(response.Body, 4*1024))
		return response.StatusCode, fmt.Errorf("upstream error: %d: %s", response.StatusCode, string(respBody))
	}

	if isMediaSSEResponse(response) {
		return handleSSEResponse(c, response)
	}
	return handleJSONResponse(c, response)
}

func parseMediaStreamFlag(raw any) bool {
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true")
	default:
		return false
	}
}

func buildMultipartForwardBody(form *multipart.Form, resolvedModel string) (io.ReadCloser, string) {
	reader, writer := io.Pipe()
	mpWriter := multipart.NewWriter(writer)
	contentType := mpWriter.FormDataContentType()

	go func() {
		defer writer.Close()
		defer mpWriter.Close()
		defer func() {
			if r := recover(); r != nil {
				_ = writer.CloseWithError(fmt.Errorf("panic in multipart builder: %v", r))
			}
		}()

		if form == nil {
			return
		}

		for fieldName, values := range form.Value {
			for _, value := range values {
				fieldValue := value
				if fieldName == "model" && resolvedModel != "" {
					fieldValue = resolvedModel
				}
				if err := mpWriter.WriteField(fieldName, fieldValue); err != nil {
					_ = writer.CloseWithError(fmt.Errorf("failed to write field %s: %w", fieldName, err))
					return
				}
			}
		}

		for fieldName, fileHeaders := range form.File {
			for _, fileHeader := range fileHeaders {
				file, err := fileHeader.Open()
				if err != nil {
					_ = writer.CloseWithError(fmt.Errorf("failed to open uploaded file: %w", err))
					return
				}

				part, err := mpWriter.CreateFormFile(fieldName, fileHeader.Filename)
				if err != nil {
					file.Close()
					_ = writer.CloseWithError(fmt.Errorf("failed to create form file: %w", err))
					return
				}
				if _, err := io.Copy(part, file); err != nil {
					file.Close()
					_ = writer.CloseWithError(fmt.Errorf("failed to copy file content: %w", err))
					return
				}
				file.Close()
			}
		}
	}()

	return reader, contentType
}

// replaceModelInJSON replaces the model field value in a JSON body.
func replaceModelInJSON(body []byte, originalModel, resolvedModel string) ([]byte, error) {
	if resolvedModel == "" || resolvedModel == originalModel {
		return body, nil
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		log.Debugf("replaceModelInJSON: failed to parse JSON body, returning original: %v", err)
		return body, nil
	}

	raw["model"] = resolvedModel
	return json.Marshal(raw)
}

// buildMediaUpstreamURL constructs the full upstream URL from base URL and path.
func buildMediaUpstreamURL(baseURL, path string) (string, error) {
	parsed, err := url.Parse(strings.TrimSuffix(baseURL, "/"))
	if err != nil {
		return "", fmt.Errorf("failed to parse base url: %w", err)
	}

	basePath := strings.TrimSuffix(parsed.Path, "/")
	normalizedPath := path
	if strings.HasSuffix(basePath, "/v1") && strings.HasPrefix(normalizedPath, "/v1/") {
		normalizedPath = strings.TrimPrefix(normalizedPath, "/v1")
	}

	parsed.Path = basePath + normalizedPath
	return parsed.String(), nil
}

// applyChannelHeaders applies channel custom headers to the request.
func applyChannelHeaders(req *http.Request, channel *dbmodel.Channel) {
	if len(channel.CustomHeader) > 0 {
		for _, header := range channel.CustomHeader {
			req.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}
}

func copyMediaForwardHeaders(req *http.Request, c *gin.Context, channel *dbmodel.Channel, key string, contentType string, streamRequested bool) {
	for headerKey, values := range c.Request.Header {
		if hopByHopHeaders[strings.ToLower(headerKey)] {
			continue
		}
		if strings.EqualFold(headerKey, "Authorization") || strings.EqualFold(headerKey, "Content-Type") || strings.EqualFold(headerKey, "Content-Length") {
			continue
		}
		for _, value := range values {
			req.Header.Add(headerKey, value)
		}
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if streamRequested {
		req.Header.Set("Accept", "text/event-stream")
	}
	req.Header.Set("Authorization", "Bearer "+key)
	applyChannelHeaders(req, channel)
}

// handleBinaryResponse streams a binary response (e.g. audio) back to the client.
func handleBinaryResponse(c *gin.Context, response *http.Response) (int, error) {
	// Copy relevant headers
	if ct := response.Header.Get("Content-Type"); ct != "" {
		c.Header("Content-Type", ct)
	}
	c.Header("Content-Disposition", response.Header.Get("Content-Disposition"))

	_, err := io.Copy(c.Writer, response.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to stream binary response: %w", err)
	}

	return response.StatusCode, nil
}

func isMediaSSEResponse(response *http.Response) bool {
	if response == nil {
		return false
	}
	return strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream")
}

func handleSSEResponse(c *gin.Context, response *http.Response) (int, error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	reader := bufio.NewReader(response.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if _, writeErr := c.Writer.Write(line); writeErr != nil {
				return 0, fmt.Errorf("failed to stream sse response: %w", writeErr)
			}
			c.Writer.Flush()
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return response.StatusCode, nil
			}
			return 0, fmt.Errorf("failed to read sse response: %w", err)
		}
	}
}

// handleJSONResponse streams a JSON response back to the client.
func handleJSONResponse(c *gin.Context, response *http.Response) (int, error) {
	// For large responses (e.g. image generation with base64), stream directly
	if ct := response.Header.Get("Content-Type"); ct != "" {
		c.Header("Content-Type", ct)
	}

	_, err := io.Copy(c.Writer, response.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to stream response: %w", err)
	}

	return response.StatusCode, nil
}
