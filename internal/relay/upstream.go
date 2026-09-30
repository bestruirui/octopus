package relay

import (
	"context"
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
	"github.com/tidwall/sjson"
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
func sendPassthrough(ctx context.Context, format llm.APIFormat, raw *httpclient.Request, channel model.Channel, outbound transformer.Outbound, streaming bool, modelName string) (*upstreamResponse, error) {
	request, err := buildPassthroughRequest(format, raw, channel, outbound, modelName)
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

// sendPassthroughStream 发起同协议流式请求并预读首个有效事件, 首个事件通过验证才算本轮取得可提交响应。
func sendPassthroughStream(ctx context.Context, format llm.APIFormat, request *httpclient.Request, client *http.Client) (*upstreamResponse, error) {
	rawRequest, err := httpclient.BuildHttpRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	// 客户端的 Accept 属于库自管头不会透传, 需显式声明才能让上游按 SSE 返回。
	rawRequest.Header.Set("Accept", "text/event-stream")

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
		return &upstreamResponse{header: response.Header.Clone(), events: events, first: event, last: last}, nil
	}

	err = events.Err()
	events.Close()
	if err == nil {
		err = errors.New("upstream stream ended before first event")
	}
	return nil, err
}

// conversionMiddleware 保存跨协议 pipeline 单次调用需要应用和取得的状态。
type conversionMiddleware struct {
	pipeline.DummyMiddleware // 提供本次无需处理的其余 pipeline 中间件方法。
	channel model.Channel // 本轮上游请求使用的渠道配置。
	format  llm.APIFormat // 上游渠道协议, 用于校验统一响应终态。
	rawBody []byte        // 上游非流式响应或错误的原始正文。
	usage   *llm.Usage    // 非流式统一响应中确认的用量。
}

// OnOutboundRawRequest 在转换后的上游请求上应用渠道参数和自定义 Header。
func (m *conversionMiddleware) OnOutboundRawRequest(_ context.Context, request *httpclient.Request) (*httpclient.Request, error) {
	normalizeDeveloperRole(m.format, request)
	return request, applyChannelConfig(m.channel, request)
}

// normalizeDeveloperRole 把转换后 chat completions 请求里 messages 的 developer 角色归一化为 system。
// developer 是 OpenAI Responses 协议对 system 的替代角色, 客户端 (如 Codex) 会用它携带指令;
// 部分 OpenAI 兼容上游不接受该角色而直接报错, 归一化后语义等价且兼容性更好。
func normalizeDeveloperRole(format llm.APIFormat, request *httpclient.Request) {
	if format != llm.APIFormatOpenAIChatCompletion {
		return
	}
	messages := gjson.GetBytes(request.Body, "messages")
	if !messages.IsArray() {
		return
	}
	for i, message := range messages.Array() {
		if message.Get("role").String() != "developer" {
			continue
		}
		body, err := sjson.SetBytes(request.Body, fmt.Sprintf("messages.%d.role", i), "system")
		if err != nil {
			return
		}
		request.Body = body
	}
	if len(request.JSONBody) > 0 {
		request.JSONBody = slices.Clone(request.Body)
	}
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

// OnOutboundLlmResponse 取得非流式用量并在回转客户端协议前校验上游终态。
func (m *conversionMiddleware) OnOutboundLlmResponse(_ context.Context, response *llm.Response) (*llm.Response, error) {
	if err := validateResponse(m.format, response); err != nil {
		return nil, err
	}
	m.usage = response.Usage
	return response, nil
}

// sendConverted 经 axonhub pipeline 把客户端请求转换成渠道协议后请求上游, 响应再转换回客户端协议。
func sendConverted(ctx context.Context, format llm.APIFormat, raw *httpclient.Request, channel model.Channel, outbound transformer.Outbound, streaming bool) (*upstreamResponse, error) {
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
	// 流式响应要等消费方读完才能归还连接, 故只在提交流式结果那一处移交, 其余出口一律就地归还。
	committed := false
	if closeIdle != nil {
		defer func() {
			if !committed {
				closeIdle()
			}
		}()
	}
	middleware := &conversionMiddleware{channel: channel, format: outbound.APIFormat()}
	processor := pipeline.NewFactory(httpclient.NewHttpClientWithClient(httpClient)).Pipeline(
		inbound,
		outbound,
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
		committed = true
		return &upstreamResponse{events: events, first: event, last: last, closeIdle: closeIdle}, nil
	}

	err = events.Err()
	events.Close()
	if err == nil {
		err = errors.New("upstream stream ended before first event")
	}
	return nil, err
}
