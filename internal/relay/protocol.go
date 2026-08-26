package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/anthropic"
	"github.com/looplj/axonhub/llm/transformer/openai"
	"github.com/looplj/axonhub/llm/transformer/openai/responses"
)

// upstreamPath 返回客户端协议在标准上游中对应的请求路径。
func upstreamPath(format llm.APIFormat) string {
	switch format {
	case llm.APIFormatOpenAIResponse:
		return "/responses"
	case llm.APIFormatAnthropicMessage:
		return "/messages"
	default:
		return "/chat/completions"
	}
}

// buildPassthroughRequest 构造同协议透传的上游请求: 目标地址, 认证和请求体由渠道决定, 客户端的其余请求头和查询参数
// 经 MergeInboundRequest 透传给上游, 其中认证类, 库自管类和逐跳类请求头会被丢弃以免覆盖渠道凭据。
func buildPassthroughRequest(format llm.APIFormat, raw *httpclient.Request, channel model.Channel) (*httpclient.Request, error) {
	// BaseURL 以 ## 结尾表示地址已完整, 不再追加版本号和协议路径。
	base := strings.TrimSuffix(channel.BaseURL, "##")
	url := transformer.BuildRequestURL(base, "v1", upstreamPath(format), "", base != channel.BaseURL)

	auth := &httpclient.AuthConfig{Type: httpclient.AuthTypeBearer, APIKey: channel.Key}
	if format == llm.APIFormatAnthropicMessage {
		auth = &httpclient.AuthConfig{Type: httpclient.AuthTypeAPIKey, APIKey: channel.Key, HeaderKey: "X-API-Key"}
	}
	// Content-Type 属于库自管头, 不会随客户端请求透传, 需按客户端原值显式重建。
	contentType := raw.Headers.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	request := httpclient.MergeInboundRequest(&httpclient.Request{
		Method:    raw.Method,
		URL:       url,
		Headers:   http.Header{"Content-Type": []string{contentType}},
		Body:      raw.Body,
		Auth:      auth,
		APIFormat: format.String(),
	}, raw)
	request, err := httpclient.FinalizeAuthHeaders(request)
	if err != nil {
		return nil, err
	}
	if err := applyChannelConfig(channel, request); err != nil {
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

// streamEventHasText 判断客户端协议事件是否携带实际可见文本; 角色、ID、用量和结束事件不算首字。
func streamEventHasText(format llm.APIFormat, event *httpclient.StreamEvent) bool {
	if event == nil || len(event.Data) == 0 {
		return false
	}

	switch format {
	case llm.APIFormatOpenAIChatCompletion:
		var parsed struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal(event.Data, &parsed) != nil {
			return false
		}
		for _, choice := range parsed.Choices {
			if choice.Delta.Content != "" {
				return true
			}
		}
	case llm.APIFormatOpenAIResponse:
		var parsed struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
		}
		return json.Unmarshal(event.Data, &parsed) == nil && parsed.Type == "response.output_text.delta" && parsed.Delta != ""
	case llm.APIFormatAnthropicMessage:
		var parsed struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		return json.Unmarshal(event.Data, &parsed) == nil && parsed.Type == "content_block_delta" && parsed.Delta.Type == "text_delta" && parsed.Delta.Text != ""
	}
	return false
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
