package relay

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer/anthropic"
	"github.com/looplj/axonhub/llm/transformer/openai"
	"github.com/looplj/axonhub/llm/transformer/openai/responses"
)

var requestIDs atomic.Uint64 // requestIDs 分配进程内严格递增的请求 ID。
var errNoActiveChannel = errors.New("no active channel") // errNoActiveChannel 表示分组尚未选择活动渠道。

// nonRetryableStatusCodes 表示重试无法改变结果的上游 HTTP 状态码。
// 一类是客户端侧请求错误（400/405/406/413/414/415/422/501），请求本身不合法，
// 等待或换渠道都无济于事；另一类是渠道凭证失效（401/403），只能通过修正渠道
// key 解决，重试同样无效。命中后立即透传给客户端，由客户端在修复后重发。
// 404/429/5xx 保持可重试：换渠道或渠道恢复后仍可能成功。
var nonRetryableStatusCodes = map[int]bool{
	http.StatusBadRequest:            true, // 400 请求体格式或语义错误。
	http.StatusUnauthorized:          true, // 401 渠道 key 无效。
	http.StatusForbidden:             true, // 403 渠道 key 无权限或被封禁。
	http.StatusMethodNotAllowed:      true, // 405 方法不支持。
	http.StatusNotAcceptable:         true, // 406 Accept 头不被接受。
	http.StatusRequestEntityTooLarge: true, // 413 请求体超限。
	http.StatusRequestURITooLong:     true, // 414 URI 超长。
	http.StatusUnsupportedMediaType:  true, // 415 Content-Type 不支持。
	http.StatusUnprocessableEntity:   true, // 422 请求体语义无法处理。
	http.StatusNotImplemented:        true, // 501 上游确实不支持该功能。
}

// upstreamErrorStatusCode 从错误链中提取上游 HTTP 状态码，非上游 HTTP 错误返回 false。
func upstreamErrorStatusCode(err error) (int, bool) {
	var llmErr *llm.ResponseError
	var httpErr *httpclient.Error
	if errors.As(err, &llmErr) && llmErr.StatusCode != 0 {
		return llmErr.StatusCode, true
	}
	if errors.As(err, &httpErr) && httpErr.StatusCode != 0 {
		return httpErr.StatusCode, true
	}
	return 0, false
}

// execution 保存单个客户端请求的全部可变执行状态。
type execution struct {
	ctx            *gin.Context       // 当前客户端请求上下文。
	protocol       *relayProtocol     // 客户端协议规则。
	request        relayRequest       // 原始请求和路由字段。
	log            LogRecord          // 持续更新的完整请求记录。
	requestMetrics model.StatsMetrics // requestMetrics 累积全部真实上游调用的用量和费用。
	attemptIndex   int                // 当前请求已经分配的最后尝试序号。
}

// HandleChatCompletions 处理 OpenAI Chat Completions 客户端请求。
func HandleChatCompletions(c *gin.Context) {
	(&execution{ctx: c, protocol: &relayProtocol{format: llm.APIFormatOpenAIChatCompletion, route: "/chat/completions", authType: httpclient.AuthTypeBearer, inbound: openai.NewInboundTransformer()}}).execute()
}

// HandleResponses 处理 OpenAI Responses 客户端请求。
func HandleResponses(c *gin.Context) {
	(&execution{ctx: c, protocol: &relayProtocol{format: llm.APIFormatOpenAIResponse, route: "/responses", authType: httpclient.AuthTypeBearer, inbound: responses.NewInboundTransformer()}}).execute()
}

// HandleMessages 处理 Anthropic Messages 客户端请求。
func HandleMessages(c *gin.Context) {
	(&execution{ctx: c, protocol: &relayProtocol{format: llm.APIFormatAnthropicMessage, route: "/messages", authType: httpclient.AuthTypeAPIKey, inbound: anthropic.NewInboundTransformer()}}).execute()
}

// execute 初始化请求，并在渠道未选择时等待、失败时重试，直至提交响应或客户端取消。
func (e *execution) execute() {
	e.log = LogRecord{LogOverview: LogOverview{ID: requestIDs.Add(1), State: RequestStateRunning, StartedAt: time.Now(), ClientProtocol: e.protocol.format}}
	ctx := e.ctx.Request.Context()
	raw, err := httpclient.ReadHTTPRequest(e.ctx.Request)
	if err != nil {
		e.emit(LogEventRequestStarted, nil)
		e.finish(RequestStateFailed, err, nil, nil)
		e.protocol.writeError(e.ctx, http.StatusBadRequest, err)
		return
	}
	e.log.RequestBody = string(raw.Body)
	parsed, err := e.protocol.inbound.TransformRequest(ctx, cloneRequest(raw, ctx))
	if err != nil {
		e.emit(LogEventRequestStarted, nil)
		e.finish(RequestStateFailed, err, nil, nil)
		e.protocol.writeError(e.ctx, http.StatusBadRequest, err)
		return
	}
	e.request = relayRequest{raw: raw, model: parsed.Model, stream: parsed.Stream != nil && *parsed.Stream}
	e.log.RequestModel = e.request.model
	e.log.Stream = e.request.stream
	e.emit(LogEventRequestStarted, nil)
	if supported := e.ctx.GetString("supported_models"); supported != "" {
		allowed := false
		for _, item := range strings.Split(supported, ",") {
			if strings.TrimSpace(item) == e.request.model {
				allowed = true
				break
			}
		}
		if !allowed {
			err = errors.New("model not supported")
			e.finish(RequestStateFailed, err, nil, nil)
			e.protocol.writeError(e.ctx, http.StatusBadRequest, err)
			return
		}
	}
	for {
		if err := ctx.Err(); err != nil {
			e.finish(RequestStateCanceled, err, nil, nil)
			return
		}

		activeItemChanged := op.GroupActiveItemChangeSignal()
		item, channel, retryInterval, retryErr := e.resolveTarget(ctx)
		if errors.Is(retryErr, errNoActiveChannel) {
			if e.log.Error != "" {
				e.log.Error = ""
				e.emit(LogEventTargetWaiting, nil)
			}
			select {
			case <-ctx.Done():
				e.finish(RequestStateCanceled, ctx.Err(), nil, nil)
				return
			case <-activeItemChanged:
			}
			continue
		}
		if retryErr != nil {
			e.recordUnavailableTarget(item, channel, retryErr)
		} else {
			done, attemptErr := e.executeAttempt(ctx, item, channel)
			if done {
				return
			}
			retryErr = attemptErr
		}

		if retryErr == nil {
			continue
		}
		select {
		case <-ctx.Done():
			e.finish(RequestStateCanceled, ctx.Err(), nil, nil)
			return
		case <-time.After(time.Duration(retryInterval) * time.Second):
		}
	}
}

// resolveTarget 加载请求模型对应的分组并校验当前活动渠道，返回本轮分组项、渠道和重试间隔。
func (e *execution) resolveTarget(ctx context.Context) (model.GroupItem, *model.Channel, int, error) {
	retryInterval := 1
	group, err := op.GroupGetByName(e.request.model, ctx)
	if err != nil {
		return model.GroupItem{}, nil, retryInterval, errors.New("model not found")
	}
	if group.RetryInterval >= 1 {
		retryInterval = group.RetryInterval
	}

	var item model.GroupItem
	if group.ActiveItemID == 0 {
		return model.GroupItem{}, nil, retryInterval, errNoActiveChannel
	}
	for _, candidate := range group.Items {
		if candidate.ID == group.ActiveItemID {
			item = candidate
			break
		}
	}
	if item.ID == 0 {
		return model.GroupItem{}, nil, retryInterval, errors.New("active channel not found")
	}
	channel, err := op.ChannelGet(item.ChannelID, ctx)
	if err != nil {
		return item, nil, retryInterval, errors.New("active channel not found")
	}
	if !channel.Enabled {
		return item, channel, retryInterval, errors.New("active channel disabled")
	}
	if channel.Key == "" {
		return item, channel, retryInterval, errors.New("active channel has no available key")
	}
	return item, channel, retryInterval, nil
}

// executeAttempt 执行当前渠道的一次上游尝试，提交前失败时交回外层继续重试。
func (e *execution) executeAttempt(ctx context.Context, item model.GroupItem, channel *model.Channel) (bool, error) {
	client, err := helper.ChannelHttpClient(channel)
	if err != nil {
		e.recordUnavailableTarget(item, channel, err)
		return false, err
	}
	attemptCtx, cancelAttempt := context.WithCancel(ctx)
	defer cancelAttempt()
	startedAt := time.Now()
	attempt := e.startAttempt(item, channel, cancelAttempt)
	result := (&forwarder{protocol: e.protocol, request: &e.request, client: client}).executeUpstream(attemptCtx, item.ModelName, channel)
	interrupted := !clearAttempt(e.log.ID, attempt.Index)
	if interrupted {
		result.err = errors.New("relay attempt interrupted")
	}
	if result.err != nil {
		return e.handleAttemptFailure(ctx, item, channel, attempt, result, time.Since(startedAt), interrupted)
	}

	e.commitAttempt(ctx, attemptCtx, item, channel, attempt, result, startedAt)
	return true, nil
}

// handleAttemptFailure 记录上游失败并决定当前请求是立即重试、等待重试还是结束。
func (e *execution) handleAttemptFailure(ctx context.Context, item model.GroupItem, channel *model.Channel, attempt *LogAttempt, result upstreamResult, duration time.Duration, interrupted bool) (bool, error) {
	if result.response != nil {
		result.response.Close()
	}
	attempt.Error = result.err.Error()
	if len(result.responseBody) > 0 {
		attempt.Error += ": " + string(result.responseBody)
	}
	if interrupted {
		e.log.Error = attempt.Error
		e.emit(LogEventAttemptFinished, attempt)
		return false, nil
	}
	if errors.Is(result.err, context.Canceled) || ctx.Err() != nil {
		e.log.Error = attempt.Error
		e.emit(LogEventAttemptFinished, attempt)
		cancelErr := result.err
		if ctx.Err() != nil {
			cancelErr = ctx.Err()
		}
		e.finish(RequestStateCanceled, cancelErr, nil, nil)
		return true, nil
	}
	if !errors.Is(result.err, errUnsupportedTarget) {
		metrics := usageMetrics(item.ModelName, result.usage)
		e.requestMetrics.Add(metrics)
		applyAttemptMetric(item, channel, metrics, duration, false)
	}
	e.log.Error = attempt.Error
	e.emit(LogEventAttemptFinished, attempt)
	// 客户端侧请求错误立即透传给客户端并结束请求，不再回到重试循环。
	if statusCode, ok := upstreamErrorStatusCode(result.err); ok && nonRetryableStatusCodes[statusCode] {
		e.finish(RequestStateFailed, result.err, result.responseBody, result.usage)
		e.protocol.writeError(e.ctx, statusCode, result.err)
		return true, nil
	}
	return false, result.err
}

// commitAttempt 将已验证的上游响应提交给客户端，并完成本次尝试和请求统计。
func (e *execution) commitAttempt(ctx, attemptCtx context.Context, item model.GroupItem, channel *model.Channel, attempt *LogAttempt, result upstreamResult, startedAt time.Time) {
	e.log.State = RequestStateCommitted
	e.emit(LogEventResponseCommitted, attempt)

	commit := result.response.Commit(attemptCtx, e.ctx)
	result.response.Close()
	usage := commit.usage
	if usage == nil {
		usage = result.usage
	}
	metricDuration := time.Since(startedAt)
	if !commit.firstWriteAt.IsZero() {
		metricDuration = commit.firstWriteAt.Sub(startedAt)
	}
	metrics := usageMetrics(item.ModelName, usage)
	e.requestMetrics.Add(metrics)
	applyAttemptMetric(item, channel, metrics, metricDuration, commit.err == nil || errors.Is(commit.err, errClientWrite) || errors.Is(commit.err, context.Canceled))
	if commit.err == nil {
		e.finish(RequestStateSuccess, nil, commit.responseBody, usage)
	} else if errors.Is(commit.err, context.Canceled) || ctx.Err() != nil {
		e.finish(RequestStateCanceled, commit.err, commit.responseBody, usage)
	} else {
		e.finish(RequestStateFailed, commit.err, commit.responseBody, usage)
	}
}

// newAttempt 分配新的尝试序号并写入请求的最终路由字段。
func (e *execution) newAttempt(channelName, modelName string) *LogAttempt {
	e.attemptIndex++
	attempt := &LogAttempt{Index: e.attemptIndex, ChannelName: channelName, ModelName: modelName}
	e.log.ActualModel = modelName
	e.log.FinalChannelName = channelName
	return attempt
}

// recordUnavailableTarget 记录当前活动目标不可用且未发出上游请求的尝试。
func (e *execution) recordUnavailableTarget(item model.GroupItem, channel *model.Channel, err error) {
	channelName := "Octopus"
	modelName := item.ModelName
	if modelName == "" {
		modelName = e.request.model
	}
	if channel != nil {
		channelName = channel.Name
	}
	attempt := e.newAttempt(channelName, modelName)
	attempt.Error = err.Error()
	e.log.Error = attempt.Error
	e.emit(LogEventAttemptFinished, attempt)
}

// startAttempt 登记可中止句柄，并发布一次真实上游请求的运行状态。
func (e *execution) startAttempt(item model.GroupItem, channel *model.Channel, cancel context.CancelFunc) *LogAttempt {
	attempt := e.newAttempt(channel.Name, item.ModelName)
	e.log.Error = ""
	setAttempt(e.log.ID, attempt.Index, cancel)
	e.emit(LogEventAttemptStarted, attempt)
	return attempt
}

// emit 向日志状态提交当前请求和本次尝试的独立快照。
func (e *execution) emit(eventType LogEventType, current *LogAttempt) {
	var attempt *LogAttempt
	if current != nil {
		copyAttempt := *current
		attempt = &copyAttempt
	}
	applyLog(eventType, e.log, attempt)
}

// finish 通过唯一完成入口补全日志和统计。
func (e *execution) finish(state RequestState, err error, responseBody []byte, usage *llm.Usage) {
	e.log.State = state
	e.log.CompletedAt = time.Now()
	e.log.Duration = e.log.CompletedAt.Sub(e.log.StartedAt)
	e.requestMetrics.WaitTime = e.log.Duration.Milliseconds()
	if state == RequestStateSuccess {
		e.log.Error = ""
	} else if err != nil {
		e.log.Error = err.Error()
	}
	if len(responseBody) > 0 {
		e.log.ResponseBody = string(responseBody)
	}
	if usage != nil {
		e.log.InputTokens = usage.PromptTokens
		e.log.OutputTokens = usage.CompletionTokens
		if usage.PromptTokensDetails != nil {
			e.log.CacheReadTokens = usage.PromptTokensDetails.CachedTokens
			e.log.CacheWriteTokens = usage.PromptTokensDetails.WriteCachedTokens
		}
	}
	e.log.TotalCost = e.requestMetrics.InputCost + e.requestMetrics.OutputCost
	applyCompletion(e.ctx.Request.Context(), e.ctx.GetInt("api_key_id"), state, e.requestMetrics)
	e.emit(LogEventRequestFinished, nil)
}
