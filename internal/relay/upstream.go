package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/rhttp"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/pipeline"
	"github.com/looplj/axonhub/llm/streams"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/anthropic"
	"github.com/looplj/axonhub/llm/transformer/openai"
	"github.com/looplj/axonhub/llm/transformer/openai/responses"
	"github.com/tidwall/gjson"
)

// upstreamResponse 是已验证但尚未写给客户端的上游成功响应; events 为 nil 表示非流式响应。
// 透传响应保留上游响应头; 跨协议响应由客户端协议决定响应头。失败一律以 error 返回。
type upstreamResponse struct {
	body   []byte                                  // 非流式响应的完整正文。
	header http.Header                             // 同协议透传时需要原样返回的上游响应头。
	events streams.Stream[*httpclient.StreamEvent] // 流式响应中首个事件之后的剩余事件。
	first  *httpclient.StreamEvent                 // 已预读并验证的首个事件。
	last   bool                                    // 首个事件已经终止整个响应流。
	usage  *llm.Usage                              // 上游本次可确认的用量。
	// closeIdle 非 nil 时为渠道专用代理独占客户端的空闲连接归还入口, 消费方读完事件流后必须调用。
	// 仅流式响应会带上它: 非流式响应返回时连接已经用完, 由发起方就地归还。
	closeIdle func()
}

// resolveUpstreamClient 按渠道代理配置取得本轮上游请求使用的客户端。
// 第二个返回值非 nil 时说明客户端是为渠道专用代理新建的独占实例, 不进共享连接池, 用完须调用它归还空闲连接。
func resolveUpstreamClient(channel model.Channel) (*http.Client, func(), error) {
	switch {
	case !channel.Proxy:
		client, err := rhttp.Direct()
		return client, nil, err
	case channel.ChannelProxy == "":
		client, err := rhttp.Proxy()
		return client, nil, err
	default:
		client, err := rhttp.New(channel.ChannelProxy)
		if err != nil {
			return nil, nil, err
		}
		return client, client.CloseIdleConnections, nil
	}
}

// sendPassthrough 以同协议透传方式请求上游, 取得的响应无需转换即可回给客户端。
// oc 提供参数覆盖与自定义 Header 条件渲染所需的请求上下文。
func sendPassthrough(ctx context.Context, format llm.APIFormat, raw *httpclient.Request, channel model.Channel, outbound transformer.Outbound, streaming bool, oc *overrideContext) (*upstreamResponse, error) {
	request, err := buildPassthroughRequest(format, raw, channel, outbound, oc)
	if err != nil {
		return nil, err
	}
	httpClient, closeIdle, err := resolveUpstreamClient(channel)
	if err != nil {
		return nil, err
	}
	if streaming {
		// 流式响应返回后仍在读上游连接, 归还入口随响应交给消费方; 取不到响应时就地归还。
		result, err := sendPassthroughStream(ctx, format, request, httpClient)
		if err != nil {
			if closeIdle != nil {
				closeIdle()
			}
			return nil, err
		}
		result.closeIdle = closeIdle
		return result, nil
	}
	// 非流式响应在 Do 返回时正文已读完, 连接可以立即归还。
	if closeIdle != nil {
		defer closeIdle()
	}

	response, err := httpclient.NewHttpClientWithClient(httpClient).Do(ctx, request)
	if err != nil {
		var failure *httpclient.Error
		if errors.As(err, &failure) && len(failure.Body) > 0 {
			return nil, fmt.Errorf("%w: %s", err, failure.Body)
		}
		return nil, err
	}
	// 同协议下响应可原样回给客户端, 仍需解析一次以取得用量并识别以 200 下发的失败终态。
	parsed, err := outbound.TransformResponse(ctx, response)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, response.Body)
	}
	if err := validateResponse(format, parsed); err != nil {
		return nil, fmt.Errorf("%w: %s", err, response.Body)
	}
	return &upstreamResponse{body: slices.Clone(response.Body), header: response.Headers.Clone(), usage: parsed.Usage}, nil
}

// sendPassthroughStream 发起同协议流式请求; 空启动事件暂存到输出开始或成功结束, 期间失败仍可重试。
func sendPassthroughStream(ctx context.Context, format llm.APIFormat, request *httpclient.Request, client *http.Client) (*upstreamResponse, error) {
	rawRequest, err := httpclient.BuildHttpRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	// 客户端的 Accept 属于库自管头不会透传, 需显式声明才能让上游按 SSE 返回。
	rawRequest.Header.Set("Accept", "text/event-stream")
	rawRequest.Header.Set("Cache-Control", "no-cache")
	rawRequest.Header.Set("Connection", "keep-alive")

	response, err := client.Do(rawRequest)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusBadRequest {
		failure, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		return nil, fmt.Errorf("upstream responded %s: %s", response.Status, failure)
	}

	events := httpclient.NewDefaultSSEDecoder(ctx, response.Body)
	var buffered []*httpclient.StreamEvent // 尚未提交的原始启动事件, 确认输出后按原顺序回放。
	for events.Next() {
		event := events.Current()
		if event == nil || len(event.Data) == 0 {
			continue
		}
		last, err := inspectStreamEvent(format, event)
		if err != nil {
			events.Close()
			return nil, fmt.Errorf("%w: %s", err, event.Data)
		}
		startup := false // 仅确认不含输出的启动事件继续暂存, 其余事件保持透传。
		if !last {
			switch format {
			case llm.APIFormatOpenAIChatCompletion:
				parsed := gjson.ParseBytes(event.Data)
				startup = parsed.Get("choices").IsArray() && len(parsed.Get("citations").Array()) == 0
				for _, choice := range parsed.Get("choices").Array() {
					if !startup || choice.Get("finish_reason").String() != "" || choice.Get("message").Type != gjson.Null ||
						len(choice.Get("logprobs.content").Array()) > 0 || len(choice.Get("logprobs.refusal").Array()) > 0 {
						startup = false
						break
					}
					// role 仅为启动信息; 其他非空增量均提交, 保留工具, 音频及扩展字段的透传。
					choice.Get("delta").ForEach(func(key, value gjson.Result) bool {
						if key.Str == "role" {
							return true
						}
						startup = value.Type == gjson.Null || (value.Type == gjson.String && value.Str == "") ||
							(value.IsArray() && len(value.Array()) == 0)
						return startup
					})
					if !startup {
						break
					}
				}
			case llm.APIFormatOpenAIResponse:
				var parsed responses.StreamEvent
				if err := json.Unmarshal(event.Data, &parsed); err != nil {
					events.Close()
					return nil, fmt.Errorf("decode responses stream event: %w", err)
				}
				switch parsed.Type {
				case responses.StreamEventTypeResponseCreated, responses.StreamEventTypeResponseInProgress, responses.StreamEventTypeResponseQueued:
					startup = parsed.Response == nil || len(parsed.Response.Output) == 0
				case responses.StreamEventTypeOutputItemAdded:
					// 工具调用从 item.added 起即属于输出; 这里只暂存空的消息和推理占位。
					if parsed.Item != nil && (parsed.Item.Type == "message" || parsed.Item.Type == "reasoning") {
						startup = (parsed.Item.Content == nil ||
							((parsed.Item.Content.Text == nil || *parsed.Item.Content.Text == "") && len(parsed.Item.Content.Items) == 0)) &&
							len(parsed.Item.Summary) == 0 && parsed.Item.ReasoningContent == nil &&
							(parsed.Item.EncryptedContent == nil || *parsed.Item.EncryptedContent == "")
					}
				case responses.StreamEventTypeContentPartAdded:
					startup = parsed.Part != nil &&
						(parsed.Part.Type == "output_text" || parsed.Part.Type == "reasoning_text" || parsed.Part.Type == "refusal") &&
						parsed.Part.Text == "" && (parsed.Part.Refusal == nil || *parsed.Part.Refusal == "")
				case responses.StreamEventTypeReasoningSummaryPartAdded:
					startup = parsed.Part != nil && parsed.Part.Text == ""
				case responses.StreamEventTypeOutputTextDelta, responses.StreamEventTypeReasoningSummaryTextDelta, responses.StreamEventTypeReasoningTextDelta, "response.refusal.delta":
					startup = parsed.Delta == ""
				}
			case llm.APIFormatAnthropicMessage:
				var parsed anthropic.StreamEvent
				if err := json.Unmarshal(event.Data, &parsed); err != nil {
					events.Close()
					return nil, fmt.Errorf("decode anthropic stream event: %w", err)
				}
				switch parsed.Type {
				case "message_start":
					startup = parsed.Message == nil || len(parsed.Message.Content) == 0
				case "ping", "content_block_stop":
					startup = true
				case "content_block_start":
					// 工具和其他内容块立即提交, 仅空文本或空思考块属于启动占位。
					if parsed.ContentBlock != nil && (parsed.ContentBlock.Type == "text" || parsed.ContentBlock.Type == "thinking") {
						startup = (parsed.ContentBlock.Text == nil || *parsed.ContentBlock.Text == "") &&
							(parsed.ContentBlock.Thinking == nil || *parsed.ContentBlock.Thinking == "") &&
							(parsed.ContentBlock.Signature == nil || *parsed.ContentBlock.Signature == "") && len(parsed.ContentBlock.Citations) == 0
					}
				case "content_block_delta":
					if parsed.Delta != nil && parsed.Delta.Type != nil {
						switch *parsed.Delta.Type {
						case "text_delta":
							startup = parsed.Delta.Text == nil || *parsed.Delta.Text == ""
						case "thinking_delta":
							startup = parsed.Delta.Thinking == nil || *parsed.Delta.Thinking == ""
						case "signature_delta":
							startup = parsed.Delta.Signature == nil || *parsed.Delta.Signature == ""
						case "input_json_delta":
							startup = parsed.Delta.PartialJSON == nil || *parsed.Delta.PartialJSON == ""
						}
					}
				case "message_delta":
					startup = parsed.Delta == nil || parsed.Delta.StopReason == nil || *parsed.Delta.StopReason == ""
				}
			}
		}
		if startup {
			buffered = append(buffered, event)
			continue
		}
		if len(buffered) > 0 {
			buffered = append(buffered, event)
			return &upstreamResponse{header: response.Header.Clone(), events: streams.PrependStream(events, buffered[1:]...), first: buffered[0]}, nil
		}
		return &upstreamResponse{header: response.Header.Clone(), events: events, first: event, last: last}, nil
	}

	err = events.Err()
	events.Close()
	if err == nil {
		err = errors.New("upstream stream ended before first event")
		if len(buffered) > 0 {
			err = llm.ErrStreamIncomplete
		}
	}
	return nil, err
}

// conversionMiddleware 保存跨协议 pipeline 单次调用需要应用和取得的状态。
type conversionMiddleware struct {
	pipeline.DummyMiddleware                  // 提供本次无需处理的其余 pipeline 中间件方法。
	channel                  model.Channel    // 本轮上游请求使用的渠道配置。
	format                   llm.APIFormat    // 上游渠道协议, 用于校验统一响应终态。
	userAgent                string           // 客户端原始 User-Agent, 转换后的请求不保留它, 需在出站请求上重建。
	oc                       *overrideContext // 参数覆盖与自定义 Header 条件渲染所需的请求上下文。
	rawBody                  []byte           // 上游非流式响应或错误的原始正文。
	usage                    *llm.Usage       // 非流式统一响应中确认的用量。
	committed                bool             // 流式结果已移交消费方, 停止预读拦截并由消费方归还连接。
}

// OnOutboundRawRequest 在转换后的上游请求上重建客户端 User-Agent, 并应用渠道参数和自定义 Header。
func (m *conversionMiddleware) OnOutboundRawRequest(_ context.Context, request *httpclient.Request) (*httpclient.Request, error) {
	if m.userAgent != "" {
		request.Headers.Set("User-Agent", m.userAgent)
	}
	return request, applyChannelConfig(m.channel, m.oc, request)
}

// OnOutboundRawError 保留上游错误状态码携带的原始正文。
func (m *conversionMiddleware) OnOutboundRawError(_ context.Context, err error) {
	var failure *httpclient.Error
	if errors.As(err, &failure) {
		m.rawBody = slices.Clone(failure.Body)
	}
}

// OnOutboundRawResponse 保留上游成功响应的原始正文, 供后续转换或终态校验失败时诊断。
func (m *conversionMiddleware) OnOutboundRawResponse(_ context.Context, response *httpclient.Response) (*httpclient.Response, error) {
	m.rawBody = slices.Clone(response.Body)
	return response, nil
}

// OnOutboundRawStream 在预读期间拦截 Responses 的失败终态, 避免其被转换成普通结束原因。
func (m *conversionMiddleware) OnOutboundRawStream(_ context.Context, stream streams.Stream[*httpclient.StreamEvent]) (streams.Stream[*httpclient.StreamEvent], error) {
	if m.format != llm.APIFormatOpenAIResponse {
		return stream, nil
	}
	return streams.MapErr(stream, func(event *httpclient.StreamEvent) (*httpclient.StreamEvent, error) {
		if m.committed || event == nil {
			return event, nil
		}
		switch responses.StreamEventType(gjson.GetBytes(event.Data, "type").String()) {
		case responses.StreamEventTypeResponseCompleted, responses.StreamEventTypeResponseFailed, responses.StreamEventTypeResponseIncomplete, responses.StreamEventTypeResponseCancelled:
			if _, err := inspectStreamEvent(m.format, event); err != nil {
				return nil, fmt.Errorf("%w: %s", err, event.Data)
			}
		}
		return event, nil
	}), nil
}

// OnOutboundLlmResponse 取得非流式用量并在回转客户端协议前校验上游终态。
func (m *conversionMiddleware) OnOutboundLlmResponse(_ context.Context, response *llm.Response) (*llm.Response, error) {
	if err := validateResponse(m.format, response); err != nil {
		return nil, err
	}
	m.usage = response.Usage
	return response, nil
}

// sendConverted 经 axonhub pipeline 把客户端请求转换成渠道协议后请求上游, 响应再转换回客户端协议。
// oc 提供参数覆盖与自定义 Header 条件渲染所需的请求上下文。
func sendConverted(ctx context.Context, format llm.APIFormat, raw *httpclient.Request, channel model.Channel, outbound transformer.Outbound, streaming bool, oc *overrideContext) (*upstreamResponse, error) {
	var inbound transformer.Inbound
	switch format {
	case llm.APIFormatOpenAIResponse:
		inbound = responses.NewInboundTransformer()
	case llm.APIFormatAnthropicMessage:
		inbound = anthropic.NewInboundTransformer()
	default:
		inbound = openai.NewInboundTransformer()
	}

	httpClient, closeIdle, err := resolveUpstreamClient(channel)
	if err != nil {
		return nil, err
	}
	middleware := &conversionMiddleware{channel: channel, format: outbound.APIFormat(), userAgent: raw.Headers.Get("User-Agent"), oc: oc}
	// 流式响应要等消费方读完才能归还连接, 故只在提交流式结果那一处移交, 其余出口一律就地归还。
	if closeIdle != nil {
		defer func() {
			if !middleware.committed {
				closeIdle()
			}
		}()
	}
	processor := pipeline.NewFactory(httpclient.NewHttpClientWithClient(httpClient)).Pipeline(
		inbound,
		outbound,
		// 正重试预算启用库的输出前预读; 当前出站转换器未实现库的重试接口, 失败仍交回 Forward 重试。
		pipeline.WithRetry(1, 0, 0),
		pipeline.WithMiddlewares(middleware),
	)
	result, err := processor.Process(ctx, raw)
	if err != nil {
		if len(middleware.rawBody) > 0 {
			return nil, fmt.Errorf("%w: %s", err, middleware.rawBody)
		}
		return nil, err
	}
	if !streaming {
		return &upstreamResponse{body: slices.Clone(result.Response.Body), usage: middleware.usage}, nil
	}

	events := result.EventStream
	for events.Next() {
		event := events.Current()
		if event == nil || len(event.Data) == 0 {
			continue
		}
		last, err := inspectStreamEvent(format, event)
		if err != nil {
			events.Close()
			return nil, fmt.Errorf("%w: %s", err, event.Data)
		}
		middleware.committed = true
		return &upstreamResponse{events: events, first: event, last: last, closeIdle: closeIdle}, nil
	}

	err = events.Err()
	events.Close()
	if err == nil {
		err = errors.New("upstream stream ended before first event")
	}
	return nil, err
}
