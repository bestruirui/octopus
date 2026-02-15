package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samber/lo"

	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/utils/xurl"
)

// ResponseInbound implements the Inbound interface for OpenAI Responses API.
type ResponseInbound struct {
	// State tracking
	hasResponseCreated      bool
	hasMessageItemStarted   bool
	hasReasoningItemStarted bool
	hasContentPartStarted   bool
	hasFinished             bool
	responseCompleted       bool

	// Response metadata
	responseID string
	model      string
	createdAt  int64

	// Content tracking
	outputIndex    int
	contentIndex   int
	sequenceNumber int
	currentItemID  string

	// Content accumulation
	accumulatedText      strings.Builder
	accumulatedReasoning strings.Builder

	// Tool call tracking
	toolCalls           map[int]*model.ToolCall
	toolCallItemStarted map[int]bool
	toolCallOutputIndex map[int]int

	// Usage tracking
	usage *model.Usage

	// Stream chunks storage for aggregation
	streamChunks []*model.InternalLLMResponse
	// storedResponse stores the non-stream response
	storedResponse *model.InternalLLMResponse
}

func (i *ResponseInbound) TransformRequest(ctx context.Context, body []byte) (*model.InternalLLMRequest, error) {
	var req ResponsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to decode responses api request: %w", err)
	}

	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	return convertToInternalRequest(&req)
}

func (i *ResponseInbound) TransformResponse(ctx context.Context, response *model.InternalLLMResponse) ([]byte, error) {
	if response == nil {
		return nil, fmt.Errorf("response is nil")
	}

	// Store the response for later retrieval
	i.storedResponse = response

	// Convert to Responses API format
	resp := convertToResponsesAPIResponse(response)

	body, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal responses api response: %w", err)
	}

	return body, nil
}

func (i *ResponseInbound) TransformStream(ctx context.Context, stream *model.InternalLLMResponse) ([]byte, error) {
	// Handle [DONE] marker
	if stream.Object == "[DONE]" {
		return []byte("data: [DONE]\n\n"), nil
	}

	// Store the chunk for aggregation
	i.streamChunks = append(i.streamChunks, stream)

	var events [][]byte

	// Initialize tool call tracking maps if needed
	if i.toolCalls == nil {
		i.toolCalls = make(map[int]*model.ToolCall)
		i.toolCallItemStarted = make(map[int]bool)
		i.toolCallOutputIndex = make(map[int]int)
	}

	// Update metadata from chunk
	if i.responseID == "" && stream.ID != "" {
		i.responseID = stream.ID
	}
	if i.model == "" && stream.Model != "" {
		i.model = stream.Model
	}
	if i.createdAt == 0 && stream.Created != 0 {
		i.createdAt = stream.Created
	}
	if stream.Usage != nil {
		i.usage = stream.Usage
	}

	// Generate response.created event if first chunk
	if !i.hasResponseCreated {
		i.hasResponseCreated = true

		response := &ResponsesResponse{
			Object:    "response",
			ID:        i.responseID,
			Model:     i.model,
			CreatedAt: i.createdAt,
			Status:    lo.ToPtr("in_progress"),
			Output:    []ResponsesItem{},
		}

		events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
			Type:     "response.created",
			Response: response,
		}))

		events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
			Type:     "response.in_progress",
			Response: response,
		}))
	}

	// Process choices
	if len(stream.Choices) > 0 {
		choice := stream.Choices[0]

		// Handle reasoning content delta
		if choice.Delta != nil && choice.Delta.ReasoningContent != nil && *choice.Delta.ReasoningContent != "" {
			events = append(events, i.handleReasoningContent(choice.Delta.ReasoningContent)...)
		}

		// Handle text content delta
		if choice.Delta != nil && choice.Delta.Content.Content != nil && *choice.Delta.Content.Content != "" {
			events = append(events, i.handleTextContent(choice.Delta.Content.Content)...)
		}

		// Handle tool calls
		if choice.Delta != nil && len(choice.Delta.ToolCalls) > 0 {
			events = append(events, i.handleToolCalls(choice.Delta.ToolCalls)...)
		}

		// Handle finish reason
		if choice.FinishReason != nil && !i.hasFinished {
			i.hasFinished = true

			// Close any open content parts and output items
			events = append(events, i.closeCurrentContentPart()...)
			events = append(events, i.closeCurrentOutputItem()...)
		}
	}

	// Handle final usage chunk and complete response
	if stream.Usage != nil && i.hasFinished && !i.responseCompleted {
		i.responseCompleted = true
		i.usage = stream.Usage

		status := "completed"
		response := &ResponsesResponse{
			Object:    "response",
			ID:        i.responseID,
			Model:     i.model,
			CreatedAt: i.createdAt,
			Status:    &status,
			Output:    []ResponsesItem{},
			Usage:     convertUsageToResponses(i.usage),
		}

		events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
			Type:     "response.completed",
			Response: response,
		}))
	}

	if len(events) == 0 {
		return nil, nil
	}

	// Join events
	result := make([]byte, 0)
	for _, event := range events {
		if event != nil {
			result = append(result, event...)
		}
	}

	return result, nil
}

func (i *ResponseInbound) enqueueEvent(ev *ResponsesStreamEvent) []byte {
	ev.SequenceNumber = i.sequenceNumber
	i.sequenceNumber++

	data, err := json.Marshal(ev)
	if err != nil {
		return nil
	}

	return formatSSEData(data)
}

func (i *ResponseInbound) handleReasoningContent(content *string) [][]byte {
	var events [][]byte

	// Start reasoning output item if not started
	if !i.hasReasoningItemStarted {
		// Close any previous output item
		events = append(events, i.closeCurrentOutputItem()...)

		i.hasReasoningItemStarted = true
		i.currentItemID = generateItemID()

		item := &ResponsesItem{
			ID:      i.currentItemID,
			Type:    "reasoning",
			Status:  lo.ToPtr("in_progress"),
			Summary: []ResponsesReasoningSummary{},
		}

		events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
			Type:        "response.output_item.added",
			OutputIndex: lo.ToPtr(i.outputIndex),
			Item:        item,
		}))

		// Emit reasoning_summary_part.added
		events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
			Type:         "response.reasoning_summary_part.added",
			ItemID:       &i.currentItemID,
			OutputIndex:  lo.ToPtr(i.outputIndex),
			SummaryIndex: lo.ToPtr(0),
			Part:         &ResponsesContentPart{Type: "summary_text"},
		}))
	}

	// Accumulate reasoning content
	i.accumulatedReasoning.WriteString(*content)

	// Emit reasoning_summary_text.delta
	events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
		Type:         "response.reasoning_summary_text.delta",
		ItemID:       &i.currentItemID,
		OutputIndex:  lo.ToPtr(i.outputIndex),
		SummaryIndex: lo.ToPtr(0),
		Delta:        *content,
	}))

	return events
}

func (i *ResponseInbound) handleTextContent(content *string) [][]byte {
	var events [][]byte

	// Close reasoning item if it was started
	if i.hasReasoningItemStarted {
		events = append(events, i.closeReasoningItem()...)
	}

	// Start message output item if not started
	if !i.hasMessageItemStarted {
		i.hasMessageItemStarted = true
		i.currentItemID = generateItemID()

		events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
			Type:        "response.output_item.added",
			OutputIndex: lo.ToPtr(i.outputIndex),
			Item: &ResponsesItem{
				ID:      i.currentItemID,
				Type:    "message",
				Status:  lo.ToPtr("in_progress"),
				Role:    "assistant",
				Content: &ResponsesInput{Items: []ResponsesItem{}},
			},
		}))
	}

	// Start content part if not started
	if !i.hasContentPartStarted {
		i.hasContentPartStarted = true

		events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
			Type:         "response.content_part.added",
			ItemID:       &i.currentItemID,
			OutputIndex:  lo.ToPtr(i.outputIndex),
			ContentIndex: &i.contentIndex,
			Part: &ResponsesContentPart{
				Type: "output_text",
				Text: lo.ToPtr(""),
			},
		}))
	}

	// Accumulate text content
	i.accumulatedText.WriteString(*content)

	// Emit output_text.delta
	events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
		Type:         "response.output_text.delta",
		ItemID:       &i.currentItemID,
		OutputIndex:  lo.ToPtr(i.outputIndex),
		ContentIndex: &i.contentIndex,
		Delta:        *content,
	}))

	return events
}

func (i *ResponseInbound) handleToolCalls(toolCalls []model.ToolCall) [][]byte {
	var events [][]byte

	// Close message item if it was started
	if i.hasMessageItemStarted {
		events = append(events, i.closeMessageItem()...)
	}

	// Close reasoning item if it was started
	if i.hasReasoningItemStarted {
		events = append(events, i.closeReasoningItem()...)
	}

	for _, tc := range toolCalls {
		toolCallIndex := tc.Index

		// Initialize tool call tracking if needed
		if _, ok := i.toolCalls[toolCallIndex]; !ok {
			events = append(events, i.closeCurrentContentPart()...)
			events = append(events, i.closeCurrentOutputItem()...)

			i.toolCalls[toolCallIndex] = &model.ToolCall{
				Index: toolCallIndex,
				ID:    tc.ID,
				Type:  tc.Type,
				Function: model.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: "",
				},
			}

			itemID := tc.ID
			if itemID == "" {
				itemID = generateItemID()
			}

			item := &ResponsesItem{
				ID:     itemID,
				Type:   "function_call",
				Status: lo.ToPtr("in_progress"),
				CallID: tc.ID,
				Name:   tc.Function.Name,
			}

			events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
				Type:        "response.output_item.added",
				OutputIndex: lo.ToPtr(i.outputIndex),
				Item:        item,
			}))

			i.toolCallItemStarted[toolCallIndex] = true
			i.toolCallOutputIndex[toolCallIndex] = i.outputIndex
			i.currentItemID = itemID
			i.outputIndex++
		}

		// Accumulate arguments
		i.toolCalls[toolCallIndex].Function.Arguments += tc.Function.Arguments

		// Emit function_call_arguments.delta
		if tc.Function.Arguments != "" {
			itemID := i.toolCalls[toolCallIndex].ID
			if itemID == "" {
				itemID = i.currentItemID
			}

			events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
				Type:         "response.function_call_arguments.delta",
				ItemID:       &itemID,
				OutputIndex:  lo.ToPtr(i.outputIndex - 1),
				ContentIndex: lo.ToPtr(0),
				Delta:        tc.Function.Arguments,
			}))
		}
	}

	return events
}

func (i *ResponseInbound) closeReasoningItem() [][]byte {
	if !i.hasReasoningItemStarted {
		return nil
	}

	var events [][]byte
	i.hasReasoningItemStarted = false
	fullReasoning := i.accumulatedReasoning.String()

	// Emit reasoning_summary_text.done
	events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
		Type:         "response.reasoning_summary_text.done",
		ItemID:       &i.currentItemID,
		OutputIndex:  lo.ToPtr(i.outputIndex),
		SummaryIndex: lo.ToPtr(0),
		Text:         fullReasoning,
	}))

	// Emit reasoning_summary_part.done
	events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
		Type:         "response.reasoning_summary_part.done",
		ItemID:       &i.currentItemID,
		OutputIndex:  lo.ToPtr(i.outputIndex),
		SummaryIndex: lo.ToPtr(0),
		Part:         &ResponsesContentPart{Type: "summary_text", Text: &fullReasoning},
	}))

	// Emit output_item.done
	item := ResponsesItem{
		ID:   i.currentItemID,
		Type: "reasoning",
		Summary: []ResponsesReasoningSummary{{
			Type: "summary_text",
			Text: fullReasoning,
		}},
	}

	events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
		Type:        "response.output_item.done",
		OutputIndex: lo.ToPtr(i.outputIndex),
		Item:        &item,
	}))

	i.outputIndex++
	i.accumulatedReasoning.Reset()

	return events
}

func (i *ResponseInbound) closeMessageItem() [][]byte {
	if !i.hasMessageItemStarted {
		return nil
	}

	var events [][]byte
	i.hasMessageItemStarted = false
	fullText := i.accumulatedText.String()

	// Close content part first
	events = append(events, i.closeCurrentContentPart()...)

	// Emit output_item.done
	item := ResponsesItem{
		ID:     i.currentItemID,
		Type:   "message",
		Status: lo.ToPtr("completed"),
		Role:   "assistant",
		Content: &ResponsesInput{
			Items: []ResponsesItem{{
				Type: "output_text",
				Text: &fullText,
			}},
		},
	}

	events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
		Type:        "response.output_item.done",
		OutputIndex: lo.ToPtr(i.outputIndex),
		Item:        &item,
	}))

	i.outputIndex++
	i.contentIndex = 0
	i.accumulatedText.Reset()

	return events
}

func (i *ResponseInbound) closeCurrentContentPart() [][]byte {
	if !i.hasContentPartStarted {
		return nil
	}

	var events [][]byte
	i.hasContentPartStarted = false
	fullText := i.accumulatedText.String()

	// Emit output_text.done
	events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
		Type:         "response.output_text.done",
		ItemID:       &i.currentItemID,
		OutputIndex:  lo.ToPtr(i.outputIndex),
		ContentIndex: &i.contentIndex,
		Text:         fullText,
	}))

	// Emit content_part.done
	events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
		Type:         "response.content_part.done",
		ItemID:       &i.currentItemID,
		OutputIndex:  lo.ToPtr(i.outputIndex),
		ContentIndex: &i.contentIndex,
		Part: &ResponsesContentPart{
			Type: "output_text",
			Text: lo.ToPtr(fullText),
		},
	}))

	return events
}

func (i *ResponseInbound) closeCurrentOutputItem() [][]byte {
	var events [][]byte

	// Close message item if open
	if i.hasMessageItemStarted {
		events = append(events, i.closeMessageItem()...)
	}

	// Close reasoning item if open
	if i.hasReasoningItemStarted {
		events = append(events, i.closeReasoningItem()...)
	}

	// Close any open tool call items
	for idx, tc := range i.toolCalls {
		if i.toolCallItemStarted[idx] {
			itemID := tc.ID
			if itemID == "" {
				itemID = i.currentItemID
			}

			// Emit function_call_arguments.done
			toolCallOutputIdx := i.toolCallOutputIndex[idx]
			events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
				Type:        "response.function_call_arguments.done",
				ItemID:      &itemID,
				OutputIndex: &toolCallOutputIdx,
				Arguments:   tc.Function.Arguments,
			}))

			// Emit output_item.done
			item := ResponsesItem{
				ID:        itemID,
				Type:      "function_call",
				Status:    lo.ToPtr("completed"),
				CallID:    tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}

			events = append(events, i.enqueueEvent(&ResponsesStreamEvent{
				Type:        "response.output_item.done",
				OutputIndex: &toolCallOutputIdx,
				Item:        &item,
			}))

			i.toolCallItemStarted[idx] = false
		}
	}

	return events
}

// GetInternalResponse returns the complete internal response for logging, statistics, etc.
// For streaming: aggregates all stored stream chunks into a complete response
// For non-streaming: returns the stored response
func (i *ResponseInbound) GetInternalResponse(ctx context.Context) (*model.InternalLLMResponse, error) {
	// Return stored response for non-stream scenario
	if i.storedResponse != nil {
		return i.storedResponse, nil
	}

	// Aggregate stream chunks for stream scenario
	if len(i.streamChunks) == 0 {
		return nil, nil
	}

	// Use the first chunk as the base
	firstChunk := i.streamChunks[0]
	result := &model.InternalLLMResponse{
		ID:                firstChunk.ID,
		Object:            "chat.completion",
		Created:           firstChunk.Created,
		Model:             firstChunk.Model,
		SystemFingerprint: firstChunk.SystemFingerprint,
		ServiceTier:       firstChunk.ServiceTier,
	}

	// Aggregate choices by index
	choicesMap := make(map[int]*model.Choice)

	for _, chunk := range i.streamChunks {
		// Update ID and Model if they appear in later chunks
		if chunk.ID != "" {
			result.ID = chunk.ID
		}
		if chunk.Model != "" {
			result.Model = chunk.Model
		}

		// Capture usage from the last chunk that has it
		if chunk.Usage != nil {
			result.Usage = chunk.Usage
		}

		for _, choice := range chunk.Choices {
			existingChoice, exists := choicesMap[choice.Index]
			if !exists {
				existingChoice = &model.Choice{
					Index:   choice.Index,
					Message: &model.Message{},
				}
				choicesMap[choice.Index] = existingChoice
			}

			// Aggregate delta content into message
			if choice.Delta != nil {
				delta := choice.Delta

				// Set role if present
				if delta.Role != "" {
					existingChoice.Message.Role = delta.Role
				}

				// Append content
				if delta.Content.Content != nil {
					if existingChoice.Message.Content.Content == nil {
						existingChoice.Message.Content.Content = new(string)
					}
					*existingChoice.Message.Content.Content += *delta.Content.Content
				}

				// Append reasoning content
				if delta.ReasoningContent != nil {
					if existingChoice.Message.ReasoningContent == nil {
						existingChoice.Message.ReasoningContent = new(string)
					}
					*existingChoice.Message.ReasoningContent += *delta.ReasoningContent
				}

				// Aggregate tool calls
				for _, toolCall := range delta.ToolCalls {
					existingChoice.Message.ToolCalls = mergeToolCall(existingChoice.Message.ToolCalls, toolCall)
				}

				// Set refusal if present
				if delta.Refusal != "" {
					existingChoice.Message.Refusal = delta.Refusal
				}
			}

			// Capture finish reason
			if choice.FinishReason != nil {
				existingChoice.FinishReason = choice.FinishReason
			}

			// Capture logprobs
			if choice.Logprobs != nil {
				if existingChoice.Logprobs == nil {
					existingChoice.Logprobs = &model.LogprobsContent{}
				}
				existingChoice.Logprobs.Content = append(existingChoice.Logprobs.Content, choice.Logprobs.Content...)
			}
		}
	}

	// Convert map to slice, sorted by index
	result.Choices = make([]model.Choice, 0, len(choicesMap))
	for idx := 0; idx < len(choicesMap); idx++ {
		if choice, exists := choicesMap[idx]; exists {
			result.Choices = append(result.Choices, *choice)
		}
	}

	// Clear stored chunks after aggregation
	i.streamChunks = nil

	return result, nil
}

// formatSSEData formats data as SSE data line
func formatSSEData(data []byte) []byte {
	return []byte(fmt.Sprintf("data: %s\n\n", string(data)))
}

// Request types

type ResponsesRequest struct {
	Model             string                `json:"model"`
	Instructions      string                `json:"instructions,omitempty"`
	Input             ResponsesInput        `json:"input"`
	Tools             []ResponsesTool       `json:"tools,omitempty"`
	ToolChoice        *ResponsesToolChoice  `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool                 `json:"parallel_tool_calls,omitempty"`
	Stream            *bool                 `json:"stream,omitempty"`
	Text              *ResponsesTextOptions `json:"text,omitempty"`
	Store             *bool                 `json:"store,omitempty"`
	ServiceTier       *string               `json:"service_tier,omitempty"`
	User              *string               `json:"user,omitempty"`
	Metadata          map[string]string     `json:"metadata,omitempty"`
	MaxOutputTokens   *int64                `json:"max_output_tokens,omitempty"`
	Temperature       *float64              `json:"temperature,omitempty"`
	TopP              *float64              `json:"top_p,omitempty"`
	Reasoning         *ResponsesReasoning   `json:"reasoning,omitempty"`
	Include           []string              `json:"include,omitempty"`
	TopLogprobs       *int64                `json:"top_logprobs,omitempty"`
}

type ResponsesInput struct {
	Text  *string
	Items []ResponsesItem
}

func (i ResponsesInput) MarshalJSON() ([]byte, error) {
	if i.Text != nil {
		return json.Marshal(i.Text)
	}
	return json.Marshal(i.Items)
}

func (i *ResponsesInput) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		i.Text = &text
		return nil
	}
	var items []ResponsesItem
	if err := json.Unmarshal(data, &items); err == nil {
		i.Items = items
		return nil
	}
	return fmt.Errorf("invalid input format")
}

type ResponsesItem struct {
	ID       string          `json:"id,omitempty"`
	Type     string          `json:"type,omitempty"`
	Role     string          `json:"role,omitempty"`
	Content  *ResponsesInput `json:"content,omitempty"`
	Status   *string         `json:"status,omitempty"`
	Text     *string         `json:"text,omitempty"`
	ImageURL *string         `json:"image_url,omitempty"`
	Detail   *string         `json:"detail,omitempty"`

	// Annotations for output_text content
	Annotations *[]ResponsesAnnotation `json:"annotations,omitempty"`

	// Function call fields
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`

	// Function call output
	Output *ResponsesInput `json:"output,omitempty"`

	// Image generation fields
	Result       *string `json:"result,omitempty"`
	Background   *string `json:"background,omitempty"`
	OutputFormat *string `json:"output_format,omitempty"`
	Quality      *string `json:"quality,omitempty"`
	Size         *string `json:"size,omitempty"`

	// Reasoning fields
	Summary          []ResponsesReasoningSummary `json:"summary,omitempty"`
	EncryptedContent *string                     `json:"encrypted_content,omitempty"`
}

func (item ResponsesItem) isOutputMessageContent() bool {
	if item.Content == nil || len(item.Content.Items) == 0 {
		return false
	}
	for _, ci := range item.Content.Items {
		if ci.Type == "output_text" {
			return true
		}
	}
	return false
}

func (item ResponsesItem) GetContentItems() []ResponsesContentItem {
	if item.Content == nil || len(item.Content.Items) == 0 {
		return nil
	}
	result := make([]ResponsesContentItem, 0, len(item.Content.Items))
	for _, ci := range item.Content.Items {
		text := ""
		if ci.Text != nil {
			text = *ci.Text
		}
		result = append(result, ResponsesContentItem{
			Type: ci.Type,
			Text: text,
		})
	}
	return result
}

type ResponsesContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ResponsesReasoningSummary struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ResponsesAnnotation struct {
	Type       string  `json:"type"`
	StartIndex *int    `json:"start_index,omitempty"`
	EndIndex   *int    `json:"end_index,omitempty"`
	URL        *string `json:"url,omitempty"`
	Title      *string `json:"title,omitempty"`
	FileID     *string `json:"file_id,omitempty"`
	Filename   *string `json:"filename,omitempty"`
}

type ResponsesTool struct {
	Type              string         `json:"type,omitempty"`
	Name              string         `json:"name,omitempty"`
	Description       string         `json:"description,omitempty"`
	Parameters        map[string]any `json:"parameters,omitempty"`
	Strict            *bool          `json:"strict,omitempty"`
	Background        string         `json:"background,omitempty"`
	OutputFormat      string         `json:"output_format,omitempty"`
	Quality           string         `json:"quality,omitempty"`
	Size              string         `json:"size,omitempty"`
	OutputCompression *int64         `json:"output_compression,omitempty"`
}

type ResponsesToolChoice struct {
	Mode *string `json:"mode,omitempty"`
	Type *string `json:"type,omitempty"`
	Name *string `json:"name,omitempty"`
}

func (t *ResponsesToolChoice) UnmarshalJSON(data []byte) error {
	var mode string
	if err := json.Unmarshal(data, &mode); err == nil {
		t.Mode = &mode
		return nil
	}

	type Alias ResponsesToolChoice
	var alias Alias
	if err := json.Unmarshal(data, &alias); err == nil {
		*t = ResponsesToolChoice(alias)
		return nil
	}

	return fmt.Errorf("invalid tool choice format")
}

type ResponsesTextOptions struct {
	Format    *ResponsesTextFormat `json:"format,omitempty"`
	Verbosity *string              `json:"verbosity,omitempty"`
}

type ResponsesTextFormat struct {
	Type   string          `json:"type,omitempty"`
	Name   string          `json:"name,omitempty"`
	Schema json.RawMessage `json:"schema,omitempty"`
}

type ResponsesReasoning struct {
	Effort    string `json:"effort,omitempty"`
	MaxTokens *int64 `json:"max_tokens,omitempty"`
}

// Response types

type ResponsesResponse struct {
	Object    string          `json:"object"`
	ID        string          `json:"id"`
	Model     string          `json:"model"`
	CreatedAt int64           `json:"created_at"`
	Output    []ResponsesItem `json:"output"`
	Status    *string         `json:"status,omitempty"`
	Usage     *ResponsesUsage `json:"usage,omitempty"`
	Error     *ResponsesError `json:"error,omitempty"`
}

type ResponsesUsage struct {
	InputTokens       int64 `json:"input_tokens"`
	InputTokenDetails struct {
		CachedTokens int64 `json:"cached_tokens"`
	} `json:"input_tokens_details"`
	OutputTokens       int64 `json:"output_tokens"`
	OutputTokenDetails struct {
		ReasoningTokens int64 `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
	TotalTokens int64 `json:"total_tokens"`
}

type ResponsesError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ResponsesStreamEvent struct {
	Type           string                `json:"type"`
	SequenceNumber int                   `json:"sequence_number"`
	Response       *ResponsesResponse    `json:"response,omitempty"`
	OutputIndex    *int                  `json:"output_index,omitempty"`
	Item           *ResponsesItem        `json:"item,omitempty"`
	ItemID         *string               `json:"item_id,omitempty"`
	ContentIndex   *int                  `json:"content_index,omitempty"`
	Delta          string                `json:"delta,omitempty"`
	Text           string                `json:"text,omitempty"`
	Name           string                `json:"name,omitempty"`
	CallID         string                `json:"call_id,omitempty"`
	Arguments      string                `json:"arguments,omitempty"`
	SummaryIndex   *int                  `json:"summary_index,omitempty"`
	Part           *ResponsesContentPart `json:"part,omitempty"`
}

type ResponsesContentPart struct {
	Type        string                `json:"type"`
	Text        *string               `json:"text,omitempty"`
	Annotations []ResponsesAnnotation `json:"annotations,omitempty"`
}

// Conversion functions

func convertToInternalRequest(req *ResponsesRequest) (*model.InternalLLMRequest, error) {
	chatReq := &model.InternalLLMRequest{
		Model:               req.Model,
		Temperature:         req.Temperature,
		TopP:                req.TopP,
		Stream:              req.Stream,
		Store:               req.Store,
		ServiceTier:         req.ServiceTier,
		User:                req.User,
		Metadata:            req.Metadata,
		MaxCompletionTokens: req.MaxOutputTokens,
		TopLogprobs:         req.TopLogprobs,
		ParallelToolCalls:   req.ParallelToolCalls,
		RawAPIFormat:        model.APIFormatOpenAIResponse,
		TransformerMetadata: map[string]string{},
		Include:             append([]string(nil), req.Include...),
	}

	if req.Input.Text == nil && len(req.Input.Items) > 0 {
		chatReq.TransformOptions.ArrayInputs = lo.ToPtr(true)
	}

	// Convert reasoning
	if req.Reasoning != nil {
		if req.Reasoning.Effort != "" {
			chatReq.ReasoningEffort = req.Reasoning.Effort
		}
		if req.Reasoning.MaxTokens != nil {
			chatReq.ReasoningBudget = req.Reasoning.MaxTokens
		}
	}

	// Convert tool choice
	if req.ToolChoice != nil {
		chatReq.ToolChoice = convertToolChoiceToInternal(req.ToolChoice)
	}

	// Convert instructions to system message
	messages := make([]model.Message, 0)
	if req.Instructions != "" {
		messages = append(messages, model.Message{
			Role: "system",
			Content: model.MessageContent{
				Content: lo.ToPtr(req.Instructions),
			},
		})
	}

	// Convert input to messages
	inputMessages, err := convertInputToMessages(&req.Input)
	if err != nil {
		return nil, err
	}
	messages = append(messages, inputMessages...)
	chatReq.Messages = messages

	// Convert tools
	if len(req.Tools) > 0 {
		tools, err := convertToolsToInternal(req.Tools)
		if err != nil {
			return nil, err
		}
		chatReq.Tools = tools
	}

	// Convert text format
	if req.Text != nil && req.Text.Format != nil && req.Text.Format.Type != "" {
		chatReq.ResponseFormat = &model.ResponseFormat{
			Type: req.Text.Format.Type,
		}
	}

	return chatReq, nil
}

func convertToolChoiceToInternal(src *ResponsesToolChoice) *model.ToolChoice {
	if src == nil {
		return nil
	}

	result := &model.ToolChoice{}
	if src.Mode != nil {
		result.ToolChoice = src.Mode
	} else if src.Type != nil && src.Name != nil {
		result.NamedToolChoice = &model.NamedToolChoice{
			Type: *src.Type,
			Function: model.ToolFunction{
				Name: *src.Name,
			},
		}
	}
	return result
}

func convertInputToMessages(input *ResponsesInput) ([]model.Message, error) {
	if input == nil {
		return nil, nil
	}

	// Simple text input
	if input.Text != nil {
		return []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					Content: input.Text,
				},
			},
		}, nil
	}

	// Array of items
	messages := make([]model.Message, 0, len(input.Items))
	for _, item := range input.Items {
		msg, err := convertItemToMessage(&item)
		if err != nil {
			return nil, err
		}
		if msg != nil {
			messages = append(messages, *msg)
		}
	}

	return messages, nil
}

func convertItemToMessage(item *ResponsesItem) (*model.Message, error) {
	if item == nil {
		return nil, nil
	}

	switch item.Type {
	case "message", "input_text", "":
		msg := &model.Message{
			Role: item.Role,
		}

		if item.Content != nil && len(item.Content.Items) > 0 && item.isOutputMessageContent() {
			msg.Content = convertContentItemsToMessageContent(item.GetContentItems())
		} else if item.Content != nil {
			msg.Content = convertInputToMessageContent(*item.Content)
		} else if item.Text != nil {
			msg.Content = model.MessageContent{Content: item.Text}
		}

		return msg, nil

	case "input_image":
		if item.ImageURL != nil {
			return &model.Message{
				Role: lo.Ternary(item.Role != "", item.Role, "user"),
				Content: model.MessageContent{
					MultipleContent: []model.MessageContentPart{
						{
							Type: "image_url",
							ImageURL: &model.ImageURL{
								URL:    *item.ImageURL,
								Detail: item.Detail,
							},
						},
					},
				},
			}, nil
		}
		return nil, nil

	case "function_call":
		callID := item.CallID
		if callID == "" {
			callID = item.ID
		}

		return &model.Message{
			Role: "assistant",
			ToolCalls: []model.ToolCall{
				{
					ID:   callID,
					Type: "function",
					Function: model.FunctionCall{
						Name:      item.Name,
						Arguments: item.Arguments,
					},
				},
			},
		}, nil

	case "function_call_output":
		callID := item.CallID
		if callID == "" {
			callID = item.ID
		}

		var toolCallID *string
		if callID != "" {
			toolCallID = lo.ToPtr(callID)
		}

		return &model.Message{
			Role:       "tool",
			ToolCallID: toolCallID,
			Content:    convertInputToMessageContent(*item.Output),
		}, nil

	case "reasoning":
		msg := &model.Message{
			Role: "assistant",
		}

		var reasoningText strings.Builder
		for _, summary := range item.Summary {
			reasoningText.WriteString(summary.Text)
		}

		if reasoningText.Len() > 0 {
			msg.ReasoningContent = lo.ToPtr(reasoningText.String())
		}

		if item.EncryptedContent != nil && *item.EncryptedContent != "" {
			msg.ReasoningSignature = item.EncryptedContent
		}

		return msg, nil

	default:
		return nil, nil
	}
}

func convertInputToMessageContent(input ResponsesInput) model.MessageContent {
	if input.Text != nil {
		return model.MessageContent{Content: input.Text}
	}

	parts := make([]model.MessageContentPart, 0, len(input.Items))
	for _, item := range input.Items {
		switch item.Type {
		case "input_text", "text", "output_text":
			if item.Text != nil {
				parts = append(parts, model.MessageContentPart{
					Type: "text",
					Text: item.Text,
				})
			}
		case "input_image":
			if item.ImageURL != nil {
				parts = append(parts, model.MessageContentPart{
					Type: "image_url",
					ImageURL: &model.ImageURL{
						URL:    *item.ImageURL,
						Detail: item.Detail,
					},
				})
			}
		}
	}

	if len(parts) == 1 && parts[0].Type == "text" && parts[0].Text != nil {
		return model.MessageContent{Content: parts[0].Text}
	}

	return model.MessageContent{MultipleContent: parts}
}

func convertContentItemsToMessageContent(items []ResponsesContentItem) model.MessageContent {
	if len(items) == 1 && (items[0].Type == "output_text" || items[0].Type == "input_text" || items[0].Type == "text") {
		return model.MessageContent{Content: lo.ToPtr(items[0].Text)}
	}

	parts := make([]model.MessageContentPart, 0, len(items))
	for _, item := range items {
		switch item.Type {
		case "output_text", "input_text", "text":
			parts = append(parts, model.MessageContentPart{
				Type: "text",
				Text: lo.ToPtr(item.Text),
			})
		}
	}

	return model.MessageContent{MultipleContent: parts}
}

func convertToolsToInternal(tools []ResponsesTool) ([]model.Tool, error) {
	result := make([]model.Tool, 0, len(tools))

	for _, tool := range tools {
		switch tool.Type {
		case "function":
			params, err := json.Marshal(tool.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function parameters: %w", err)
			}

			result = append(result, model.Tool{
				Type: "function",
				Function: model.Function{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  params,
					Strict:      tool.Strict,
				},
			})

		case "image_generation":
			result = append(result, model.Tool{
				Type: "image_generation",
				ImageGeneration: &model.ImageGeneration{
					Background:        tool.Background,
					OutputFormat:      tool.OutputFormat,
					Quality:           tool.Quality,
					Size:              tool.Size,
					OutputCompression: tool.OutputCompression,
				},
			})
		}
	}

	return result, nil
}

func convertToResponsesAPIResponse(resp *model.InternalLLMResponse) *ResponsesResponse {
	result := &ResponsesResponse{
		Object:    "response",
		ID:        resp.ID,
		Model:     resp.Model,
		CreatedAt: resp.Created,
		Output:    make([]ResponsesItem, 0),
		Status:    lo.ToPtr("completed"),
	}

	// Convert usage
	result.Usage = convertUsageToResponses(resp.Usage)

	// Convert choices to output items
	for _, choice := range resp.Choices {
		var message *model.Message
		if choice.Message != nil {
			message = choice.Message
		} else if choice.Delta != nil {
			message = choice.Delta
		}

		if message == nil {
			continue
		}

		// Handle reasoning content
		if message.ReasoningContent != nil && *message.ReasoningContent != "" {
			result.Output = append(result.Output, ResponsesItem{
				ID:     generateItemID(),
				Type:   "reasoning",
				Status: lo.ToPtr("completed"),
				Summary: []ResponsesReasoningSummary{
					{
						Type: "summary_text",
						Text: *message.ReasoningContent,
					},
				},
			})
		}

		// Handle tool calls
		if len(message.ToolCalls) > 0 {
			for _, toolCall := range message.ToolCalls {
				result.Output = append(result.Output, ResponsesItem{
					ID:        toolCall.ID,
					Type:      "function_call",
					CallID:    toolCall.ID,
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
					Status:    lo.ToPtr("completed"),
				})
			}
		}

		// Handle text content
		if message.Content.Content != nil && *message.Content.Content != "" {
			text := *message.Content.Content
			result.Output = append(result.Output, ResponsesItem{
				ID:   generateItemID(),
				Type: "message",
				Role: "assistant",
				Content: &ResponsesInput{
					Items: []ResponsesItem{
						{
							Type:        "output_text",
							Text:        &text,
							Annotations: &[]ResponsesAnnotation{},
						},
					},
				},
				Status: lo.ToPtr("completed"),
			})
		} else if len(message.Content.MultipleContent) > 0 {
			contentItems := make([]ResponsesItem, 0)

			for _, part := range message.Content.MultipleContent {
				switch part.Type {
				case "text":
					if part.Text != nil {
						text := *part.Text
						contentItems = append(contentItems, ResponsesItem{
							Type:        "output_text",
							Text:        &text,
							Annotations: &[]ResponsesAnnotation{},
						})
					}
				case "image_url":
					if part.ImageURL != nil {
						result.Output = append(result.Output, ResponsesItem{
							ID:     generateItemID(),
							Type:   "image_generation_call",
							Role:   "assistant",
							Result: lo.ToPtr(xurl.ExtractBase64FromDataURL(part.ImageURL.URL)),
							Status: lo.ToPtr("completed"),
						})
					}
				}
			}

			if len(contentItems) > 0 {
				result.Output = append(result.Output, ResponsesItem{
					ID:      generateItemID(),
					Type:    "message",
					Role:    "assistant",
					Content: &ResponsesInput{Items: contentItems},
					Status:  lo.ToPtr("completed"),
				})
			}
		}

		// Set status based on finish reason
		if choice.FinishReason != nil {
			switch *choice.FinishReason {
			case "stop":
				result.Status = lo.ToPtr("completed")
			case "length":
				result.Status = lo.ToPtr("incomplete")
			case "tool_calls":
				result.Status = lo.ToPtr("completed")
			case "error":
				result.Status = lo.ToPtr("failed")
			}
		}
	}

	// If no output items, create empty message
	if len(result.Output) == 0 {
		emptyText := ""
		result.Output = []ResponsesItem{
			{
				ID:   generateItemID(),
				Type: "message",
				Role: "assistant",
				Content: &ResponsesInput{
					Items: []ResponsesItem{
						{
							Type: "output_text",
							Text: &emptyText,
						},
					},
				},
				Status: lo.ToPtr("completed"),
			},
		}
	}

	return result
}

func convertUsageToResponses(usage *model.Usage) *ResponsesUsage {
	if usage == nil {
		return nil
	}

	result := &ResponsesUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.TotalTokens,
	}

	if usage.PromptTokensDetails != nil {
		result.InputTokenDetails.CachedTokens = usage.PromptTokensDetails.CachedTokens
	}

	if usage.CompletionTokensDetails != nil {
		result.OutputTokenDetails.ReasoningTokens = usage.CompletionTokensDetails.ReasoningTokens
	}

	return result
}

func generateItemID() string {
	return fmt.Sprintf("item_%s", lo.RandomString(16, lo.AlphanumericCharset))
}
