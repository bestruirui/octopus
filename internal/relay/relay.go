package relay

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	dbmodel "github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/relay/breaker"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/transformer/inbound"
	tmodel "github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/tmaxmax/go-sse"
)

// Handler 处理入站请求并转发到上游服务
func Handler(inboundType inbound.InboundType, c *gin.Context) {
	// 解析请求
	internalRequest, inAdapter, err := parseRequest(inboundType, c)
	if err != nil {
		return
	}
	supportedModels := c.GetString("supported_models")
	if supportedModels != "" {
		supportedModelsArray := strings.Split(supportedModels, ",")
		if !slices.Contains(supportedModelsArray, internalRequest.Model) {
			resp.Error(c, http.StatusBadRequest, "model not supported")
			return
		}
	}

	// 初始化统计和日志
	apiKeyID := c.GetInt("api_key_id")
	requestModel := internalRequest.Model
	metrics := NewRelayMetrics(requestModel)
	metrics.SetInternalRequest(internalRequest)
	metrics.SetAPIKeyID(apiKeyID)
	// 获取通道分组
	group, err := op.GroupGetMap(internalRequest.Model, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusNotFound, "model not found")
		return
	}
	metrics.SetGroup(group.ID, group.Name)
	cbCfg := breaker.ResolveEffectiveConfig()
	cbManager := breaker.GetManager()

	const maxAttempts = 3
	var lastRelayErr *RelayError
	var lastStatusCode int
	b := balancer.GetBalancer(group.Mode)
	for round := 1; round <= maxAttempts; round++ {
		now := time.Now()
		snapshotItems := cloneGroupItems(group.Items)
		filteredItems, earliestRetryAt := cbManager.FilterAvailable(snapshotItems, now, cbCfg)
		if cbCfg.Enabled && len(filteredItems) == 0 && len(snapshotItems) > 0 {
			cbData := buildAllOpenData(group.ID, requestModel, earliestRetryAt, now)
			retryAfter := cbData.RetryAfterSeconds
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			metrics.AddAttempt(dbmodel.ChannelAttempt{
				Round:             round,
				AttemptNum:        1,
				Success:           false,
				Error:             "all upstream circuit breakers are open",
				CBDecision:        string(breaker.DecisionAllOpen),
				EarliestRetryAt:   cbData.EarliestRetryAt,
				RetryAfterSeconds: cbData.RetryAfterSeconds,
			}, false, 0)
			rerr := newRelayError(http.StatusServiceUnavailable, RelayErrorSourceLocal, true, false, nil, "all upstream circuit breakers are open")
			metrics.Save(c.Request.Context(), false, rerr, 0)
			resp.ErrorWithData(c, http.StatusServiceUnavailable, rerr.Error(), gin.H{
				"circuit_breaker": cbData,
			})
			return
		}

		remainingItems := cloneGroupItems(filteredItems)
		attemptNum := 0
		for len(remainingItems) > 0 {
			attemptNum++
			select {
			case <-c.Request.Context().Done():
				log.Infof("request context canceled, stopping retry")
				return
			default:
			}
			item := b.Select(remainingItems)
			if item == nil {
				break
			}
			remainingItems = removeGroupItem(remainingItems, item.ID)

			attemptStart := time.Now()
			breakerKey := breaker.BuildKey(item.ChannelID, item.ModelName)
			cbActive := cbCfg.Enabled
			channel, err := op.ChannelGet(item.ChannelID, c.Request.Context())
			if err != nil {
				rerr := newRelayError(http.StatusBadGateway, RelayErrorSourceLocal, true, false, err, fmt.Sprintf("failed to get channel: %v", err))
				metrics.AddAttempt(dbmodel.ChannelAttempt{
					Round:            round,
					AttemptNum:       attemptNum,
					ChannelID:        item.ChannelID,
					ModelName:        item.ModelName,
					Error:            rerr.Error(),
					RelayStatusCode:  rerr.StatusCode,
					RelayErrorSource: string(rerr.Source),
					RelayRetryable:   rerr.Retryable,
					RelayTrippable:   rerr.Trippable,
					BreakerKey:       breakerKey,
				}, false, time.Since(attemptStart))
				lastRelayErr = rerr
				lastStatusCode = rerr.StatusCode
				continue
			}
			if channel.Enabled == false {
				rerr := newRelayError(http.StatusBadGateway, RelayErrorSourceLocal, true, false, nil, fmt.Sprintf("channel %s is disabled", channel.Name))
				metrics.AddAttempt(dbmodel.ChannelAttempt{
					Round:            round,
					AttemptNum:       attemptNum,
					ChannelID:        channel.ID,
					ChannelName:      channel.Name,
					ModelName:        item.ModelName,
					Error:            rerr.Error(),
					RelayStatusCode:  rerr.StatusCode,
					RelayErrorSource: string(rerr.Source),
					RelayRetryable:   rerr.Retryable,
					RelayTrippable:   rerr.Trippable,
					BreakerKey:       breakerKey,
				}, false, time.Since(attemptStart))
				lastRelayErr = rerr
				lastStatusCode = rerr.StatusCode
				continue
			}

			log.Infof("request model %s, mode: %d, forwarding to channel: %s model: %s (round %d/%d, item %d)", internalRequest.Model, group.Mode, channel.Name, item.ModelName, round, maxAttempts, attemptNum)

			internalRequest.Model = item.ModelName
			metrics.SetChannel(channel.ID, channel.Name, item.ModelName)

			outAdapter := outbound.Get(channel.Type)
			if outAdapter == nil {
				rerr := newRelayError(http.StatusBadGateway, RelayErrorSourceLocal, true, false, nil, fmt.Sprintf("unsupported channel type: %d", channel.Type))
				metrics.AddAttempt(dbmodel.ChannelAttempt{
					Round:            round,
					AttemptNum:       attemptNum,
					ChannelID:        channel.ID,
					ChannelName:      channel.Name,
					ModelName:        item.ModelName,
					Error:            rerr.Error(),
					RelayStatusCode:  rerr.StatusCode,
					RelayErrorSource: string(rerr.Source),
					RelayRetryable:   rerr.Retryable,
					RelayTrippable:   rerr.Trippable,
					BreakerKey:       breakerKey,
				}, false, time.Since(attemptStart))
				lastRelayErr = rerr
				lastStatusCode = rerr.StatusCode
				continue
			}

			// 验证 channel 类型与请求类型匹配
			if internalRequest.IsEmbeddingRequest() && !outbound.IsEmbeddingChannelType(channel.Type) {
				rerr := newRelayError(http.StatusBadGateway, RelayErrorSourceLocal, true, false, nil, fmt.Sprintf("channel type %d not compatible with embedding request", channel.Type))
				metrics.AddAttempt(dbmodel.ChannelAttempt{
					Round:            round,
					AttemptNum:       attemptNum,
					ChannelID:        channel.ID,
					ChannelName:      channel.Name,
					ModelName:        item.ModelName,
					Error:            rerr.Error(),
					RelayStatusCode:  rerr.StatusCode,
					RelayErrorSource: string(rerr.Source),
					RelayRetryable:   rerr.Retryable,
					RelayTrippable:   rerr.Trippable,
					BreakerKey:       breakerKey,
				}, false, time.Since(attemptStart))
				lastRelayErr = rerr
				lastStatusCode = rerr.StatusCode
				continue
			}

			if internalRequest.IsChatRequest() && !outbound.IsChatChannelType(channel.Type) {
				rerr := newRelayError(http.StatusBadGateway, RelayErrorSourceLocal, true, false, nil, fmt.Sprintf("channel type %d not compatible with chat request", channel.Type))
				metrics.AddAttempt(dbmodel.ChannelAttempt{
					Round:            round,
					AttemptNum:       attemptNum,
					ChannelID:        channel.ID,
					ChannelName:      channel.Name,
					ModelName:        item.ModelName,
					Error:            rerr.Error(),
					RelayStatusCode:  rerr.StatusCode,
					RelayErrorSource: string(rerr.Source),
					RelayRetryable:   rerr.Retryable,
					RelayTrippable:   rerr.Trippable,
					BreakerKey:       breakerKey,
				}, false, time.Since(attemptStart))
				lastRelayErr = rerr
				lastStatusCode = rerr.StatusCode
				continue
			}

			acquire := breaker.AttemptAcquire{
				Key:         breakerKey,
				Allowed:     true,
				Decision:    breaker.DecisionDisabled,
				StateBefore: dbmodel.CircuitBreakerStateClosed,
				StateAfter:  dbmodel.CircuitBreakerStateClosed,
			}
			if cbActive {
				acquire = cbManager.Acquire(breakerKey, time.Now(), cbCfg)
			}
			if cbActive && !acquire.Allowed {
				metrics.AddAttempt(dbmodel.ChannelAttempt{
					Round:            round,
					AttemptNum:       attemptNum,
					ChannelID:        channel.ID,
					ChannelName:      channel.Name,
					ModelName:        item.ModelName,
					Error:            string(acquire.Decision),
					RelayStatusCode:  http.StatusServiceUnavailable,
					RelayErrorSource: string(RelayErrorSourceLocal),
					RelayRetryable:   true,
					RelayTrippable:   false,
					BreakerKey:       breakerKey,
					CBDecision:       string(acquire.Decision),
					CBStateBefore:    string(acquire.StateBefore),
					CBStateAfter:     string(acquire.StateAfter),
					CBTripCount:      acquire.TripCount,
					CBOpenUntil:      formatTimeRFC3339(acquire.OpenUntil),
					ProbeInFlight:    acquire.ProbeInFlight,
				}, false, time.Since(attemptStart))
				lastRelayErr = newRelayError(http.StatusServiceUnavailable, RelayErrorSourceLocal, true, false, nil, string(acquire.Decision))
				lastStatusCode = http.StatusServiceUnavailable
				continue
			}

			rc := &relayContext{
				c:                    c,
				inAdapter:            inAdapter,
				outAdapter:           outAdapter,
				internalRequest:      internalRequest,
				channel:              channel,
				metrics:              metrics,
				usedKey:              channel.GetChannelKey(),
				firstTokenTimeOutSec: group.FirstTokenTimeOut,
			}

			statusCode, rerr := rc.forward()
			if rerr == nil {
				// 成功
				attemptDuration := time.Since(attemptStart)
				cbResult := breaker.RecordResult{
					Decision:   acquire.Decision,
					StateAfter: dbmodel.CircuitBreakerStateClosed,
				}
				if cbActive {
					cbResult = cbManager.RecordSuccess(breakerKey, acquire, cbCfg, time.Now())
				}
				rc.collectResponse()
				metrics.AddAttempt(dbmodel.ChannelAttempt{
					Round:           round,
					AttemptNum:      attemptNum,
					ChannelID:       channel.ID,
					ChannelName:     channel.Name,
					ModelName:       item.ModelName,
					RelayStatusCode: statusCode,
					BreakerKey:      breakerKey,
					CBDecision:      string(acquire.Decision),
					CBStateBefore:   string(acquire.StateBefore),
					CBStateAfter:    string(cbResult.StateAfter),
					CBTripCount:     cbResult.TripCount,
					CBOpenUntil:     formatTimeRFC3339(cbResult.OpenUntil),
					ProbeInFlight:   cbResult.ProbeInFlight,
				}, true, attemptDuration)
				rc.usedKey.StatusCode = statusCode
				rc.usedKey.LastUseTimeStamp = time.Now().Unix()
				rc.usedKey.TotalCost += metrics.Stats.InputCost + metrics.Stats.OutputCost
				op.ChannelKeyUpdate(rc.usedKey)
				metrics.Save(c.Request.Context(), true, nil, round)
				return
			}
			attemptDuration := time.Since(attemptStart)
			cbResult := breaker.RecordResult{
				Decision:   acquire.Decision,
				StateAfter: dbmodel.CircuitBreakerStateClosed,
			}
			if cbActive {
				cbResult = cbManager.RecordNonTrippable(breakerKey, acquire, cbCfg)
				if rerr.Trippable {
					cbResult = cbManager.RecordFailure(breakerKey, rerr.Error(), acquire, cbCfg, time.Now())
				}
			}

			metrics.AddAttempt(dbmodel.ChannelAttempt{
				Round:            round,
				AttemptNum:       attemptNum,
				ChannelID:        channel.ID,
				ChannelName:      channel.Name,
				ModelName:        item.ModelName,
				Error:            rerr.Error(),
				RelayStatusCode:  statusCode,
				RelayErrorSource: string(rerr.Source),
				RelayRetryable:   rerr.Retryable,
				RelayTrippable:   rerr.Trippable,
				BreakerKey:       breakerKey,
				CBDecision:       string(cbResult.Decision),
				CBStateBefore:    string(acquire.StateBefore),
				CBStateAfter:     string(cbResult.StateAfter),
				CBTripCount:      cbResult.TripCount,
				CBOpenUntil:      formatTimeRFC3339(cbResult.OpenUntil),
				ProbeInFlight:    cbResult.ProbeInFlight,
			}, false, attemptDuration)

			rc.usedKey.StatusCode = statusCode
			rc.usedKey.LastUseTimeStamp = time.Now().Unix()
			op.ChannelKeyUpdate(rc.usedKey)

			if c.Writer.Written() {
				// Streaming responses may have already started; retrying would corrupt the client stream.
				rc.collectResponse()
				metrics.Save(c.Request.Context(), false, rerr, 0)
				return
			}
			lastRelayErr = rerr
			if statusCode > 0 {
				lastStatusCode = statusCode
			} else if rerr.StatusCode > 0 {
				lastStatusCode = rerr.StatusCode
			}
			if !rerr.Retryable {
				metrics.Save(c.Request.Context(), false, rerr, 0)
				resp.Error(c, relayStatusCode(rerr, lastStatusCode), rerr.Error())
				return
			}

			nextItem, ok := b.Next(remainingItems, item)
			if !ok || nextItem == nil {
				break
			}
		}
	}

	// 所有通道都失败
	if lastRelayErr == nil {
		lastRelayErr = newRelayError(http.StatusBadGateway, RelayErrorSourceLocal, true, false, nil, "all channels failed")
	}
	metrics.Save(c.Request.Context(), false, lastRelayErr, 0)
	resp.Error(c, relayStatusCode(lastRelayErr, lastStatusCode), "all channels failed")
}

// parseRequest 解析并验证入站请求
func parseRequest(inboundType inbound.InboundType, c *gin.Context) (*tmodel.InternalLLMRequest, tmodel.Inbound, error) {
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

	return internalRequest, inAdapter, nil
}

// forward 转发请求到上游服务
func (rc *relayContext) forward() (int, *RelayError) {
	ctx := rc.c.Request.Context()

	// 构建出站请求
	outboundRequest, err := rc.outAdapter.TransformRequest(
		ctx,
		rc.internalRequest,
		rc.channel.GetBaseUrl(),
		rc.usedKey.ChannelKey,
	)
	if err != nil {
		log.Warnf("failed to create request: %v", err)
		return 0, newRelayError(http.StatusBadRequest, RelayErrorSourceLocal, false, false, err, fmt.Sprintf("failed to create request: %v", err))
	}

	// 复制请求头
	rc.copyHeaders(outboundRequest)

	// 发送请求
	response, err := rc.sendRequest(outboundRequest)
	if err != nil {
		return 0, classifyTransportError(fmt.Errorf("failed to send request: %w", err))
	}
	defer response.Body.Close()

	// 检查响应状态
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		if err != nil {
			return response.StatusCode, classifyUpstreamStatus(response.StatusCode, "")
		}
		return response.StatusCode, classifyUpstreamStatus(response.StatusCode, string(body))
	}

	// 处理响应
	if rc.internalRequest.Stream != nil && *rc.internalRequest.Stream {
		if err := rc.handleStreamResponse(ctx, response); err != nil {
			if rc.c.Writer.Written() {
				return response.StatusCode, newRelayError(response.StatusCode, RelayErrorSourceNetwork, false, false, err, err.Error())
			}
			return response.StatusCode, classifyTransportError(err)
		}
		return response.StatusCode, nil
	}
	if err := rc.handleResponse(ctx, response); err != nil {
		return response.StatusCode, newRelayError(response.StatusCode, RelayErrorSourceLocal, false, false, err, err.Error())
	}
	return response.StatusCode, nil
}

// copyHeaders 复制请求头，过滤 hop-by-hop 头
func (rc *relayContext) copyHeaders(outboundRequest *http.Request) {
	for key, values := range rc.c.Request.Header {
		if hopByHopHeaders[strings.ToLower(key)] {
			continue
		}
		for _, value := range values {
			outboundRequest.Header.Set(key, value)
		}
	}
	if len(rc.channel.CustomHeader) > 0 {
		for _, header := range rc.channel.CustomHeader {
			outboundRequest.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}
}

// sendRequest 发送 HTTP 请求
func (rc *relayContext) sendRequest(req *http.Request) (*http.Response, error) {
	httpClient, err := helper.ChannelHttpClient(rc.channel)
	if err != nil {
		log.Warnf("failed to get http client: %v", err)
		return nil, err
	}

	response, err := httpClient.Do(req)
	if err != nil {
		log.Warnf("failed to send request: %v", err)
		return nil, err
	}

	return response, nil
}

// handleStreamResponse 处理流式响应
func (rc *relayContext) handleStreamResponse(ctx context.Context, response *http.Response) error {
	// 流式响应应当是 SSE
	// 某些上游可能会返回非SSE的JSON响应 (由于 Accept headers 配置错误)
	if ct := response.Header.Get("Content-Type"); ct != "" && !strings.Contains(strings.ToLower(ct), "text/event-stream") {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		return fmt.Errorf("upstream returned non-SSE content-type %q for stream request: %s", ct, string(body))
	}

	// 设置 SSE 响应头
	rc.c.Header("Content-Type", "text/event-stream")
	rc.c.Header("Cache-Control", "no-cache")
	rc.c.Header("Connection", "keep-alive")
	rc.c.Header("X-Accel-Buffering", "no")

	firstToken := true

	// Streaming "time to first token" timeout: only applies before we write anything to the client.
	// We read SSE events in a goroutine so we can race the first meaningful output against a timer.
	type sseReadResult struct {
		data string
		err  error
	}
	results := make(chan sseReadResult, 1)
	go func() {
		defer close(results)
		readCfg := &sse.ReadConfig{MaxEventSize: maxSSEEventSize}
		for ev, err := range sse.Read(response.Body, readCfg) {
			if err != nil {
				results <- sseReadResult{err: err}
				return
			}
			results <- sseReadResult{data: ev.Data}
		}
	}()

	var firstTokenTimer *time.Timer
	var firstTokenC <-chan time.Time
	if firstToken && rc.firstTokenTimeOutSec > 0 {
		firstTokenTimer = time.NewTimer(time.Duration(rc.firstTokenTimeOutSec) * time.Second)
		firstTokenC = firstTokenTimer.C
		defer func() {
			if firstTokenTimer != nil {
				firstTokenTimer.Stop()
			}
		}()
	}

	for {
		// 检查客户端是否断开
		select {
		case <-ctx.Done():
			log.Infof("client disconnected, stopping stream")
			return nil
		case <-firstTokenC:
			// Abort upstream stream before any client writes; caller will retry next channel.
			log.Warnf("first token timeout (%ds), switching channel", rc.firstTokenTimeOutSec)
			_ = response.Body.Close()
			return fmt.Errorf("first token timeout (%ds)", rc.firstTokenTimeOutSec)
		case r, ok := <-results:
			if !ok {
				log.Infof("stream end")
				return nil
			}
			if r.err != nil {
				log.Warnf("failed to read event: %v", r.err)
				return fmt.Errorf("failed to read stream event: %w", r.err)
			}

			// 转换流式数据
			data, err := rc.transformStreamData(ctx, r.data)
			if err != nil || len(data) == 0 {
				continue
			}
			// 记录首个 Token 时间
			if firstToken {
				rc.metrics.SetFirstTokenTime(time.Now())
				firstToken = false
				// Disable the first-token timer once we have meaningful output.
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

			rc.c.Writer.Write(data)
			rc.c.Writer.Flush()
		}
	}
}

// transformStreamData 转换流式数据
func (rc *relayContext) transformStreamData(ctx context.Context, data string) ([]byte, error) {
	// 上游格式 → 内部格式
	internalStream, err := rc.outAdapter.TransformStream(ctx, []byte(data))
	if err != nil {
		log.Warnf("failed to transform stream: %v", err)
		return nil, err
	}
	if internalStream == nil {
		return nil, nil
	}

	// 内部格式 → 入站格式
	inStream, err := rc.inAdapter.TransformStream(ctx, internalStream)
	if err != nil {
		log.Warnf("failed to transform stream: %v", err)
		return nil, err
	}

	return inStream, nil
}

// handleResponse 处理非流式响应
func (rc *relayContext) handleResponse(ctx context.Context, response *http.Response) error {
	// 上游格式 → 内部格式
	internalResponse, err := rc.outAdapter.TransformResponse(ctx, response)
	if err != nil {
		log.Warnf("failed to transform response: %v", err)
		return fmt.Errorf("failed to transform outbound response: %w", err)
	}

	// 内部格式 → 入站格式
	inResponse, err := rc.inAdapter.TransformResponse(ctx, internalResponse)
	if err != nil {
		log.Warnf("failed to transform response: %v", err)
		return fmt.Errorf("failed to transform inbound response: %w", err)
	}

	rc.c.Data(http.StatusOK, "application/json", inResponse)
	return nil
}

// collectResponse 收集响应信息
func (rc *relayContext) collectResponse() {
	internalResponse, err := rc.inAdapter.GetInternalResponse(rc.c.Request.Context())
	if err != nil || internalResponse == nil {
		return
	}

	// 设置响应内容
	rc.metrics.SetInternalResponse(internalResponse)
}

func cloneGroupItems(items []dbmodel.GroupItem) []dbmodel.GroupItem {
	out := make([]dbmodel.GroupItem, len(items))
	copy(out, items)
	return out
}

func removeGroupItem(items []dbmodel.GroupItem, itemID int) []dbmodel.GroupItem {
	if len(items) == 0 {
		return items
	}
	out := make([]dbmodel.GroupItem, 0, len(items)-1)
	removed := false
	for _, item := range items {
		if !removed && item.ID == itemID {
			removed = true
			continue
		}
		out = append(out, item)
	}
	if !removed {
		return items
	}
	return out
}

func buildAllOpenData(groupID int, modelName string, earliestRetryAt, now time.Time) dbmodel.CircuitBreakerAllOpenData {
	if earliestRetryAt.IsZero() || earliestRetryAt.Before(now) {
		earliestRetryAt = now.Add(1 * time.Second)
	}
	retryAfter := int(math.Ceil(earliestRetryAt.Sub(now).Seconds()))
	if retryAfter < 1 {
		retryAfter = 1
	}
	return dbmodel.CircuitBreakerAllOpenData{
		Reason:            "all_open",
		EarliestRetryAt:   earliestRetryAt.UTC().Format(time.RFC3339Nano),
		RetryAfterSeconds: retryAfter,
		Scope: dbmodel.CircuitBreakerAllOpenScope{
			GroupID:   groupID,
			ModelName: modelName,
		},
	}
}

func relayStatusCode(rerr *RelayError, fallback int) int {
	if rerr != nil && rerr.StatusCode > 0 {
		return rerr.StatusCode
	}
	if fallback > 0 {
		return fallback
	}
	return http.StatusBadGateway
}

func formatTimeRFC3339(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339Nano)
}
