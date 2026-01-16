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

	// Debug: log response metadata without the large base64 image data
	if len(resp.Choices) > 0 && len(resp.Choices[0].Message.Images) > 0 {
		imageCount := len(resp.Choices[0].Message.Images)
		totalImageSize := 0
		for _, img := range resp.Choices[0].Message.Images {
			if img.ImageURL != nil {
				totalImageSize += len(img.ImageURL.URL)
			}
		}
		fmt.Printf("[DEBUG OpenAI Response] Images detected: count=%d, totalSize=%d bytes\n", imageCount, totalImageSize)
	} else {
		// Only log responses without large image data
		bodyStr := string(body)
		fmt.Printf("[DEBUG OpenAI Response] Length: %d, First 2000 chars: %s\n", len(body), bodyStr[:min(len(bodyStr), 2000)])
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

	// Debug: log stream chunk metadata without the large base64 image data
	debugStr := string(eventData)
	// Hide large base64 image data in logs to reduce storage pressure
	if len(resp.Choices) > 0 && len(resp.Choices[0].Delta.Images) > 0 {
		imageCount := len(resp.Choices[0].Delta.Images)
		totalImageSize := 0
		for _, img := range resp.Choices[0].Delta.Images {
			if img.ImageURL != nil {
				totalImageSize += len(img.ImageURL.URL)
			}
		}
		fmt.Printf("[DEBUG OpenAI Stream] Images detected: count=%d, totalSize=%d bytes\n", imageCount, totalImageSize)
	} else {
		// Only log non-image chunks to keep logs manageable
		fmt.Printf("[DEBUG OpenAI Stream] Length: %d, First 2000 chars: %s\n", len(eventData), debugStr[:min(len(debugStr), 2000)])
	}

	return &resp, nil
}
