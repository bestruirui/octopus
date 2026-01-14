package authropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/samber/lo"

	anthropicModel "github.com/bestruirui/octopus/internal/transformer/inbound/anthropic"
	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/utils/xurl"
)

type MessageOutbound struct {
	// Stream state tracking
	streamID    string
	streamModel string
	streamUsage *model.Usage
	toolIndex   int
	toolCalls   map[int]*model.ToolCall
	initialized bool
}

func (o *MessageOutbound) TransformRequest(ctx context.Context, request *model.InternalLLMRequest, baseUrl, key string) (*http.Request, error) {
	if request == nil {
		return nil, fmt.Errorf("request is nil")
	}

	// Convert to Anthropic request format
	anthropicReq := convertToAnthropicRequest(request)

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal anthropic request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	// For streaming requests, Anthropic returns Server-Sent Events.
	if request.Stream != nil && *request.Stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("Anthropic-Version", "2023-06-01")
	req.Header.Set("X-API-Key", key)

	// Parse and set URL
	parsedUrl, err := url.Parse(strings.TrimSuffix(baseUrl, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse base url: %w", err)
	}

	parsedUrl.Path = parsedUrl.Path + "/messages"
	// Pass through the original query parameters exactly as-is
	if request.Query != nil {
		parsedUrl.RawQuery = request.Query.Encode()
	}
	req.URL = parsedUrl

	return req, nil
}

func (o *MessageOutbound) TransformResponse(ctx context.Context, response *http.Response) (*model.InternalLLMResponse, error) {
	if response == nil {
		return nil, fmt.Errorf("response is nil")
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("response body is empty")
	}

	// Check for error response
	if response.StatusCode >= 400 {
		var errResp anthropicModel.AnthropicError
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, &model.ResponseError{
				StatusCode: response.StatusCode,
				Detail: model.ErrorDetail{
					Message: errResp.Error.Message,
					Type:    errResp.Error.Type,
				},
			}
		}
		return nil, fmt.Errorf("HTTP error %d: %s", response.StatusCode, string(body))
	}

	var anthropicResp anthropicModel.Message
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal anthropic response: %w", err)
	}

	// Convert to internal response
	return convertToLLMResponse(&anthropicResp), nil
}

func (o *MessageOutbound) TransformStream(ctx context.Context, eventData []byte) (*model.InternalLLMResponse, error) {
	if len(eventData) == 0 {
		return nil, nil
	}

	// Handle [DONE] marker
	if bytes.HasPrefix(eventData, []byte("[DONE]")) {
		return &model.InternalLLMResponse{
			Object: "[DONE]",
		}, nil
	}

	// Initialize state if needed
	if !o.initialized {
		o.toolCalls = make(map[int]*model.ToolCall)
		o.toolIndex = -1
		o.initialized = true
	}

	// Parse the streaming event
	var streamEvent anthropicModel.StreamEvent
	if err := json.Unmarshal(eventData, &streamEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stream event: %w", err)
	}

	resp := &model.InternalLLMResponse{
		ID:      o.streamID,
		Model:   o.streamModel,
		Object:  "chat.completion.chunk",
		Created: 0,
	}

	switch streamEvent.Type {
	case "message_start":
		if streamEvent.Message != nil {
			o.streamID = streamEvent.Message.ID
			o.streamModel = streamEvent.Message.Model
			resp.ID = o.streamID
			resp.Model = o.streamModel

			if streamEvent.Message.Usage != nil {
				o.streamUsage = convertAnthropicUsage(streamEvent.Message.Usage)
				resp.Usage = o.streamUsage
			}
		}

		resp.Choices = []model.Choice{
			{
				Index: 0,
				Delta: &model.Message{
					Role: "assistant",
				},
			},
		}

	case "content_block_start":
		if streamEvent.ContentBlock != nil {
			switch streamEvent.ContentBlock.Type {
			case "tool_use":
				o.toolIndex++
				toolCall := model.ToolCall{
					Index: o.toolIndex,
					ID:    streamEvent.ContentBlock.ID,
					Type:  "function",
					Function: model.FunctionCall{
						Name:      lo.FromPtr(streamEvent.ContentBlock.Name),
						Arguments: "",
					},
				}
				o.toolCalls[o.toolIndex] = &toolCall

				resp.Choices = []model.Choice{
					{
						Index: 0,
						Delta: &model.Message{
							Role:      "assistant",
							ToolCalls: []model.ToolCall{toolCall},
						},
					},
				}
			case "text", "thinking":
				// These are handled in content_block_delta
				return nil, nil
			default:
				return nil, nil
			}
		}

	case "content_block_delta":
		if streamEvent.Delta != nil && streamEvent.Delta.Type != nil {
			choice := model.Choice{
				Index: 0,
				Delta: &model.Message{
					Role: "assistant",
				},
			}

			switch *streamEvent.Delta.Type {
			case "text_delta":
				if streamEvent.Delta.Text != nil {
					choice.Delta.Content = model.MessageContent{
						Content: streamEvent.Delta.Text,
					}
				}
			case "input_json_delta":
				if streamEvent.Delta.PartialJSON != nil && o.toolIndex >= 0 {
					choice.Delta.ToolCalls = []model.ToolCall{
						{
							Index: o.toolIndex,
							ID:    o.toolCalls[o.toolIndex].ID,
							Type:  "function",
							Function: model.FunctionCall{
								Arguments: *streamEvent.Delta.PartialJSON,
							},
						},
					}
				}
			case "thinking_delta":
				if streamEvent.Delta.Thinking != nil {
					choice.Delta.ReasoningContent = streamEvent.Delta.Thinking
				}
			case "signature_delta":
				if streamEvent.Delta.Signature != nil {
					choice.Delta.ReasoningSignature = streamEvent.Delta.Signature
				}
			default:
				return nil, nil
			}

			resp.Choices = []model.Choice{choice}
		}

	case "message_delta":
		if streamEvent.Usage != nil {
			usage := convertAnthropicUsage(streamEvent.Usage)
			if o.streamUsage != nil {
				usage.PromptTokens = o.streamUsage.PromptTokens
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			}
			o.streamUsage = usage
		}

		if streamEvent.Delta != nil && streamEvent.Delta.StopReason != nil {
			finishReason := convertStopReason(streamEvent.Delta.StopReason)
			resp.Choices = []model.Choice{
				{
					Index:        0,
					FinishReason: finishReason,
				},
			}
		}

	case "message_stop":
		resp.Choices = []model.Choice{}
		if o.streamUsage != nil {
			resp.Usage = o.streamUsage
		}

	case "content_block_stop", "ping":
		return nil, nil

	default:
		return nil, nil
	}

	return resp, nil
}

// convertToAnthropicRequest converts internal LLM request to Anthropic format
func convertToAnthropicRequest(req *model.InternalLLMRequest) *anthropicModel.MessageRequest {
	result := &anthropicModel.MessageRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
		MaxTokens:   resolveMaxTokens(req),
		System:      convertSystemPrompt(req),
	}

	if req.Metadata != nil && req.Metadata["user_id"] != "" {
		result.Metadata = &anthropicModel.AnthropicMetadata{UserID: req.Metadata["user_id"]}
	}

	// Convert messages
	result.Messages = convertMessages(req)

	// Convert tools
	if len(req.Tools) > 0 {
		result.Tools = convertTools(req.Tools)
	}

	// Convert stop sequences
	if req.Stop != nil {
		result.StopSequences = convertStopSequences(req.Stop)
	}

	// Convert thinking/reasoning
	if req.ReasoningEffort != "" {
		result.Thinking = &anthropicModel.Thinking{
			Type:         "enabled",
			BudgetTokens: getThinkingBudget(req.ReasoningEffort, req.ReasoningBudget),
		}
	}

	return result
}

func resolveMaxTokens(req *model.InternalLLMRequest) int64 {
	switch {
	case req.MaxTokens != nil:
		return *req.MaxTokens
	case req.MaxCompletionTokens != nil:
		return *req.MaxCompletionTokens
	default:
		return 8192
	}
}

func convertSystemPrompt(req *model.InternalLLMRequest) *anthropicModel.SystemPrompt {
	var systemMessages []model.Message
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		}
	}

	if len(systemMessages) == 0 {
		return nil
	}

	wasArrayFormat := req.TransformerMetadata != nil && req.TransformerMetadata["anthropic_system_array_format"] == "true"

	if len(systemMessages) == 1 {
		if wasArrayFormat {
			return &anthropicModel.SystemPrompt{
				MultiplePrompts: []anthropicModel.SystemPromptPart{{
					Type:         "text",
					Text:         lo.FromPtr(systemMessages[0].Content.Content),
					CacheControl: convertCacheControl(systemMessages[0].CacheControl),
				}},
			}
		}
		return &anthropicModel.SystemPrompt{
			Prompt: systemMessages[0].Content.Content,
		}
	}

	parts := make([]anthropicModel.SystemPromptPart, 0, len(systemMessages))
	for _, msg := range systemMessages {
		parts = append(parts, anthropicModel.SystemPromptPart{
			Type:         "text",
			Text:         lo.FromPtr(msg.Content.Content),
			CacheControl: convertCacheControl(msg.CacheControl),
		})
	}
	return &anthropicModel.SystemPrompt{
		MultiplePrompts: parts,
	}
}

func convertMessages(req *model.InternalLLMRequest) []anthropicModel.MessageParam {
	messages := make([]anthropicModel.MessageParam, 0, len(req.Messages))
	processedIndexes := make(map[int]bool)

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			continue
		}

		converted := convertSingleMessage(msg, req.Messages, processedIndexes)
		messages = append(messages, converted...)
	}

	return messages
}

func convertSingleMessage(msg model.Message, allMessages []model.Message, processedIndexes map[int]bool) []anthropicModel.MessageParam {
	switch msg.Role {
	case "tool":
		return convertToolMessage(msg, allMessages, processedIndexes)
	case "user":
		if msg.MessageIndex != nil && processedIndexes[*msg.MessageIndex] {
			return nil
		}
		return convertUserMessage(msg)
	case "assistant":
		return convertAssistantMessage(msg)
	default:
		return nil
	}
}

func convertToolMessage(msg model.Message, allMessages []model.Message, processedIndexes map[int]bool) []anthropicModel.MessageParam {
	if msg.MessageIndex == nil {
		return []anthropicModel.MessageParam{{
			Role: "user",
			Content: anthropicModel.MessageContent{
				MultipleContent: []anthropicModel.MessageContentBlock{convertToolResultBlock(msg)},
			},
		}}
	}

	if processedIndexes[*msg.MessageIndex] {
		return nil
	}

	var toolMsgs []model.Message
	for _, m := range allMessages {
		if m.Role == "tool" && m.MessageIndex != nil && *m.MessageIndex == *msg.MessageIndex {
			toolMsgs = append(toolMsgs, m)
		}
	}

	if len(toolMsgs) == 0 {
		return nil
	}

	contentBlocks := make([]anthropicModel.MessageContentBlock, 0, len(toolMsgs))
	for _, tm := range toolMsgs {
		contentBlocks = append(contentBlocks, convertToolResultBlock(tm))
	}

	// Merge the associated user message content (if any) into the same Anthropic user message.
	// In Anthropic Messages, tool_result blocks live inside a user message's content array.
	// Our internal format represents tool results as separate "tool" role messages, but the
	// original Anthropic request may also include additional user content alongside tool_result.
	if userMsg := findUserMessageByIndex(allMessages, *msg.MessageIndex); userMsg != nil {
		userContent := buildMessageContent(*userMsg)
		if len(userContent.MultipleContent) > 0 {
			contentBlocks = append(contentBlocks, userContent.MultipleContent...)
		} else if userContent.Content != nil && *userContent.Content != "" {
			contentBlocks = append(contentBlocks, anthropicModel.MessageContentBlock{
				Type: "text",
				Text: userContent.Content,
			})
		}
	}

	processedIndexes[*msg.MessageIndex] = true

	return []anthropicModel.MessageParam{{
		Role:    "user",
		Content: anthropicModel.MessageContent{MultipleContent: contentBlocks},
	}}
}

func findUserMessageByIndex(allMessages []model.Message, messageIndex int) *model.Message {
	for i := range allMessages {
		m := &allMessages[i]
		if m.Role == "user" && m.MessageIndex != nil && *m.MessageIndex == messageIndex {
			return m
		}
	}
	return nil
}

func convertToolResultBlock(msg model.Message) anthropicModel.MessageContentBlock {
	block := anthropicModel.MessageContentBlock{
		Type:         "tool_result",
		ToolUseID:    msg.ToolCallID,
		CacheControl: convertCacheControl(msg.CacheControl),
		IsError:      msg.ToolCallIsError,
	}

	if msg.Content.Content != nil {
		block.Content = &anthropicModel.MessageContent{
			Content: msg.Content.Content,
		}
	} else if len(msg.Content.MultipleContent) > 0 {
		blocks := make([]anthropicModel.MessageContentBlock, 0, len(msg.Content.MultipleContent))
		for _, part := range msg.Content.MultipleContent {
			if part.Type == "text" && part.Text != nil {
				blocks = append(blocks, anthropicModel.MessageContentBlock{
					Type: "text",
					Text: part.Text,
				})
			}
		}
		block.Content = &anthropicModel.MessageContent{
			MultipleContent: blocks,
		}
	}

	return block
}

func convertUserMessage(msg model.Message) []anthropicModel.MessageParam {
	content := buildMessageContent(msg)
	return []anthropicModel.MessageParam{{Role: "user", Content: content}}
}

func convertAssistantMessage(msg model.Message) []anthropicModel.MessageParam {
	if len(msg.ToolCalls) > 0 {
		return convertAssistantWithToolCalls(msg)
	}

	content := buildMessageContent(msg)
	return []anthropicModel.MessageParam{{Role: "assistant", Content: content}}
}

func convertAssistantWithToolCalls(msg model.Message) []anthropicModel.MessageParam {
	var blocks []anthropicModel.MessageContentBlock

	// Add thinking block if present
	if msg.ReasoningContent != nil && *msg.ReasoningContent != "" {
		blocks = append(blocks, anthropicModel.MessageContentBlock{
			Type:      "thinking",
			Thinking:  msg.ReasoningContent,
			Signature: msg.ReasoningSignature,
		})
	}

	// Add text content if present
	if msg.Content.Content != nil && *msg.Content.Content != "" {
		blocks = append(blocks, anthropicModel.MessageContentBlock{
			Type:         "text",
			Text:         msg.Content.Content,
			CacheControl: convertCacheControl(msg.CacheControl),
		})
	} else if len(msg.Content.MultipleContent) > 0 {
		for _, part := range msg.Content.MultipleContent {
			if part.Type == "text" && part.Text != nil {
				blocks = append(blocks, anthropicModel.MessageContentBlock{
					Type:         "text",
					Text:         part.Text,
					CacheControl: convertCacheControl(part.CacheControl),
				})
			}
		}
	}

	// Add tool calls
	for _, toolCall := range msg.ToolCalls {
		input := json.RawMessage("{}")
		if toolCall.Function.Arguments != "" {
			if json.Valid([]byte(toolCall.Function.Arguments)) {
				input = json.RawMessage(toolCall.Function.Arguments)
			}
		}
		blocks = append(blocks, anthropicModel.MessageContentBlock{
			Type:         "tool_use",
			ID:           toolCall.ID,
			Name:         &toolCall.Function.Name,
			Input:        input,
			CacheControl: convertCacheControl(toolCall.CacheControl),
		})
	}

	if len(blocks) == 0 {
		return nil
	}

	return []anthropicModel.MessageParam{{
		Role:    "assistant",
		Content: anthropicModel.MessageContent{MultipleContent: blocks},
	}}
}

func buildMessageContent(msg model.Message) anthropicModel.MessageContent {
	// Handle simple string content
	if msg.Content.Content != nil {
		if msg.CacheControl != nil || hasThinkingContent(msg) {
			return buildMultipleContentWithThinking(msg)
		}
		return anthropicModel.MessageContent{Content: msg.Content.Content}
	}

	// Handle multiple content parts
	if len(msg.Content.MultipleContent) > 0 {
		return convertMultiplePartContent(msg)
	}

	return anthropicModel.MessageContent{}
}

func hasThinkingContent(msg model.Message) bool {
	return msg.ReasoningContent != nil && *msg.ReasoningContent != ""
}

func buildMultipleContentWithThinking(msg model.Message) anthropicModel.MessageContent {
	var blocks []anthropicModel.MessageContentBlock

	if msg.ReasoningContent != nil && *msg.ReasoningContent != "" {
		blocks = append(blocks, anthropicModel.MessageContentBlock{
			Type:      "thinking",
			Thinking:  msg.ReasoningContent,
			Signature: msg.ReasoningSignature,
		})
	}

	blocks = append(blocks, anthropicModel.MessageContentBlock{
		Type:         "text",
		Text:         msg.Content.Content,
		CacheControl: convertCacheControl(msg.CacheControl),
	})

	return anthropicModel.MessageContent{MultipleContent: blocks}
}

func convertMultiplePartContent(msg model.Message) anthropicModel.MessageContent {
	blocks := make([]anthropicModel.MessageContentBlock, 0, len(msg.Content.MultipleContent))

	for _, part := range msg.Content.MultipleContent {
		switch part.Type {
		case "text":
			if part.Text != nil {
				blocks = append(blocks, anthropicModel.MessageContentBlock{
					Type:         "text",
					Text:         part.Text,
					CacheControl: convertCacheControl(part.CacheControl),
				})
			}
		case "image_url":
			if part.ImageURL != nil && part.ImageURL.URL != "" {
				block := convertImageURLToBlock(part)
				if block != nil {
					blocks = append(blocks, *block)
				}
			}
		}
	}

	// Add tool calls if present
	for _, toolCall := range msg.ToolCalls {
		input := json.RawMessage("{}")
		if toolCall.Function.Arguments != "" {
			if json.Valid([]byte(toolCall.Function.Arguments)) {
				input = json.RawMessage(toolCall.Function.Arguments)
			}
		}
		blocks = append(blocks, anthropicModel.MessageContentBlock{
			Type:         "tool_use",
			ID:           toolCall.ID,
			Name:         &toolCall.Function.Name,
			Input:        input,
			CacheControl: convertCacheControl(toolCall.CacheControl),
		})
	}

	if len(blocks) == 0 {
		return anthropicModel.MessageContent{}
	}

	return anthropicModel.MessageContent{MultipleContent: blocks}
}

func convertImageURLToBlock(part model.MessageContentPart) *anthropicModel.MessageContentBlock {
	if part.ImageURL == nil || part.ImageURL.URL == "" {
		return nil
	}

	url := part.ImageURL.URL
	if parsed := xurl.ParseDataURL(url); parsed != nil {
		return &anthropicModel.MessageContentBlock{
			Type: "image",
			Source: &anthropicModel.ImageSource{
				Type:      "base64",
				MediaType: parsed.MediaType,
				Data:      parsed.Data,
			},
			CacheControl: convertCacheControl(part.CacheControl),
		}
	}

	return &anthropicModel.MessageContentBlock{
		Type: "image",
		Source: &anthropicModel.ImageSource{
			Type: "url",
			URL:  part.ImageURL.URL,
		},
		CacheControl: convertCacheControl(part.CacheControl),
	}
}

func convertTools(tools []model.Tool) []anthropicModel.Tool {
	result := make([]anthropicModel.Tool, 0, len(tools))
	for _, tool := range tools {
		if tool.Type != "function" {
			continue
		}
		result = append(result, anthropicModel.Tool{
			Name:         tool.Function.Name,
			Description:  tool.Function.Description,
			InputSchema:  tool.Function.Parameters,
			CacheControl: convertCacheControl(tool.CacheControl),
		})
	}
	return result
}

func convertStopSequences(stop *model.Stop) []string {
	if stop == nil {
		return nil
	}
	if stop.Stop != nil {
		return []string{*stop.Stop}
	}
	if len(stop.MultipleStop) > 0 {
		return stop.MultipleStop
	}
	return nil
}

func convertCacheControl(cc *model.CacheControl) *anthropicModel.CacheControl {
	if cc == nil {
		return nil
	}
	return &anthropicModel.CacheControl{
		Type: cc.Type,
		TTL:  cc.TTL,
	}
}

func getThinkingBudget(effort string, budget *int64) int64 {
	if budget != nil {
		return *budget
	}

	switch effort {
	case "low":
		return 1024
	case "medium":
		return 8192
	case "high":
		return 32768
	default:
		return 8192
	}
}

// Response conversion functions

func convertToLLMResponse(resp *anthropicModel.Message) *model.InternalLLMResponse {
	if resp == nil {
		return &model.InternalLLMResponse{
			Object: "chat.completion",
		}
	}

	result := &model.InternalLLMResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Model:   resp.Model,
		Created: 0,
	}

	var (
		content           model.MessageContent
		thinkingText      *string
		thinkingSignature *string
		toolCalls         []model.ToolCall
		textParts         []string
	)

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if block.Text != nil && *block.Text != "" {
				textParts = append(textParts, *block.Text)
				content.MultipleContent = append(content.MultipleContent, model.MessageContentPart{
					Type: "text",
					Text: block.Text,
				})
			}
		case "tool_use":
			if block.ID != "" && block.Name != nil {
				input := "{}"
				if len(block.Input) > 0 {
					input = string(block.Input)
				}
				toolCalls = append(toolCalls, model.ToolCall{
					ID:   block.ID,
					Type: "function",
					Function: model.FunctionCall{
						Name:      *block.Name,
						Arguments: input,
					},
				})
			}
		case "thinking":
			if block.Thinking != nil {
				thinkingText = block.Thinking
			}
			thinkingSignature = block.Signature
		}
	}

	// If we only have text content, use simple string format
	if len(textParts) > 0 && len(content.MultipleContent) == len(textParts) {
		allText := strings.Join(textParts, "")
		content.Content = &allText
		content.MultipleContent = nil
	}

	message := &model.Message{
		Role:               resp.Role,
		Content:            content,
		ToolCalls:          toolCalls,
		ReasoningContent:   thinkingText,
		ReasoningSignature: thinkingSignature,
	}

	choice := model.Choice{
		Index:        0,
		Message:      message,
		FinishReason: convertStopReason(resp.StopReason),
	}

	result.Choices = []model.Choice{choice}
	result.Usage = convertAnthropicUsage(resp.Usage)

	return result
}

func convertStopReason(stopReason *string) *string {
	if stopReason == nil {
		return nil
	}

	switch *stopReason {
	case "end_turn":
		return lo.ToPtr("stop")
	case "max_tokens":
		return lo.ToPtr("length")
	case "stop_sequence", "pause_turn":
		return lo.ToPtr("stop")
	case "tool_use":
		return lo.ToPtr("tool_calls")
	case "refusal":
		return lo.ToPtr("content_filter")
	default:
		return stopReason
	}
}

func convertAnthropicUsage(usage *anthropicModel.Usage) *model.Usage {
	if usage == nil {
		return nil
	}

	result := &model.Usage{
		PromptTokens:             usage.InputTokens,
		CompletionTokens:         usage.OutputTokens,
		TotalTokens:              usage.InputTokens + usage.OutputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens,
		CacheCreationInputTokens: usage.CacheCreationInputTokens,
		AnthropicUsage:           true,
	}

	if usage.CacheReadInputTokens > 0 {
		result.PromptTokensDetails = &model.PromptTokensDetails{
			CachedTokens: usage.CacheReadInputTokens,
		}
	}
	return result
}
