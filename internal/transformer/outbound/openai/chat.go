package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/bestruirui/octopus/internal/transformer/model"
)

type ChatOutbound struct{}

func (o *ChatOutbound) TransformRequest(ctx context.Context, request *model.InternalLLMRequest, baseUrl, key string) (*http.Request, error) {
	// Kimi K2.5 等模型在启用 thinking 模式时，要求 assistant 的 tool_calls 消息包含 reasoning_content 字段
	// 这个处理必须在 ClearHelpFields 之前完成，因为我们需要保留原始请求信息
	o.fillMissingReasoningContent(request)

	request.ClearHelpFields()

	// Convert developer role to system role for compatibility
	for i := range request.Messages {
		if request.Messages[i].Role == "developer" {
			request.Messages[i].Role = "system"
		}
	}

	if request.Stream != nil && *request.Stream {
		if request.StreamOptions == nil {
			request.StreamOptions = &model.StreamOptions{IncludeUsage: true}
		} else if !request.StreamOptions.IncludeUsage {
			request.StreamOptions.IncludeUsage = true
		}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	parsedUrl, err := url.Parse(strings.TrimSuffix(baseUrl, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse base url: %w", err)
	}
	parsedUrl.Path = parsedUrl.Path + "/chat/completions"
	req.URL = parsedUrl
	req.Method = http.MethodPost
	return req, nil
}

// fillMissingReasoningContent 为 assistant 的 tool_calls 消息填充 reasoning_content 字段
// 某些模型（如 Kimi K2.5）在启用 thinking 模式时，要求 assistant 的 tool_calls 消息必须包含 reasoning_content 字段
func (o *ChatOutbound) fillMissingReasoningContent(request *model.InternalLLMRequest) {
	// 只有启用了 thinking 模式时才需要处理（在 relay.go 中已根据原始模型名设置）
	if request.EnableThinking == nil || !*request.EnableThinking {
		return
	}

	// Kimi K2.5 在启用 thinking 模式时，tool_choice 不能为 "specified" 或 "required"
	// 也不能指定特定工具（NamedToolChoice），需要将其改为 "auto"
	// 这个限制适用于所有启用了 thinking 模式的模型
	if request.ToolChoice != nil {
		if request.ToolChoice.ToolChoice != nil {
			choice := *request.ToolChoice.ToolChoice
			if choice == "specified" || choice == "required" {
				autoChoice := "auto"
				request.ToolChoice.ToolChoice = &autoChoice
			}
		} else if request.ToolChoice.NamedToolChoice != nil {
			// 如果指定了具体工具，也改为 auto
			autoChoice := "auto"
			request.ToolChoice.ToolChoice = &autoChoice
			request.ToolChoice.NamedToolChoice = nil
		}
	}

	for i := range request.Messages {
		msg := &request.Messages[i]
		// 只处理 assistant 角色的消息，且包含 tool_calls 的消息
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			// 如果 reasoning_content 为空，添加一个空格
			// 注意：必须使用非空值，因为 omitempty 会省略空字符串
			if msg.GetReasoningContent() == "" {
				spaceStr := " "
				msg.ReasoningContent = &spaceStr
			}
		}
	}
}

func (o *ChatOutbound) TransformResponse(ctx context.Context, response *http.Response) (*model.InternalLLMResponse, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("response body is empty")
	}

	var resp model.InternalLLMResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &resp, nil
}

func (o *ChatOutbound) TransformStream(ctx context.Context, eventData []byte) (*model.InternalLLMResponse, error) {
	if bytes.HasPrefix(eventData, []byte("[DONE]")) {
		return &model.InternalLLMResponse{
			Object: "[DONE]",
		}, nil
	}

	var errCheck struct {
		Error *model.ErrorDetail `json:"error"`
	}
	if err := json.Unmarshal(eventData, &errCheck); err == nil && errCheck.Error != nil {
		return nil, &model.ResponseError{
			Detail: *errCheck.Error,
		}
	}

	var resp model.InternalLLMResponse
	if err := json.Unmarshal(eventData, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stream chunk: %w", err)
	}
	return &resp, nil
}
