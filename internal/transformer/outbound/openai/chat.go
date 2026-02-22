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
	request.ClearHelpFields()
	ensureKimiReasoningContentForToolCalls(request, baseUrl)

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

func ensureKimiReasoningContentForToolCalls(request *model.InternalLLMRequest, baseURL string) {
	if request == nil || !isKimiChatCompatibilityRequired(request.Model, baseURL) {
		return
	}

	for i := range request.Messages {
		msg := &request.Messages[i]
		if msg.Role != "assistant" || len(msg.ToolCalls) == 0 {
			continue
		}

		existing := strings.TrimSpace(msg.GetReasoningContent())
		if existing != "" {
			msg.SetReasoningContent(existing)
			continue
		}

		// Kimi treats empty reasoning_content as missing when thinking is enabled.
		// Prefer nearby assistant reasoning/context, then fall back to a stable non-empty marker.
		if inferred := inferAssistantReasoningContent(request.Messages, i); inferred != "" {
			msg.SetReasoningContent(inferred)
			continue
		}

		msg.SetReasoningContent("tool call")
	}
}

func isKimiChatCompatibilityRequired(modelName, baseURL string) bool {
	modelLower := strings.ToLower(strings.TrimSpace(modelName))
	baseLower := strings.ToLower(strings.TrimSpace(baseURL))
	return strings.Contains(modelLower, "kimi") || strings.Contains(baseLower, "api.kimi.com")
}

func inferAssistantReasoningContent(messages []model.Message, idx int) string {
	if idx < 0 || idx >= len(messages) {
		return ""
	}

	// Prefer most recent assistant reasoning before the current message.
	for i := idx - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != "assistant" {
			continue
		}
		if reasoning := strings.TrimSpace(msg.GetReasoningContent()); reasoning != "" {
			return reasoning
		}
		if msg.Content.Content != nil {
			if content := strings.TrimSpace(*msg.Content.Content); content != "" {
				return content
			}
		}
	}

	// Fall back to current assistant textual content when present.
	if messages[idx].Content.Content != nil {
		if content := strings.TrimSpace(*messages[idx].Content.Content); content != "" {
			return content
		}
	}

	return ""
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
