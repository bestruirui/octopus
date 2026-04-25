package relay

import (
	"bytes"
	"encoding/json"
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
	requestModel, bodyBytes, err := extractModelFromRequest(c, cfg)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
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
			statusCode, fwdErr := forwardMediaRequest(c, cfg, channel, usedKey.ChannelKey, bodyBytes, requestModel, resolvedModel)

			// 检查是否已写入响应（媒体端点可能是流式）
			written := c.Writer.Written()

			// 使用错误分类驱动决策
			decision := ClassifyRelayError(statusCode, fwdErr, written)

			usedKey.StatusCode = statusCode
			usedKey.LastUseTimeStamp = time.Now().Unix()

			if decision.Scope == ScopeNone && !decision.IsError {
				// Success
				usedKey.TotalCost += 0 // Media endpoints don't have token-based cost
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
func extractModelFromRequest(c *gin.Context, cfg mediaEndpointConfig) (string, []byte, error) {
	if cfg.MultipartInput {
		return extractModelFromMultipart(c)
	}
	return extractModelFromJSON(c)
}

// extractModelFromJSON reads the JSON body and extracts the "model" field.
func extractModelFromJSON(c *gin.Context) (string, []byte, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read request body: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", nil, fmt.Errorf("invalid JSON body: %w", err)
	}

	model, _ := raw["model"].(string)
	return model, body, nil
}

// extractModelFromMultipart extracts the model from a multipart/form-data request.
func extractModelFromMultipart(c *gin.Context) (string, []byte, error) {
	// Parse the multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return "", nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	model := c.Request.FormValue("model")
	// We'll re-read the full multipart body in forwardMediaRequestMultipart
	return model, nil, nil
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
) (int, error) {
	if cfg.MultipartInput {
		return forwardMediaRequestMultipart(c, cfg, channel, key, requestModel, resolvedModel)
	}
	return forwardMediaRequestJSON(c, cfg, channel, key, bodyBytes, requestModel, resolvedModel)
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

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	// Apply channel custom headers
	applyChannelHeaders(req, channel)

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
) (int, error) {
	ctx := c.Request.Context()

	// Build upstream URL
	upstreamURL, err := buildMediaUpstreamURL(channel.GetBaseUrl(), cfg.UpstreamPath)
	if err != nil {
		return 0, fmt.Errorf("failed to build upstream URL: %w", err)
	}

	// Reconstruct the multipart request for upstream
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Copy form fields (replacing model if needed)
	if c.Request.MultipartForm != nil {
		for fieldName, values := range c.Request.MultipartForm.Value {
			for _, value := range values {
				fieldValue := value
				if fieldName == "model" && resolvedModel != "" {
					fieldValue = resolvedModel
				}
				if err := writer.WriteField(fieldName, fieldValue); err != nil {
					return 0, fmt.Errorf("failed to write field %s: %w", fieldName, err)
				}
			}
		}

		// Copy file fields
		for fieldName, fileHeaders := range c.Request.MultipartForm.File {
			for _, fileHeader := range fileHeaders {
				file, err := fileHeader.Open()
				if err != nil {
					return 0, fmt.Errorf("failed to open uploaded file: %w", err)
				}
				part, err := writer.CreateFormFile(fieldName, fileHeader.Filename)
				if err != nil {
					file.Close()
					return 0, fmt.Errorf("failed to create form file: %w", err)
				}
				if _, err := io.Copy(part, file); err != nil {
					file.Close()
					return 0, fmt.Errorf("failed to copy file content: %w", err)
				}
				file.Close()
			}
		}
	}

	if err := writer.Close(); err != nil {
		return 0, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create upstream request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, &buf)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+key)

	// Apply channel custom headers
	applyChannelHeaders(req, channel)

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

	return handleJSONResponse(c, response)
}

// replaceModelInJSON replaces the model field value in a JSON body.
func replaceModelInJSON(body []byte, originalModel, resolvedModel string) ([]byte, error) {
	if resolvedModel == "" || resolvedModel == originalModel {
		return body, nil
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return body, nil // best-effort: return original if parse fails
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
	parsed.Path = parsed.Path + path
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
