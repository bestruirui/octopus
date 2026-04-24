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

	"github.com/lingyuins/octopus/internal/transformer/model"
)

type ChatOutbound struct{}

func (o *ChatOutbound) TransformRequest(ctx context.Context, request *model.InternalLLMRequest, baseUrl, key string) (*http.Request, error) {
	compatRequest := cloneRequestForOpenAICompat(request)
	compatRequest.ClearHelpFields()

	// Convert developer role to system role for compatibility
	for i := range compatRequest.Messages {
		if compatRequest.Messages[i].Role == "developer" {
			compatRequest.Messages[i].Role = "system"
		}
	}

	normalizeMessagesForOpenAICompat(compatRequest.Messages)

	if compatRequest.Stream != nil && *compatRequest.Stream {
		if compatRequest.StreamOptions == nil {
			compatRequest.StreamOptions = &model.StreamOptions{IncludeUsage: true}
		} else if !compatRequest.StreamOptions.IncludeUsage {
			compatRequest.StreamOptions.IncludeUsage = true
		}
	}

	body, err := json.Marshal(compatRequest)
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

func cloneRequestForOpenAICompat(request *model.InternalLLMRequest) *model.InternalLLMRequest {
	if request == nil {
		return nil
	}

	cloned := *request
	if len(request.Messages) > 0 {
		cloned.Messages = append([]model.Message(nil), request.Messages...)
	}
	if request.StreamOptions != nil {
		streamOptions := *request.StreamOptions
		cloned.StreamOptions = &streamOptions
	}

	return &cloned
}

func normalizeMessagesForOpenAICompat(messages []model.Message) {
	for i := range messages {
		normalizeMessageForOpenAICompat(&messages[i])
	}
}

func normalizeMessageForOpenAICompat(msg *model.Message) {
	if len(msg.Content.MultipleContent) > 0 {
		if text, ok := flattenTextOnlyContent(msg.Content.MultipleContent); ok {
			msg.Content = model.MessageContent{
				Content: &text,
			}
		} else if msg.Role == "tool" {
			text := flattenTextContent(msg.Content.MultipleContent)
			msg.Content = model.MessageContent{
				Content: &text,
			}
		}
	}

	if msg.Content.Content == nil && len(msg.Content.MultipleContent) == 0 {
		empty := ""
		msg.Content.Content = &empty
	}
}

func flattenTextOnlyContent(parts []model.MessageContentPart) (string, bool) {
	if len(parts) == 0 {
		return "", false
	}

	textParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Type != "text" || part.Text == nil {
			return "", false
		}
		textParts = append(textParts, *part.Text)
	}

	return strings.Join(textParts, "\n"), true
}

func flattenTextContent(parts []model.MessageContentPart) string {
	if len(parts) == 0 {
		return ""
	}

	textParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Type == "text" && part.Text != nil {
			textParts = append(textParts, *part.Text)
		}
	}

	return strings.Join(textParts, "\n")
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
