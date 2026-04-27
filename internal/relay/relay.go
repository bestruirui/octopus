package relay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/helper"
	dbmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/relay/balancer"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/transformer/inbound"
	"github.com/lingyuins/octopus/internal/transformer/model"
	"github.com/lingyuins/octopus/internal/transformer/outbound"
	"github.com/lingyuins/octopus/internal/utils/log"
	"github.com/tmaxmax/go-sse"
)

func resolveRequestedUpstreamModel(requestModel string) (string, bool) {
	trimmed := strings.TrimSpace(requestModel)
	if trimmed == "" {
		return "", false
	}
	prefix, upstream, ok := strings.Cut(trimmed, "/")
	if !ok {
		return "", false
	}
	if !strings.EqualFold(strings.TrimSpace(prefix), "zen") {
		return "", false
	}
	upstream = strings.TrimSpace(upstream)
	if upstream == "" {
		return "", false
	}
	return upstream, true
}

func detectZenPreferredChannelTypes(requestModel string, isEmbeddingRequest bool) map[outbound.OutboundType]struct{} {
	upstreamModel, ok := resolveRequestedUpstreamModel(requestModel)
	if !ok {
		return nil
	}
	if isEmbeddingRequest {
		return map[outbound.OutboundType]struct{}{
			outbound.OutboundTypeOpenAIEmbedding: {},
		}
	}

	lowerModel := strings.ToLower(strings.TrimSpace(upstreamModel))
	switch {
	case strings.HasPrefix(lowerModel, "claude"):
		return map[outbound.OutboundType]struct{}{
			outbound.OutboundTypeAnthropic: {},
		}
	case strings.HasPrefix(lowerModel, "gemini"), strings.HasPrefix(lowerModel, "models/gemini"), strings.HasPrefix(lowerModel, "gemma"):
		return map[outbound.OutboundType]struct{}{
			outbound.OutboundTypeGemini: {},
		}
	case strings.HasPrefix(lowerModel, "gpt-"), strings.HasPrefix(lowerModel, "o1"), strings.HasPrefix(lowerModel, "o3"), strings.HasPrefix(lowerModel, "o4"), strings.HasPrefix(lowerModel, "text-embedding"), strings.HasPrefix(lowerModel, "text-moderation"):
		return map[outbound.OutboundType]struct{}{
			outbound.OutboundTypeOpenAIChat:     {},
			outbound.OutboundTypeOpenAIResponse: {},
			outbound.OutboundTypeVolcengine:     {},
		}
	default:
		return nil
	}
}

func isZenCandidateChannelAllowed(requestModel string, channelType outbound.OutboundType, isEmbeddingRequest bool) bool {
	preferred := detectZenPreferredChannelTypes(requestModel, isEmbeddingRequest)
	if len(preferred) == 0 {
		return true
	}
	_, ok := preferred[channelType]
	return ok
}

func resolveCandidateModelName(requestModel string, item dbmodel.GroupItem) string {
	if upstreamModel, ok := resolveRequestedUpstreamModel(requestModel); ok {
		if strings.TrimSpace(item.ModelName) == "" || strings.EqualFold(strings.TrimSpace(item.ModelName), "zen") {
			return upstreamModel
		}
	}
	return item.ModelName
}

// Handler 处理入站请求并转发到上游服务
func Handler(endpointType string, inboundType inbound.InboundType, c *gin.Context) {
	// 解析请求
	internalRequest, inAdapter, err := parseRequest(inboundType, c)
	if err != nil {
		return
	}
	supportedModels := c.GetString("supported_models")
	if supportedModels != "" {
		supportedModelsArray := strings.Split(supportedModels, ",")
		for i := range supportedModelsArray {
			supportedModelsArray[i] = strings.TrimSpace(supportedModelsArray[i])
		}
		if !slices.Contains(supportedModelsArray, internalRequest.Model) {
			resp.Error(c, http.StatusBadRequest, "model not supported")
			return
		}
	}

	requestModel := internalRequest.Model
	apiKeyID := c.GetInt("api_key_id")
	var streamSession *relayStreamSession
	var streamSessionOwned bool
	var lastErr error
	if shouldUseRelayStreamSession(internalRequest) {
		sessionHash := buildRelayStreamSessionHash(endpointType, int(inboundType), apiKeyID, internalRequest.RawRequest)
		session, created, err := acquireRelayStreamSession(internalRequest.ConversationID, apiKeyID, sessionHash)
		if err != nil {
			statusCode := http.StatusConflict
			if !errors.Is(err, errRelayConversationBusy) {
				statusCode = http.StatusInternalServerError
			}
			resp.Error(c, statusCode, err.Error())
			return
		}
		streamSession = session
		streamSessionOwned = created
		if !created {
			req := &relayRequest{
				c:               c,
				clientCtx:       c.Request.Context(),
				internalRequest: internalRequest,
				apiKeyID:        apiKeyID,
				requestModel:    requestModel,
				streamSession:   streamSession,
			}
			serveRelayStreamSession(c, req)
			return
		}
		defer func() {
			if streamSession == nil || !streamSessionOwned || streamSession.IsDone() {
				return
			}
			if lastErr == nil {
				lastErr = errors.New("relay stream ended without a terminal result")
			}
			streamSession.Finish(lastErr)
		}()
	}

	operationCtx, cancel := newRelayOperationContext()
	defer cancel()

	// 获取通道分组
	group, err := op.GroupGetEnabledMapByEndpoint(endpointType, requestModel, operationCtx)
	if err != nil {
		lastErr = err
		resp.Error(c, http.StatusNotFound, "model not found")
		return
	}

	// 创建迭代器（策略排序 + 粘性优先）
	iter := balancer.NewIterator(group, apiKeyID, requestModel)
	if iter.Len() == 0 {
		lastErr = errors.New("no available channel")
		resp.Error(c, http.StatusServiceUnavailable, "no available channel")
		return
	}

	// 初始化 Metrics
	metrics := NewRelayMetrics(apiKeyID, requestModel, internalRequest)

	// 请求级上下文
	req := &relayRequest{
		c:                 c,
		clientCtx:         c.Request.Context(),
		operationCtx:      operationCtx,
		inAdapter:         inAdapter,
		internalRequest:   internalRequest,
		metrics:           metrics,
		apiKeyID:          apiKeyID,
		requestModel:      requestModel,
		groupEndpointType: group.EndpointType,
		iter:              iter,
		streamSession:     streamSession,
	}

	maxAttemptsPerCandidate := getMaxAttemptsPerCandidate()
	ratelimitCooldown := getRatelimitCooldown()
	maxTotalAttempts := getMaxTotalAttempts()

outer:
	for iter.Next() {
		if maxTotalAttempts > 0 && iter.ForwardedAttempts() >= maxTotalAttempts {
			lastErr = fmt.Errorf("reached relay max total attempts: %d", maxTotalAttempts)
			break
		}
		if err := req.operationCtx.Err(); err != nil {
			lastErr = err
			logRelayErrorfByContext(err, "relay operation ended before request completed: %v", err)
			metrics.Save(false, err, iter.Attempts())
			return
		}

		item := iter.Item()

		// 获取通道
		channel, err := op.ChannelGet(item.ChannelID, req.operationCtx)
		if err != nil {
			log.Warnf("failed to get channel %d: %v", item.ChannelID, err)
			iter.Skip(item.ChannelID, 0, fmt.Sprintf("channel_%d", item.ChannelID), fmt.Sprintf("channel not found: %v", err))
			lastErr = err
			continue
		}
		if !channel.Enabled {
			iter.Skip(channel.ID, 0, channel.Name, "channel disabled")
			continue
		}

		usedKey := channel.GetChannelKeyWithCooldown(ratelimitCooldown)
		if usedKey.ChannelKey == "" {
			iter.Skip(channel.ID, 0, channel.Name, "no available key")
			continue
		}

		resolvedModelName := resolveCandidateModelName(requestModel, item)
		if strings.TrimSpace(resolvedModelName) == "" {
			iter.Skip(channel.ID, usedKey.ID, channel.Name, "resolved upstream model is empty")
			continue
		}

		// 熔断检查
		if iter.SkipCircuitBreak(channel.ID, usedKey.ID, channel.Name, resolvedModelName) {
			continue
		}

		// 出站适配器
		outAdapter := outbound.Get(channel.Type)
		if outAdapter == nil {
			iter.Skip(channel.ID, usedKey.ID, channel.Name, fmt.Sprintf("unsupported channel type: %d", channel.Type))
			continue
		}

		// 类型兼容性检查
		if internalRequest.IsEmbeddingRequest() && !outbound.IsEmbeddingChannelType(channel.Type) {
			iter.Skip(channel.ID, usedKey.ID, channel.Name, "channel type not compatible with embedding request")
			continue
		}
		if internalRequest.IsChatRequest() && !outbound.IsChatChannelType(channel.Type) {
			iter.Skip(channel.ID, usedKey.ID, channel.Name, "channel type not compatible with chat request")
			continue
		}
		if !isZenCandidateChannelAllowed(requestModel, channel.Type, internalRequest.IsEmbeddingRequest()) {
			iter.Skip(channel.ID, usedKey.ID, channel.Name, "channel type not preferred for zen model prefix")
			continue
		}

		// 设置实际模型
		internalRequest.Model = resolvedModelName

		var failedKeyIDs []int
	innerRetry:
		for tryIndex := 1; tryIndex <= maxAttemptsPerCandidate; tryIndex++ {
			if maxTotalAttempts > 0 && iter.ForwardedAttempts() >= maxTotalAttempts {
				lastErr = fmt.Errorf("reached relay max total attempts: %d", maxTotalAttempts)
				break outer
			}
			if err := req.operationCtx.Err(); err != nil {
				lastErr = err
				logRelayErrorfByContext(err, "relay operation ended before request completed: %v", err)
				metrics.Save(false, err, iter.Attempts())
				return
			}

			if tryIndex > 1 {
				usedKey = channel.GetChannelKeyExcludingWithCooldown(failedKeyIDs, ratelimitCooldown)
				if usedKey.ChannelKey == "" {
					log.Infof("channel %s has no more keys to retry, moving to next channel", channel.Name)
					break innerRetry
				}
				if iter.SkipCircuitBreak(channel.ID, usedKey.ID, channel.Name, resolvedModelName) {
					failedKeyIDs = append(failedKeyIDs, usedKey.ID)
					tryIndex--
					continue
				}
			}

			log.Infof("request model %s, mode: %d, forwarding to channel: %s model: %s key_id: %d (candidate %d/%d, attempt %d/%d, sticky=%t)",
				requestModel, group.Mode, channel.Name, resolvedModelName, usedKey.ID,
				iter.Index()+1, iter.Len(), tryIndex, maxAttemptsPerCandidate, iter.IsSticky())

			ra := &relayAttempt{
				relayRequest:         req,
				outAdapter:           outAdapter,
				channel:              channel,
				usedKey:              usedKey,
				firstTokenTimeOutSec: group.FirstTokenTimeOut,
				tryIndex:             tryIndex,
				tryTotal:             maxAttemptsPerCandidate,
			}

			result := ra.attempt()
			if result.Success {
				metrics.Save(true, nil, iter.Attempts())
				return
			}

			// 根据错误分类决策进行重试控制
			switch result.Decision.Scope {
			case ScopeNone:
				// 不重试，直接失败
				lastErr = result.Err
				metrics.Save(false, lastErr, iter.Attempts())
				resp.BadGateway(c)
				return

			case ScopeAbortAll:
				// 停止所有重试（流式响应已写入）
				lastErr = result.Err
				metrics.Save(false, result.Err, iter.Attempts())
				return

			case ScopeSameChannel:
				// 同候选换 Key 重试
				lastErr = result.Err
				failedKeyIDs = append(failedKeyIDs, usedKey.ID)
				// 继续内层循环，尝试下一个 Key
				continue innerRetry

			case ScopeNextChannel:
				// 换下一个候选重试
				lastErr = result.Err
				failedKeyIDs = append(failedKeyIDs, usedKey.ID)
				// 跳出内层循环，进入下一个候选
				break innerRetry

			default:
				// 未知决策，保守停止
				lastErr = result.Err
				metrics.Save(false, lastErr, iter.Attempts())
				resp.BadGateway(c)
				return
			}
		}
	}

	// 所有通道都失败
	metrics.Save(false, lastErr, iter.Attempts())
	resp.Error(c, http.StatusBadGateway, "all channels failed")
}

// attempt 统一管理一次通道尝试的完整生命周期
func (ra *relayAttempt) attempt() attemptResult {
	span := ra.iter.StartAttempt(ra.channel.ID, ra.usedKey.ID, ra.channel.Name, ra.internalRequest.Model)

	// 转发请求
	statusCode, fwdErr := ra.forward()

	// 检查是否已写入流式响应
	written := ra.c.Writer.Written()

	// 使用错误分类驱动决策
	decision := ClassifyRelayError(statusCode, fwdErr, written)

	// 更新 channel key 状态
	ra.usedKey.StatusCode = statusCode
	ra.usedKey.LastUseTimeStamp = time.Now().Unix()

	if decision.Scope == ScopeNone && !decision.IsError {
		// ====== 成功 ======
		ra.collectResponse()
		ra.usedKey.TotalCost += ra.metrics.Stats.InputCost + ra.metrics.Stats.OutputCost
		op.ChannelKeyUpdate(ra.usedKey)

		span.End(dbmodel.AttemptSuccess, statusCode, "")

		// Channel 维度统计
		updateChannelSuccessStats(ra.channel.ID, span.Duration().Milliseconds(), ra.metrics.Stats)

		// 熔断器：记录成功
		balancer.RecordSuccess(ra.channel.ID, ra.usedKey.ID, ra.internalRequest.Model)
		// Auto策略：记录成功
		balancer.RecordAutoSuccess(ra.channel.ID, ra.internalRequest.Model)
		// 会话保持：更新粘性记录
		balancer.SetSticky(ra.apiKeyID, ra.requestModel, ra.channel.ID, ra.usedKey.ID)

		return attemptResult{Success: true, Decision: decision}
	}

	// ====== 失败 ======
	op.ChannelKeyUpdate(ra.usedKey)

	// 构造日志消息
	msg := decision.String()
	if ra.tryTotal > 1 {
		msg = fmt.Sprintf("attempt %d/%d: %s", ra.tryIndex, ra.tryTotal, msg)
	}
	span.End(dbmodel.AttemptFailed, statusCode, msg)

	// Channel 维度统计
	op.StatsChannelUpdate(ra.channel.ID, dbmodel.StatsMetrics{
		WaitTime:      span.Duration().Milliseconds(),
		RequestFailed: 1,
	})

	// 熔断器和 Auto 策略：只在换候选或停止时记录失败
	// 换 Key 重试不触发熔断计数，避免误熔断
	if decision.Scope == ScopeNextChannel || decision.Scope == ScopeAbortAll {
		balancer.RecordFailure(ra.channel.ID, ra.usedKey.ID, ra.internalRequest.Model)
		balancer.RecordAutoFailure(ra.channel.ID, ra.internalRequest.Model)
	}

	if written {
		ra.collectResponse()
	}

	// 记录决策日志
	if decision.IsError {
		logRelayErrorfByContext(fwdErr, "channel %s failed on attempt %d/%d: %s (decision: %s)",
			ra.channel.Name, ra.tryIndex, ra.tryTotal, fwdErr, decision.Scope.String())
	}

	return attemptResult{
		Success:  false,
		Written:  written,
		Err:      fmt.Errorf("channel %s failed on attempt %d/%d: %v", ra.channel.Name, ra.tryIndex, ra.tryTotal, fwdErr),
		Decision: decision,
	}
}

// parseRequest 解析并验证入站请求
func parseRequest(inboundType inbound.InboundType, c *gin.Context) (*model.InternalLLMRequest, model.Inbound, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return nil, nil, err
	}

	inAdapter := inbound.Get(inboundType)
	internalRequest, err := inAdapter.TransformRequest(c.Request.Context(), body)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return nil, nil, err
	}

	// Pass through the original query parameters
	internalRequest.Query = c.Request.URL.Query()

	if err := internalRequest.Validate(); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return nil, nil, err
	}

	populateRelayRequestSessionFields(c, internalRequest, body)

	return internalRequest, inAdapter, nil
}

// forward 转发请求到上游服务
func (ra *relayAttempt) forward() (int, error) {
	ctx := ra.operationCtx

	requestForOutbound, err := prepareInternalRequestForOutbound(ra.channel, ra.internalRequest, ra.groupEndpointType)
	if err != nil {
		log.Warnf("failed to prepare outbound request data: %v", err)
		return 0, fmt.Errorf("failed to prepare outbound request data: %w", err)
	}

	// 构建出站请求
	outboundRequest, err := ra.outAdapter.TransformRequest(
		ctx,
		requestForOutbound,
		ra.channel.GetBaseUrl(),
		ra.usedKey.ChannelKey,
	)
	if err != nil {
		log.Warnf("failed to create request: %v", err)
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// 复制请求头
	ra.copyHeaders(outboundRequest)

	// 发送请求
	response, err := ra.sendRequest(outboundRequest)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	// 检查响应状态
	statusCode, err := ra.handleForwardResponse(response)
	if err != nil {
		return statusCode, err
	}

	// 处理响应
	if ra.internalRequest.Stream != nil && *ra.internalRequest.Stream {
		if err := ra.handleStreamResponse(ctx, response); err != nil {
			return 0, err
		}
		return response.StatusCode, nil
	}
	if err := ra.handleResponse(ctx, response); err != nil {
		return 0, err
	}
	return response.StatusCode, nil
}

func (ra *relayAttempt) handleForwardResponse(response *http.Response) (int, error) {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return response.StatusCode, nil
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}
	return response.StatusCode, fmt.Errorf("upstream error: %d: %s", response.StatusCode, string(body))
}

// copyHeaders 复制请求头，过滤 hop-by-hop 头
func (ra *relayAttempt) copyHeaders(outboundRequest *http.Request) {
	for key, values := range ra.c.Request.Header {
		if hopByHopHeaders[strings.ToLower(key)] {
			continue
		}
		for _, value := range values {
			outboundRequest.Header.Set(key, value)
		}
	}
	if len(ra.channel.CustomHeader) > 0 {
		for _, header := range ra.channel.CustomHeader {
			outboundRequest.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}
}

// sendRequest 发送 HTTP 请求
func (ra *relayAttempt) sendRequest(req *http.Request) (*http.Response, error) {
	httpClient, err := helper.ChannelHttpClient(ra.channel)
	if err != nil {
		log.Warnf("failed to get http client: %v", err)
		return nil, err
	}

	response, err := httpClient.Do(req)
	if err != nil {
		logRelayErrorfByContext(err, "failed to send request: %v", err)
		return nil, err
	}

	return response, nil
}

// handleStreamResponse 处理流式响应
func (ra *relayAttempt) handleStreamResponse(ctx context.Context, response *http.Response) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if ct := response.Header.Get("Content-Type"); ct != "" && !strings.Contains(strings.ToLower(ct), "text/event-stream") {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		return fmt.Errorf("upstream returned non-SSE content-type %q for stream request: %s", ct, string(body))
	}

	// 设置 SSE 响应头
	ra.c.Header("Content-Type", "text/event-stream")
	ra.c.Header("Cache-Control", "no-cache")
	ra.c.Header("Connection", "keep-alive")
	ra.c.Header("X-Accel-Buffering", "no")
	if ra.internalRequest.ConversationID != "" {
		ra.c.Header("X-Conversation-ID", ra.internalRequest.ConversationID)
	}

	firstToken := true
	clientDone := ra.clientCtx.Done()
	clientDisconnected := false
	clientDisconnectLogged := false
	markClientDisconnected := func() {
		if clientDisconnected {
			return
		}
		clientDisconnected = true
		clientDone = nil
	}
	logClientDisconnected := func() {
		if !clientDisconnected || clientDisconnectLogged {
			return
		}
		clientDisconnectLogged = true
		log.Warnf(clientDisconnectedLogMessage)
	}

	type sseReadResult struct {
		data string
		err  error
	}
	results := make(chan sseReadResult, 1)
	go func() {
		defer close(results)
		readCfg := &sse.ReadConfig{MaxEventSize: maxSSEEventSize}
		for ev, err := range sse.Read(response.Body, readCfg) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err != nil {
				select {
				case results <- sseReadResult{err: err}:
				case <-ctx.Done():
				}
				return
			}
			select {
			case results <- sseReadResult{data: ev.Data}:
			case <-ctx.Done():
				return
			}
		}
	}()

	var firstTokenTimer *time.Timer
	var firstTokenC <-chan time.Time
	if firstToken && ra.firstTokenTimeOutSec > 0 {
		firstTokenTimer = time.NewTimer(time.Duration(ra.firstTokenTimeOutSec) * time.Second)
		firstTokenC = firstTokenTimer.C
		defer func() {
			if firstTokenTimer != nil {
				firstTokenTimer.Stop()
			}
		}()
	}

	for {
		select {
		case <-clientDone:
			if ra.streamSession == nil {
				log.Infof("client disconnected, stopping stream")
				return nil
			}
			markClientDisconnected()
		case <-firstTokenC:
			logClientDisconnected()
			log.Warnf("first token timeout (%ds), switching channel", ra.firstTokenTimeOutSec)
			_ = response.Body.Close()
			return fmt.Errorf("first token timeout (%ds)", ra.firstTokenTimeOutSec)
		case r, ok := <-results:
			if !ok {
				logClientDisconnected()
				if ra.streamSession != nil {
					ra.streamSession.Finish(nil)
				}
				log.Infof("stream end")
				return nil
			}
			if r.err != nil {
				logClientDisconnected()
				logRelayErrorfByContext(r.err, "failed to read event: %v", r.err)
				return fmt.Errorf("failed to read stream event: %w", r.err)
			}

			data, err := ra.transformStreamData(ctx, r.data)
			if err != nil || len(data) == 0 {
				continue
			}
			if firstToken {
				ra.metrics.SetFirstTokenTime(time.Now())
				firstToken = false
				if firstTokenTimer != nil {
					if !firstTokenTimer.Stop() {
						select {
						case <-firstTokenTimer.C:
						default:
						}
					}
					firstTokenTimer = nil
					firstTokenC = nil
				}
			}

			if ra.streamSession != nil {
				sessionEvents := ra.streamSession.AddPayload(data)
				if clientDisconnected {
					logClientDisconnected()
					continue
				}
				for _, event := range sessionEvents {
					if _, err := ra.c.Writer.Write(formatRelaySSEEvent(event.Sequence, event.Payload)); err != nil {
						markClientDisconnected()
						logClientDisconnected()
						break
					}
					ra.c.Writer.Flush()
				}
				continue
			}

			if clientDisconnected {
				logClientDisconnected()
				continue
			}
			if _, err := ra.c.Writer.Write(data); err != nil {
				markClientDisconnected()
				logClientDisconnected()
				continue
			}
			ra.c.Writer.Flush()
		}
	}
}

// transformStreamData 转换流式数据
func (ra *relayAttempt) transformStreamData(ctx context.Context, data string) ([]byte, error) {
	internalStream, err := ra.outAdapter.TransformStream(ctx, []byte(data))
	if err != nil {
		logRelayErrorfByContext(err, "failed to transform stream: %v", err)
		return nil, err
	}
	if internalStream == nil {
		return nil, nil
	}

	inStream, err := ra.inAdapter.TransformStream(ctx, internalStream)
	if err != nil {
		logRelayErrorfByContext(err, "failed to transform stream: %v", err)
		return nil, err
	}

	return inStream, nil
}

// handleResponse 处理非流式响应
func (ra *relayAttempt) handleResponse(ctx context.Context, response *http.Response) error {
	internalResponse, err := ra.outAdapter.TransformResponse(ctx, response)
	if err != nil {
		logRelayErrorfByContext(err, "failed to transform response: %v", err)
		return fmt.Errorf("failed to transform outbound response: %w", err)
	}

	inResponse, err := ra.inAdapter.TransformResponse(ctx, internalResponse)
	if err != nil {
		logRelayErrorfByContext(err, "failed to transform response: %v", err)
		return fmt.Errorf("failed to transform inbound response: %w", err)
	}

	ra.c.Data(http.StatusOK, "application/json", inResponse)
	return nil
}

// collectResponse 收集响应信息
func (ra *relayAttempt) collectResponse() {
	internalResponse, err := ra.inAdapter.GetInternalResponse(ra.operationCtx)
	if err != nil || internalResponse == nil {
		return
	}

	ra.metrics.SetInternalResponse(internalResponse, ra.internalRequest.Model)
}
