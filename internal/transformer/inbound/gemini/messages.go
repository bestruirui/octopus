package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/samber/lo"
)

type MessagesInbound struct {
	// streamChunks stores stream chunks for aggregation
	streamChunks []*model.InternalLLMResponse
	// storedResponse stores the non-stream response
	storedResponse *model.InternalLLMResponse
}

func (i *MessagesInbound) TransformRequest(ctx context.Context, body []byte) (*model.InternalLLMRequest, error) {
	var geminiReq model.GeminiGenerateContentRequest
	if err := json.Unmarshal(body, &geminiReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gemini request: %w", err)
	}

	// Convert to internal LLM request
	chatReq := &model.InternalLLMRequest{
		Model:               "",
		Metadata:            map[string]string{},
		RawAPIFormat:        model.APIFormatGeminiContents,
		TransformerMetadata: map[string]string{},
	}

	// Convert system instruction if present
	messages := make([]model.Message, 0)
	if geminiReq.SystemInstruction != nil && len(geminiReq.SystemInstruction.Parts) > 0 {
		systemContent := convertGeminiPartsToText(geminiReq.SystemInstruction.Parts)
		messages = append(messages, model.Message{
			Role: "system",
			Content: model.MessageContent{
				Content: &systemContent,
			},
		})
	}

	// Convert contents (messages)
	for _, content := range geminiReq.Contents {
		msg := model.Message{}

		// Convert role
		switch content.Role {
		case "user":
			msg.Role = "user"
		case "model":
			msg.Role = "assistant"
		case "function":
			msg.Role = "tool"
		default:
			msg.Role = content.Role
		}

		// Convert parts to content
		if len(content.Parts) > 0 {
			// Check if it's a function call
			if hasFunctionCall(content.Parts) {
				toolCalls := convertGeminiPartsToToolCalls(content.Parts)
				msg.ToolCalls = toolCalls
			} else if hasFunctionResponse(content.Parts) {
				// Tool result
				for _, part := range content.Parts {
					if part.FunctionResponse != nil {
						msg.Role = "tool"
						msg.ToolCallID = &part.FunctionResponse.Name
						responseJSON, _ := json.Marshal(part.FunctionResponse.Response)
						responseStr := string(responseJSON)
						msg.Content = model.MessageContent{
							Content: &responseStr,
						}
						break
					}
				}
			} else {
				// Regular text content
				textContent := convertGeminiPartsToText(content.Parts)
				msg.Content = model.MessageContent{
					Content: &textContent,
				}
			}
		}

		messages = append(messages, msg)
	}

	chatReq.Messages = messages

	// Convert generation config
	if geminiReq.GenerationConfig != nil {
		if geminiReq.GenerationConfig.MaxOutputTokens > 0 {
			maxTokens := int64(geminiReq.GenerationConfig.MaxOutputTokens)
			chatReq.MaxTokens = &maxTokens
		}
		if geminiReq.GenerationConfig.Temperature != nil {
			chatReq.Temperature = geminiReq.GenerationConfig.Temperature
		}
		if geminiReq.GenerationConfig.TopP != nil {
			chatReq.TopP = geminiReq.GenerationConfig.TopP
		}
		if geminiReq.GenerationConfig.TopK != nil {
			// TopK is not supported in the standard model, but we can store it in metadata
			chatReq.TransformerMetadata["gemini_top_k"] = fmt.Sprintf("%d", *geminiReq.GenerationConfig.TopK)
		}
		if geminiReq.GenerationConfig.StopSequences != nil {
			stopSequences := &model.Stop{
				MultipleStop: geminiReq.GenerationConfig.StopSequences,
			}
			chatReq.Stop = stopSequences
		}
	}

	// Convert tools
	if len(geminiReq.Tools) > 0 {
		tools := make([]model.Tool, 0)
		for _, geminiTool := range geminiReq.Tools {
			if geminiTool.FunctionDeclarations != nil {
				for _, funcDecl := range geminiTool.FunctionDeclarations {
					paramsJSON, _ := json.Marshal(funcDecl.Parameters)
					tool := model.Tool{
						Type: "function",
						Function: model.Function{
							Name:        funcDecl.Name,
							Description: funcDecl.Description,
							Parameters:  paramsJSON,
						},
					}
					tools = append(tools, tool)
				}
			}
		}
		chatReq.Tools = tools
	}

	return chatReq, nil
}

func (i *MessagesInbound) TransformResponse(ctx context.Context, response *model.InternalLLMResponse) ([]byte, error) {
	// Store the response for later retrieval
	i.storedResponse = response

	// Convert internal response to Gemini response
	geminiResp := convertLLMToGeminiResponse(response, false)

	body, err := json.Marshal(geminiResp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini response: %w", err)
	}
	return body, nil
}

func (i *MessagesInbound) TransformStream(ctx context.Context, stream *model.InternalLLMResponse) ([]byte, error) {
	if stream.Object == "[DONE]" {
		return []byte("data: [DONE]\n\n"), nil
	}

	// Store the chunk for aggregation
	i.streamChunks = append(i.streamChunks, stream)

	// Convert to Gemini streaming format
	geminiResp := convertLLMToGeminiResponse(stream, true)

	body, err := json.Marshal(geminiResp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini stream: %w", err)
	}

	return []byte("data: " + string(body) + "\n\n"), nil
}

func (i *MessagesInbound) GetInternalResponse(ctx context.Context) (*model.InternalLLMResponse, error) {
	if i.storedResponse != nil {
		return i.storedResponse, nil
	}

	if len(i.streamChunks) > 0 {
		// Aggregate stream chunks into a complete response
		return aggregateStreamChunks(i.streamChunks), nil
	}

	return nil, fmt.Errorf("no response stored")
}

// Helper functions

func convertGeminiPartsToText(parts []*model.GeminiPart) string {
	var texts []string
	for _, part := range parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	if len(texts) == 0 {
		return ""
	}
	return texts[0] // For simplicity, return first text part
}

func hasFunctionCall(parts []*model.GeminiPart) bool {
	for _, part := range parts {
		if part.FunctionCall != nil {
			return true
		}
	}
	return false
}

func hasFunctionResponse(parts []*model.GeminiPart) bool {
	for _, part := range parts {
		if part.FunctionResponse != nil {
			return true
		}
	}
	return false
}

func convertGeminiPartsToToolCalls(parts []*model.GeminiPart) []model.ToolCall {
	toolCalls := make([]model.ToolCall, 0)
	for idx, part := range parts {
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
	return toolCalls
}

func convertLLMToGeminiResponse(response *model.InternalLLMResponse, isStream bool) *model.GeminiGenerateContentResponse {
	geminiResp := &model.GeminiGenerateContentResponse{
		Candidates: []*model.GeminiCandidate{},
	}

	if len(response.Choices) > 0 {
		for _, choice := range response.Choices {
			candidate := &model.GeminiCandidate{
				Index: choice.Index,
			}

			// Convert finish reason
			if choice.FinishReason != nil {
				reason := convertFinishReason(*choice.FinishReason)
				candidate.FinishReason = &reason
			}

			// Convert message/delta
			var msg *model.Message
			if isStream && choice.Delta != nil {
				msg = choice.Delta
			} else if choice.Message != nil {
				msg = choice.Message
			}

			if msg != nil {
				content := &model.GeminiContent{
					Role:  "model",
					Parts: []*model.GeminiPart{},
				}

				// Convert text content
				if msg.Content.Content != nil && *msg.Content.Content != "" {
					content.Parts = append(content.Parts, &model.GeminiPart{
						Text: *msg.Content.Content,
					})
				}

				// Convert tool calls
				if len(msg.ToolCalls) > 0 {
					for _, toolCall := range msg.ToolCalls {
						if toolCall.Function.Name != "" {
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
				}

				candidate.Content = content
			}

			geminiResp.Candidates = append(geminiResp.Candidates, candidate)
		}
	}

	// Add usage info if present
	if response.Usage != nil {
		geminiResp.UsageMetadata = &model.GeminiUsageMetadata{
			PromptTokenCount:     int(response.Usage.PromptTokens),
			CandidatesTokenCount: int(response.Usage.CompletionTokens),
			TotalTokenCount:      int(response.Usage.TotalTokens),
		}
	}

	return geminiResp
}

func convertFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "STOP"
	case "length":
		return "MAX_TOKENS"
	case "tool_calls":
		return "STOP"
	case "content_filter":
		return "SAFETY"
	default:
		return "OTHER"
	}
}

func aggregateStreamChunks(chunks []*model.InternalLLMResponse) *model.InternalLLMResponse {
	if len(chunks) == 0 {
		return nil
	}

	// Start with the first chunk
	result := &model.InternalLLMResponse{
		ID:      chunks[0].ID,
		Object:  "chat.completion",
		Created: chunks[0].Created,
		Model:   chunks[0].Model,
		Choices: []model.Choice{},
	}

	// Aggregate content and usage
	contentBuilder := ""
	var totalUsage *model.Usage

	for _, chunk := range chunks {
		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]
			if choice.Delta != nil && choice.Delta.Content.Content != nil {
				contentBuilder += *choice.Delta.Content.Content
			}
			if choice.FinishReason != nil {
				result.Choices = []model.Choice{
					{
						Index:        0,
						Message:      &model.Message{},
						FinishReason: choice.FinishReason,
					},
				}
			}
		}
		if chunk.Usage != nil {
			totalUsage = chunk.Usage
		}
	}

	// Set aggregated content
	if len(result.Choices) > 0 {
		result.Choices[0].Message.Role = "assistant"
		result.Choices[0].Message.Content = model.MessageContent{
			Content: lo.ToPtr(contentBuilder),
		}
	} else {
		result.Choices = []model.Choice{
			{
				Index: 0,
				Message: &model.Message{
					Role: "assistant",
					Content: model.MessageContent{
						Content: lo.ToPtr(contentBuilder),
					},
				},
			},
		}
	}

	result.Usage = totalUsage

	return result
}
