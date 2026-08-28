package relay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
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
}

// sendPassthrough 以同协议透传方式请求上游, 取得的响应无需转换即可回给客户端。
func sendPassthrough(ctx context.Context, format llm.APIFormat, raw *httpclient.Request, channel model.Channel, outbound transformer.Outbound, streaming bool) (*upstreamResponse, error) {
	request, err := buildPassthroughRequest(format, raw, channel)
	if err != nil {
		return nil, err
	}
	client, err := helper.ChannelHttpClient(&channel)
	if err != nil {
		return nil, err
	}
	if streaming {
		return sendPassthroughStream(ctx, format, request, client)
	}

	response, err := httpclient.NewHttpClientWithClient(client).Do(ctx, request)
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

	client, err := helper.ChannelHttpClient(&channel)
	if err != nil {
		return nil, err
	}
	middleware := &conversionMiddleware{channel: channel, format: outbound.APIFormat()}
	processor := pipeline.NewFactory(httpclient.NewHttpClientWithClient(client)).Pipeline(
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
		return &upstreamResponse{events: events, first: event, last: last}, nil
	}

	err = events.Err()
	events.Close()
	if err == nil {
		err = errors.New("upstream stream ended before first event")
	}
	return nil, err
}
