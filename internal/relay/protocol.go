package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/anthropic"
	"github.com/looplj/axonhub/llm/transformer/openai"
	"github.com/looplj/axonhub/llm/transformer/openai/responses"
)

// relayProtocol 描述一个客户端协议的路由、认证和转换规则。
type relayProtocol struct {
	format      llm.APIFormat       // 客户端协议标识。
	route       string              // 客户端和标准上游路径。
	authType    string              // 同协议上游认证方式。
	inbound     transformer.Inbound // inbound 负责客户端协议转换。
	omitBodyLog bool                // omitBodyLog 避免将大型或二进制正文复制到内存日志。
}

// writeError 将内部错误转换成客户端协议响应。
func (p *relayProtocol) writeError(c *gin.Context, statusCode int, err error) {
	var llmErr *llm.ResponseError
	var httpErr *httpclient.Error
	if !errors.As(err, &llmErr) && !errors.As(err, &httpErr) &&
		!errors.Is(err, transformer.ErrInvalidRequest) && !errors.Is(err, transformer.ErrInvalidModel) {
		err = &llm.ResponseError{StatusCode: statusCode, Detail: llm.ErrorDetail{Message: err.Error(), Type: "api_error"}}
	}
	response := p.inbound.TransformError(c.Request.Context(), err)
	copyHeaders(c.Writer.Header(), response.Headers)
	contentType := c.Writer.Header().Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(response.StatusCode, contentType, response.Body)
	c.Abort()
}

// streamTerminal 判断客户端协议流是否结束，并返回异常终止原因。
func (p *relayProtocol) streamTerminal(event *httpclient.StreamEvent) (bool, error) {
	if event == nil {
		return false, nil
	}
	switch p.format {
	case llm.APIFormatOpenAIChatCompletion:
		if bytes.Equal(event.Data, llm.DoneStreamEvent.Data) {
			return true, nil
		}
		var streamError openai.OpenAIError
		if err := json.Unmarshal(event.Data, &streamError); err != nil {
			return true, fmt.Errorf("failed to decode OpenAI stream event: %w", err)
		}
		if event.Type == "error" || streamError.Detail.Message != "" || streamError.Detail.Type != "" || streamError.Detail.Code != "" {
			if streamError.Detail.Message == "" {
				streamError.Detail.Message = "OpenAI stream error"
			}
			return true, &llm.ResponseError{Detail: streamError.Detail}
		}
		return false, nil
	case llm.APIFormatOpenAIResponse:
		var streamEvent responses.StreamEvent
		if err := json.Unmarshal(event.Data, &streamEvent); err != nil {
			return true, fmt.Errorf("failed to decode Responses stream event: %w", err)
		}
		switch streamEvent.Type {
		case responses.StreamEventTypeResponseCompleted:
			if streamEvent.Response == nil || streamEvent.Response.Status == nil || *streamEvent.Response.Status == "" || *streamEvent.Response.Status == "completed" {
				return true, nil
			}
			if streamEvent.Response.Error != nil {
				return true, &llm.ResponseError{Detail: llm.ErrorDetail{Code: streamEvent.Response.Error.Code, Message: streamEvent.Response.Error.Message, Type: streamEvent.Response.Error.Type}}
			}
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Message: "response " + *streamEvent.Response.Status, Type: "response_" + *streamEvent.Response.Status}}
		case responses.StreamEventTypeResponseFailed:
			if streamEvent.Response != nil && streamEvent.Response.Error != nil {
				return true, &llm.ResponseError{Detail: llm.ErrorDetail{Code: streamEvent.Response.Error.Code, Message: streamEvent.Response.Error.Message, Type: streamEvent.Response.Error.Type}}
			}
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Message: "response failed", Type: "response_failed"}}
		case responses.StreamEventTypeResponseIncomplete:
			message := "response incomplete"
			if streamEvent.Response != nil && streamEvent.Response.IncompleteDetails != nil && streamEvent.Response.IncompleteDetails.Reason != "" {
				message += ": " + streamEvent.Response.IncompleteDetails.Reason
			}
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Message: message, Type: "response_incomplete"}}
		case responses.StreamEventTypeResponseCancelled:
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Message: "response cancelled", Type: "response_cancelled"}}
		case responses.StreamEventTypeError:
			if streamEvent.Message == "" {
				streamEvent.Message = "Responses stream error"
			}
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Code: streamEvent.Code, Message: streamEvent.Message, Type: "stream_error"}}
		default:
			return false, nil
		}
	case llm.APIFormatAnthropicMessage:
		var streamEvent anthropic.StreamEvent
		if err := json.Unmarshal(event.Data, &streamEvent); err != nil {
			return true, fmt.Errorf("failed to decode Anthropic stream event: %w", err)
		}
		if streamEvent.Type == "" {
			return true, errors.New("Anthropic stream event type is empty")
		}
		if event.Type != "" && event.Type != streamEvent.Type {
			return true, fmt.Errorf("Anthropic stream event type mismatch: %s != %s", event.Type, streamEvent.Type)
		}
		if streamEvent.Type == "message_stop" {
			return true, nil
		}
		if streamEvent.Type == "error" {
			var streamError anthropic.AnthropicError
			if err := json.Unmarshal(event.Data, &streamError); err != nil {
				return true, fmt.Errorf("failed to decode Anthropic stream error: %w", err)
			}
			if streamError.Error.Message == "" {
				streamError.Error.Message = "Anthropic stream error"
			}
			return true, &llm.ResponseError{Detail: llm.ErrorDetail{Message: streamError.Error.Message, Type: streamError.Error.Type, RequestID: streamError.RequestID}}
		}
		return false, nil
	default:
		return false, nil
	}
}
