package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/anthropic"
	"github.com/looplj/axonhub/llm/transformer/openai"
	"github.com/looplj/axonhub/llm/transformer/openai/responses"
)

// buildPassthroughRequest 构造同协议透传的上游请求: 目标地址, 认证和请求体由渠道决定, 客户端的其余请求头和查询参数
// 经 MergeInboundRequest 透传给上游, 其中认证类, 库自管类和逐跳类请求头会被丢弃以免覆盖渠道凭据。
// 地址与认证取自出站转换器对一个占位请求的转换结果, 使透传与跨协议转换共用同一套地址拼接, 避免两处规则分歧;
// openai 与 anthropic 出站转换器会校验模型名非空, 故占位请求必须带本轮真实的上游模型名。
// oc 提供参数覆盖与自定义 Header 条件渲染所需的请求上下文。
func buildPassthroughRequest(format llm.APIFormat, raw *httpclient.Request, channel model.Channel, outbound transformer.Outbound, oc *overrideContext) (*httpclient.Request, error) {
	probeText := "probe"
	probe, err := outbound.TransformRequest(context.Background(), &llm.Request{
		Model:    oc.Model,
		Messages: []llm.Message{{Role: "user", Content: llm.MessageContent{Content: &probeText}}},
	})
	if err != nil {
		return nil, fmt.Errorf("resolve upstream endpoint: %w", err)
	}

	// Content-Type 属于库自管头, 不会随客户端请求透传, 需按客户端原值显式重建。
	contentType := raw.Headers.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	request := httpclient.MergeInboundRequest(&httpclient.Request{
		Method:    raw.Method,
		URL:       probe.URL,
		Headers:   http.Header{"Content-Type": []string{contentType}},
		Body:      raw.Body,
		Auth:      probe.Auth,
		APIFormat: format.String(),
	}, raw)
	request, err = httpclient.FinalizeAuthHeaders(request)
	if err != nil {
		return nil, err
	}
	if err := applyChannelConfig(channel, oc, request); err != nil {
		return nil, err
	}
	return request, nil
}

// inspectStreamEvent 判断一个客户端协议流事件是否结束了整个响应流, 并识别以事件形式下发的上游错误。
// 返回 true 表示流已结束; 返回的 error 非空表示该事件本身即失败, 本轮不可提交。
func inspectStreamEvent(format llm.APIFormat, event *httpclient.StreamEvent) (bool, error) {
	if event == nil || len(event.Data) == 0 {
		return false, nil
	}

	switch format {
	case llm.APIFormatOpenAIChatCompletion:
		if bytes.Equal(event.Data, llm.DoneStreamEvent.Data) {
			return true, nil
		}
		var failure openai.OpenAIError
		if err := json.Unmarshal(event.Data, &failure); err != nil {
			return true, fmt.Errorf("decode openai stream event: %w", err)
		}
		if event.Type == "error" || failure.Detail.Message != "" || failure.Detail.Type != "" || failure.Detail.Code != "" {
			if failure.Detail.Message == "" {
				failure.Detail.Message = "openai stream error"
			}
			return true, &llm.ResponseError{Detail: failure.Detail}
		}
		return false, nil

	case llm.APIFormatOpenAIResponse:
		var parsed responses.StreamEvent
		if err := json.Unmarshal(event.Data, &parsed); err != nil {
			return true, fmt.Errorf("decode responses stream event: %w", err)
		}
		switch parsed.Type {
		case responses.StreamEventTypeResponseCompleted:
			// 已完成事件仍可能携带非 completed 的终态, 需按 status 与 error 区分成败。
			if parsed.Response == nil || parsed.Response.Status == nil || *parsed.Response.Status == "" || *parsed.Response.Status == "completed" {
				return true, nil
			}
			if parsed.Response.Error != nil {
				return true, &llm.ResponseError{Detail: llm.ErrorDetail{Code: parsed.Response.Error.Code, Message: parsed.Response.Error.Message, Type: parsed.Response.Error.Type}}
			}
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Message: "response " + *parsed.Response.Status, Type: "response_" + *parsed.Response.Status}}
		case responses.StreamEventTypeResponseFailed:
			if parsed.Response != nil && parsed.Response.Error != nil {
				return true, &llm.ResponseError{Detail: llm.ErrorDetail{Code: parsed.Response.Error.Code, Message: parsed.Response.Error.Message, Type: parsed.Response.Error.Type}}
			}
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Message: "response failed", Type: "response_failed"}}
		case responses.StreamEventTypeResponseIncomplete:
			message := "response incomplete"
			if parsed.Response != nil && parsed.Response.IncompleteDetails != nil && parsed.Response.IncompleteDetails.Reason != "" {
				message += ": " + parsed.Response.IncompleteDetails.Reason
			}
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Message: message, Type: "response_incomplete"}}
		case responses.StreamEventTypeResponseCancelled:
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Message: "response cancelled", Type: "response_cancelled"}}
		case responses.StreamEventTypeError:
			if parsed.Message == "" {
				parsed.Message = "responses stream error"
			}
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Code: parsed.Code, Message: parsed.Message, Type: "stream_error"}}
		default:
			return false, nil
		}

	case llm.APIFormatAnthropicMessage:
		var parsed anthropic.StreamEvent
		if err := json.Unmarshal(event.Data, &parsed); err != nil {
			return true, fmt.Errorf("decode anthropic stream event: %w", err)
		}
		if parsed.Type == "" {
			return true, errors.New("anthropic stream event type is empty")
		}
		// SSE 事件名与正文类型不一致说明流已错乱, 不能继续按协议解析。
		if event.Type != "" && event.Type != parsed.Type {
			return true, fmt.Errorf("anthropic stream event type mismatch: %s != %s", event.Type, parsed.Type)
		}
		if parsed.Type == "message_stop" {
			return true, nil
		}
		if parsed.Type == "error" {
			var failure anthropic.AnthropicError
			if err := json.Unmarshal(event.Data, &failure); err != nil {
				return true, fmt.Errorf("decode anthropic stream error: %w", err)
			}
			if failure.Error.Message == "" {
				failure.Error.Message = "anthropic stream error"
			}
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Message: failure.Error.Message, Type: failure.Error.Type, RequestID: failure.RequestID}}
		}
		return false, nil

	default:
		return false, nil
	}
}

// validateResponse 检查统一响应中需要在提交前判定为失败的终止原因; 仅 Responses 协议会以正常响应下发这类终态。
func validateResponse(format llm.APIFormat, response *llm.Response) error {
	if response == nil {
		return errors.New("upstream response is empty")
	}
	if format != llm.APIFormatOpenAIResponse || len(response.Choices) == 0 || response.Choices[0].FinishReason == nil {
		return nil
	}
	switch *response.Choices[0].FinishReason {
	case "error":
		return &llm.ResponseError{Detail: llm.ErrorDetail{Message: "response failed", Type: "response_failed"}}
	case "length":
		return &llm.ResponseError{Detail: llm.ErrorDetail{Message: "response incomplete", Type: "response_incomplete"}}
	case "cancelled":
		return &llm.ResponseError{Detail: llm.ErrorDetail{Message: "response cancelled", Type: "response_cancelled"}}
	default:
		return nil
	}
}
