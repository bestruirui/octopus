package gemini

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

type MessagesOutbound struct{}

func (o *MessagesOutbound) TransformRequest(ctx context.Context, request *model.InternalLLMRequest, baseUrl, key string) (*http.Request, error) {
	// Convert internal request to Gemini format
	geminiReq := convertLLMToGeminiRequest(request)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	// Build URL
	parsedUrl, err := url.Parse(strings.TrimSuffix(baseUrl, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse base url: %w", err)
	}

	// Determine if streaming
	isStream := request.Stream != nil && *request.Stream
	method := "generateContent"
	if isStream {
		method = "streamGenerateContent"
	}

	// Build path: /models/{model}:{method}
	modelName := request.Model
	if !strings.Contains(modelName, "/") {
		modelName = "models/" + modelName
	}
	parsedUrl.Path = fmt.Sprintf("%s/%s:%s", parsedUrl.Path, modelName, method)

	// Add API key as query parameter
	q := parsedUrl.Query()
	q.Set("key", key)
	if isStream {
		q.Set("alt", "sse")
	}
	parsedUrl.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsedUrl.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (o *MessagesOutbound) TransformResponse(ctx context.Context, response *http.Response) (*model.InternalLLMResponse, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("response body is empty")
	}

	var geminiResp model.GeminiGenerateContentResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gemini response: %w", err)
	}

	// Convert Gemini response to internal format
	return convertGeminiToLLMResponse(&geminiResp, false), nil
}

func (o *MessagesOutbound) TransformStream(ctx context.Context, eventData []byte) (*model.InternalLLMResponse, error) {
	// Handle [DONE] marker
	if bytes.HasPrefix(eventData, []byte("[DONE]")) || len(eventData) == 0 {
		return &model.InternalLLMResponse{
			Object: "[DONE]",
		}, nil
	}

	// Parse Gemini streaming response
	var geminiResp model.GeminiGenerateContentResponse
	if err := json.Unmarshal(eventData, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gemini stream chunk: %w", err)
	}

	// Convert to internal format
	return convertGeminiToLLMResponse(&geminiResp, true), nil
}

// Helper functions

func convertLLMToGeminiRequest(request *model.InternalLLMRequest) *model.GeminiGenerateContentRequest {
	geminiReq := &model.GeminiGenerateContentRequest{
		Contents: []*model.GeminiContent{},
	}

	// Convert messages
	var systemInstruction *model.GeminiContent
	for _, msg := range request.Messages {
		switch msg.Role {
		case "system", "developer":
			// Collect system messages into system instruction
			if systemInstruction == nil {
				systemInstruction = &model.GeminiContent{
					Parts: []*model.GeminiPart{},
				}
			}
			if msg.Content.Content != nil {
				systemInstruction.Parts = append(systemInstruction.Parts, &model.GeminiPart{
					Text: *msg.Content.Content,
				})
			}

		case "user":
			content := &model.GeminiContent{
				Role:  "user",
				Parts: []*model.GeminiPart{},
			}
			if msg.Content.Content != nil {
				content.Parts = append(content.Parts, &model.GeminiPart{
					Text: *msg.Content.Content,
				})
			}
			geminiReq.Contents = append(geminiReq.Contents, content)

		case "assistant":
			content := &model.GeminiContent{
				Role:  "model",
				Parts: []*model.GeminiPart{},
			}
			// Handle text content
			if msg.Content.Content != nil && *msg.Content.Content != "" {
				content.Parts = append(content.Parts, &model.GeminiPart{
					Text: *msg.Content.Content,
				})
			}
			// Handle tool calls
			if len(msg.ToolCalls) > 0 {
				for _, toolCall := range msg.ToolCalls {
					var args map[string]interface{}
					_ = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
					content.Parts = append(content.Parts, &model.GeminiPart{
						FunctionCall: &model.GeminiFunctionCall{
							Name: toolCall.Function.Name,
							Args: args,
						},
					})
				}
			}
			geminiReq.Contents = append(geminiReq.Contents, content)

		case "tool":
			// Tool result
			content := &model.GeminiContent{
				Role:  "function",
				Parts: []*model.GeminiPart{},
			}
			if msg.ToolCallID != nil && msg.Content.Content != nil {
				var response map[string]interface{}
				_ = json.Unmarshal([]byte(*msg.Content.Content), &response)
				content.Parts = append(content.Parts, &model.GeminiPart{
					FunctionResponse: &model.GeminiFunctionResponse{
						Name:     *msg.ToolCallID,
						Response: response,
					},
				})
			}
			geminiReq.Contents = append(geminiReq.Contents, content)
		}
	}

	geminiReq.SystemInstruction = systemInstruction

	// Convert generation config
	config := &model.GeminiGenerationConfig{}
	hasConfig := false

	if request.MaxTokens != nil {
		config.MaxOutputTokens = int(*request.MaxTokens)
		hasConfig = true
	}
	if request.Temperature != nil {
		config.Temperature = request.Temperature
		hasConfig = true
	}
	if request.TopP != nil {
		config.TopP = request.TopP
		hasConfig = true
	}
	// TopK is stored in metadata if present
	if topKStr, ok := request.TransformerMetadata["gemini_top_k"]; ok {
		var topK int
		fmt.Sscanf(topKStr, "%d", &topK)
		config.TopK = &topK
		hasConfig = true
	}
	if request.Stop != nil && request.Stop.MultipleStop != nil {
		config.StopSequences = request.Stop.MultipleStop
		hasConfig = true
	} else if request.Stop != nil && request.Stop.Stop != nil {
		config.StopSequences = []string{*request.Stop.Stop}
		hasConfig = true
	}

	if hasConfig {
		geminiReq.GenerationConfig = config
	}

	// Convert tools
	if len(request.Tools) > 0 {
		tools := []*model.GeminiTool{}
		functionDeclarations := []*model.GeminiFunctionDeclaration{}

		for _, tool := range request.Tools {
			if tool.Type == "function" {
				var params map[string]interface{}
				_ = json.Unmarshal(tool.Function.Parameters, &params)
				funcDecl := &model.GeminiFunctionDeclaration{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  params,
				}
				functionDeclarations = append(functionDeclarations, funcDecl)
			}
		}

		if len(functionDeclarations) > 0 {
			tools = append(tools, &model.GeminiTool{
				FunctionDeclarations: functionDeclarations,
			})
		}

		geminiReq.Tools = tools
	}

	return geminiReq

}

func convertGeminiToLLMResponse(geminiResp *model.GeminiGenerateContentResponse, isStream bool) *model.InternalLLMResponse {
	resp := &model.InternalLLMResponse{
		Choices: []model.Choice{},
	}

	if isStream {
		resp.Object = "chat.completion.chunk"
	} else {
		resp.Object = "chat.completion"
	}

	// Convert candidates to choices
	for _, candidate := range geminiResp.Candidates {
		choice := model.Choice{
			Index: candidate.Index,
		}

		// Convert finish reason
		if candidate.FinishReason != nil {
			reason := convertGeminiFinishReason(*candidate.FinishReason)
			choice.FinishReason = &reason
		}

		// Convert content
		if candidate.Content != nil {
			msg := &model.Message{
				Role: "assistant",
			}

			// Extract text and function calls from parts
			var textParts []string
			var toolCalls []model.ToolCall

			for idx, part := range candidate.Content.Parts {
				if part.Text != "" {
					textParts = append(textParts, part.Text)
				}
				if part.FunctionCall != nil {
					argsJSON, _ := json.Marshal(part.FunctionCall.Args)
					toolCall := model.ToolCall{
						Index: idx,
						ID:    fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, idx),
						Type:  "function",
						Function: model.FunctionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(argsJSON),
						},
					}
					toolCalls = append(toolCalls, toolCall)
				}
			}

			// Set content
			if len(textParts) > 0 {
				text := strings.Join(textParts, "")
				msg.Content = model.MessageContent{
					Content: &text,
				}
			}

			// Set tool calls
			if len(toolCalls) > 0 {
				msg.ToolCalls = toolCalls
				if choice.FinishReason == nil {
					reason := "tool_calls"
					choice.FinishReason = &reason
				}
			}

			if isStream {
				choice.Delta = msg
			} else {
				choice.Message = msg
			}
		}

		resp.Choices = append(resp.Choices, choice)
	}

	// Convert usage metadata
	if geminiResp.UsageMetadata != nil {
		resp.Usage = &model.Usage{
			PromptTokens:     int64(geminiResp.UsageMetadata.PromptTokenCount),
			CompletionTokens: int64(geminiResp.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int64(geminiResp.UsageMetadata.TotalTokenCount),
		}
	}

	return resp
}

func convertGeminiFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	default:
		return "stop"
	}
}
