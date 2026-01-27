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

	"github.com/samber/lo"

	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/utils/log"
)

// ResponseOutbound implements the Outbound interface for OpenAI Responses API.
type ResponseOutbound struct {
	// Stream state tracking
	streamID    string
	streamModel string
	initialized bool
}

func (o *ResponseOutbound) TransformRequest(ctx context.Context, request *model.InternalLLMRequest, baseUrl, key string) (*http.Request, error) {
	if request == nil {
		return nil, fmt.Errorf("request is nil")
	}

	// Convert to Responses API request format
	responsesReq := ConvertToResponsesRequest(request)

	body, err := json.Marshal(responsesReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal responses api request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	// Parse and set URL
	parsedUrl, err := url.Parse(strings.TrimSuffix(baseUrl, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse base url: %w", err)
	}
	parsedUrl.Path = parsedUrl.Path + "/responses"
	req.URL = parsedUrl
	req.Method = http.MethodPost

	return req, nil
}

func (o *ResponseOutbound) TransformResponse(ctx context.Context, response *http.Response) (*model.InternalLLMResponse, error) {
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
		var errResp struct {
			Error model.ErrorDetail `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, &model.ResponseError{
				StatusCode: response.StatusCode,
				Detail:     errResp.Error,
			}
		}
		return nil, fmt.Errorf("HTTP error %d: %s", response.StatusCode, string(body))
	}

	var resp ResponsesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal responses api response: %w", err)
	}

	// Convert to internal response
	return convertToLLMResponseFromResponses(&resp), nil
}

func (o *ResponseOutbound) TransformStream(ctx context.Context, eventData []byte) (*model.InternalLLMResponse, error) {
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
		o.initialized = true
	}

	// Parse the streaming event
	var streamEvent ResponsesStreamEvent
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
	case "response.created", "response.in_progress":
		if streamEvent.Response != nil {
			o.streamID = streamEvent.Response.ID
			o.streamModel = streamEvent.Response.Model
			resp.ID = o.streamID
			resp.Model = o.streamModel
		}
		resp.Choices = []model.Choice{
			{
				Index: 0,
				Delta: &model.Message{
					Role: "assistant",
				},
			},
		}

	case "response.output_text.delta":
		resp.Choices = []model.Choice{
			{
				Index: 0,
				Delta: &model.Message{
					Role: "assistant",
					Content: model.MessageContent{
						Content: lo.ToPtr(streamEvent.Delta),
					},
				},
			},
		}

	case "response.function_call_arguments.delta":
		resp.Choices = []model.Choice{
			{
				Index: 0,
				Delta: &model.Message{
					Role: "assistant",
					ToolCalls: []model.ToolCall{
						{
							Index: streamEvent.OutputIndex,
							ID:    streamEvent.CallID,
							Type:  "function",
							Function: model.FunctionCall{
								Name:      streamEvent.Name,
								Arguments: streamEvent.Delta,
							},
						},
					},
				},
			},
		}

	case "response.output_item.added":
		if streamEvent.Item != nil && streamEvent.Item.Type == "function_call" {
			resp.Choices = []model.Choice{
				{
					Index: 0,
					Delta: &model.Message{
						Role: "assistant",
						ToolCalls: []model.ToolCall{
							{
								Index: streamEvent.OutputIndex,
								ID:    streamEvent.Item.CallID,
								Type:  "function",
								Function: model.FunctionCall{
									Name:      streamEvent.Item.Name,
									Arguments: "{}",
								},
							},
						},
					},
				},
			}
		} else {
			return nil, nil
		}

	case "response.reasoning_summary_text.delta":
		resp.Choices = []model.Choice{
			{
				Index: 0,
				Delta: &model.Message{
					Role:             "assistant",
					ReasoningContent: lo.ToPtr(streamEvent.Delta),
				},
			},
		}

	case "response.completed":
		if streamEvent.Response != nil {
			var finishReason *string
			if streamEvent.Response.Status != nil {
				switch *streamEvent.Response.Status {
				case "completed":
					finishReason = lo.ToPtr("stop")
				case "incomplete":
					finishReason = lo.ToPtr("length")
				case "failed":
					finishReason = lo.ToPtr("error")
				}
			}
			resp.Choices = []model.Choice{
				{
					Index:        0,
					FinishReason: finishReason,
				},
			}
			if streamEvent.Response.Usage != nil {
				resp.Usage = convertResponsesUsage(streamEvent.Response.Usage)
			}
		}

	case "response.failed", "response.incomplete", "error":
		resp.Choices = []model.Choice{
			{
				Index:        0,
				FinishReason: lo.ToPtr("error"),
			},
		}

	default:
		// Skip unhandled events
		return nil, nil
	}

	return resp, nil
}

// ResponsesRequest represents the OpenAI Responses API request format.
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
	Annotations []ResponsesAnnotation `json:"annotations,omitempty"`

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
	Summary []ResponsesReasoningSummary `json:"summary,omitempty"`
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

func (t ResponsesToolChoice) MarshalJSON() ([]byte, error) {
	// If only Mode is set and it's a simple mode like "auto", "none", "required"
	if t.Mode != nil && t.Type == nil && t.Name == nil {
		return json.Marshal(*t.Mode)
	}
	// Otherwise, serialize as an object
	type Alias ResponsesToolChoice
	return json.Marshal(Alias(t))
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
	Effort string `json:"effort,omitempty"`
}

// ResponsesResponse represents the OpenAI Responses API response format.
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
	Type           string             `json:"type"`
	SequenceNumber int                `json:"sequence_number"`
	Response       *ResponsesResponse `json:"response,omitempty"`
	OutputIndex    int                `json:"output_index"`
	Item           *ResponsesItem     `json:"item,omitempty"`
	ItemID         *string            `json:"item_id,omitempty"`
	ContentIndex   *int               `json:"content_index,omitempty"`
	Delta          string             `json:"delta,omitempty"`
	Text           string             `json:"text,omitempty"`
	Name           string             `json:"name,omitempty"`
	CallID         string             `json:"call_id,omitempty"`
	Arguments      string             `json:"arguments,omitempty"`
	SummaryIndex   *int               `json:"summary_index,omitempty"`
	Code           string             `json:"code,omitempty"`
	Message        string             `json:"message,omitempty"`
}

// Conversion functions

func ConvertToResponsesRequest(req *model.InternalLLMRequest) *ResponsesRequest {
	result := &ResponsesRequest{
		Model:             req.Model,
		Temperature:       req.Temperature,
		TopP:              req.TopP,
		Stream:            req.Stream,
		Store:             req.Store,
		ServiceTier:       req.ServiceTier,
		User:              req.User,
		Metadata:          req.Metadata,
		MaxOutputTokens:   req.MaxCompletionTokens,
		ParallelToolCalls: req.ParallelToolCalls,
	}

	// Convert instructions from system messages
	result.Instructions = convertInstructionsFromMessages(req.Messages)

	// Convert input from messages
	result.Input = convertInputFromMessages(req.Messages)

	// Convert tools
	if len(req.Tools) > 0 {
		result.Tools = convertToolsToResponses(req.Tools)
	}

	// Convert tool choice
	if req.ToolChoice != nil {
		result.ToolChoice = convertToolChoiceToResponses(req.ToolChoice)
		log.Infof("tool choice %v", result.ToolChoice)
	}

	// Convert text options
	if req.ResponseFormat != nil {
		result.Text = &ResponsesTextOptions{
			Format: &ResponsesTextFormat{
				Type: req.ResponseFormat.Type,
			},
		}
	}

	// Convert reasoning
	if req.ReasoningEffort != "" || req.ReasoningBudget != nil {
		result.Reasoning = &ResponsesReasoning{
			Effort: req.ReasoningEffort,
		}
	}

	return result
}

func convertInstructionsFromMessages(msgs []model.Message) string {
	var instructions []string
	for _, msg := range msgs {
		if msg.Role != "system" && msg.Role != "developer" {
			continue
		}
		if msg.Content.Content != nil {
			instructions = append(instructions, *msg.Content.Content)
		}
		if len(msg.Content.MultipleContent) > 0 {
			var sb strings.Builder
			for _, p := range msg.Content.MultipleContent {
				if p.Type == "text" && p.Text != nil {
					if sb.Len() > 0 {
						sb.WriteString("\n")
					}
					sb.WriteString(*p.Text)
				}
			}
			if sb.Len() > 0 {
				instructions = append(instructions, sb.String())
			}
		}
	}
	return strings.Join(instructions, "\n")
}

func convertInputFromMessages(msgs []model.Message) ResponsesInput {
	if len(msgs) == 0 {
		return ResponsesInput{}
	}

	// Check for simple single user message
	nonSystemMsgs := make([]model.Message, 0)
	for _, msg := range msgs {
		if msg.Role != "system" && msg.Role != "developer" {
			nonSystemMsgs = append(nonSystemMsgs, msg)
		}
	}

	if len(nonSystemMsgs) == 1 && nonSystemMsgs[0].Content.Content != nil && nonSystemMsgs[0].Role == "user" {
		return ResponsesInput{Text: nonSystemMsgs[0].Content.Content}
	}

	var items []ResponsesItem
	for _, msg := range msgs {
		switch msg.Role {
		case "system", "developer":
			continue
		case "user":
			items = append(items, convertUserMessageToResponses(msg))
		case "assistant":
			items = append(items, convertAssistantMessageToResponses(msg)...)
		case "tool":
			items = append(items, convertToolMessageToResponses(msg))
		}
	}

	return ResponsesInput{Items: items}
}

func convertUserMessageToResponses(msg model.Message) ResponsesItem {
	var contentItems []ResponsesItem

	if msg.Content.Content != nil {
		contentItems = append(contentItems, ResponsesItem{
			Type: "input_text",
			Text: msg.Content.Content,
		})
	} else {
		for _, p := range msg.Content.MultipleContent {
			switch p.Type {
			case "text":
				if p.Text != nil {
					contentItems = append(contentItems, ResponsesItem{
						Type: "input_text",
						Text: p.Text,
					})
				}
			case "image_url":
				if p.ImageURL != nil {
					contentItems = append(contentItems, ResponsesItem{
						Type:     "input_image",
						ImageURL: &p.ImageURL.URL,
						Detail:   p.ImageURL.Detail,
					})
				}
			}
		}
	}

	return ResponsesItem{
		Role:    msg.Role,
		Content: &ResponsesInput{Items: contentItems},
	}
}

func convertAssistantMessageToResponses(msg model.Message) []ResponsesItem {
	var items []ResponsesItem

	// Handle tool calls
	for _, tc := range msg.ToolCalls {
		items = append(items, ResponsesItem{
			Type:      "function_call",
			CallID:    tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	// Handle content
	var contentItems []ResponsesItem
	if msg.Content.Content != nil {
		contentItems = append(contentItems, ResponsesItem{
			Type: "output_text",
			Text: msg.Content.Content,
		})
	} else {
		for _, p := range msg.Content.MultipleContent {
			if p.Type == "text" && p.Text != nil {
				contentItems = append(contentItems, ResponsesItem{
					Type: "output_text",
					Text: p.Text,
				})
			}
		}
	}

	if len(contentItems) > 0 {
		items = append(items, ResponsesItem{
			Type:    "message",
			Role:    msg.Role,
			Status:  lo.ToPtr("completed"),
			Content: &ResponsesInput{Items: contentItems},
		})
	}

	return items
}

func convertToolMessageToResponses(msg model.Message) ResponsesItem {
	var output ResponsesInput

	if msg.Content.Content != nil {
		output.Text = msg.Content.Content
	} else if len(msg.Content.MultipleContent) > 0 {
		for _, p := range msg.Content.MultipleContent {
			if p.Type == "text" && p.Text != nil {
				output.Items = append(output.Items, ResponsesItem{
					Type: "input_text",
					Text: p.Text,
				})
			}
		}
	}

	if output.Text == nil && len(output.Items) == 0 {
		output.Text = lo.ToPtr("")
	}

	return ResponsesItem{
		Type:   "function_call_output",
		CallID: lo.FromPtr(msg.ToolCallID),
		Output: &output,
	}
}

func convertToolsToResponses(tools []model.Tool) []ResponsesTool {
	result := make([]ResponsesTool, 0, len(tools))
	for _, tool := range tools {
		switch tool.Type {
		case "function":
			rt := ResponsesTool{
				Type:        "function",
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Strict:      tool.Function.Strict,
			}
			if len(tool.Function.Parameters) > 0 {
				var params map[string]any
				if err := json.Unmarshal(tool.Function.Parameters, &params); err == nil {
					rt.Parameters = params
				}
			}
			result = append(result, rt)
		case "image_generation":
			rt := ResponsesTool{
				Type: "image_generation",
			}
			if tool.ImageGeneration != nil {
				rt.Background = tool.ImageGeneration.Background
				rt.OutputFormat = tool.ImageGeneration.OutputFormat
				rt.Quality = tool.ImageGeneration.Quality
				rt.Size = tool.ImageGeneration.Size
				rt.OutputCompression = tool.ImageGeneration.OutputCompression
			}
			result = append(result, rt)
		}
	}
	return result
}

func convertToolChoiceToResponses(tc *model.ToolChoice) *ResponsesToolChoice {
	if tc == nil {
		return nil
	}

	result := &ResponsesToolChoice{}
	if tc.ToolChoice != nil {
		result.Mode = tc.ToolChoice
	} else if tc.NamedToolChoice != nil {
		result.Type = &tc.NamedToolChoice.Type
		result.Name = &tc.NamedToolChoice.Function.Name
	}
	return result
}

func convertToLLMResponseFromResponses(resp *ResponsesResponse) *model.InternalLLMResponse {
	if resp == nil {
		return &model.InternalLLMResponse{
			Object: "chat.completion",
		}
	}

	result := &model.InternalLLMResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Model:   resp.Model,
		Created: resp.CreatedAt,
	}

	var (
		contentParts     []model.MessageContentPart
		textContent      strings.Builder
		reasoningContent strings.Builder
		toolCalls        []model.ToolCall
	)

	for _, outputItem := range resp.Output {
		switch outputItem.Type {
		case "message":
			if outputItem.Content != nil {
				for _, item := range outputItem.Content.Items {
					if item.Type == "output_text" && item.Text != nil {
						textContent.WriteString(*item.Text)
					}
				}
			}
		case "output_text":
			if outputItem.Text != nil {
				textContent.WriteString(*outputItem.Text)
			}
		case "function_call":
			toolCalls = append(toolCalls, model.ToolCall{
				ID:   outputItem.CallID,
				Type: "function",
				Function: model.FunctionCall{
					Name:      outputItem.Name,
					Arguments: outputItem.Arguments,
				},
			})
		case "reasoning":
			for _, summary := range outputItem.Summary {
				reasoningContent.WriteString(summary.Text)
			}
		case "image_generation_call":
			if outputItem.Result != nil && *outputItem.Result != "" {
				outputFormat := "png"
				if outputItem.OutputFormat != nil {
					outputFormat = *outputItem.OutputFormat
				}
				contentParts = append(contentParts, model.MessageContentPart{
					Type: "image_url",
					ImageURL: &model.ImageURL{
						URL: "data:image/" + outputFormat + ";base64," + *outputItem.Result,
					},
				})
			}
		}
	}

	choice := model.Choice{
		Index: 0,
		Message: &model.Message{
			Role:      "assistant",
			ToolCalls: toolCalls,
		},
	}

	// Set reasoning content if present
	if reasoningContent.Len() > 0 {
		choice.Message.ReasoningContent = lo.ToPtr(reasoningContent.String())
	}

	// Set message content
	if textContent.Len() > 0 {
		if len(contentParts) > 0 {
			textPart := model.MessageContentPart{
				Type: "text",
				Text: lo.ToPtr(textContent.String()),
			}
			contentParts = append([]model.MessageContentPart{textPart}, contentParts...)
			choice.Message.Content = model.MessageContent{
				MultipleContent: contentParts,
			}
		} else {
			choice.Message.Content = model.MessageContent{
				Content: lo.ToPtr(textContent.String()),
			}
		}
	} else if len(contentParts) > 0 {
		choice.Message.Content = model.MessageContent{
			MultipleContent: contentParts,
		}
	}

	// Set finish reason based on status
	if len(toolCalls) > 0 {
		choice.FinishReason = lo.ToPtr("tool_calls")
	} else if resp.Status != nil {
		switch *resp.Status {
		case "completed":
			choice.FinishReason = lo.ToPtr("stop")
		case "failed":
			choice.FinishReason = lo.ToPtr("error")
		case "incomplete":
			choice.FinishReason = lo.ToPtr("length")
		}
	}

	result.Choices = []model.Choice{choice}
	result.Usage = convertResponsesUsage(resp.Usage)

	return result
}

func convertResponsesUsage(usage *ResponsesUsage) *model.Usage {
	if usage == nil {
		return nil
	}

	result := &model.Usage{
		PromptTokens:     usage.InputTokens,
		CompletionTokens: usage.OutputTokens,
		TotalTokens:      usage.TotalTokens,
	}

	if usage.InputTokenDetails.CachedTokens > 0 {
		result.PromptTokensDetails = &model.PromptTokensDetails{
			CachedTokens: usage.InputTokenDetails.CachedTokens,
		}
	}

	if usage.OutputTokenDetails.ReasoningTokens > 0 {
		result.CompletionTokensDetails = &model.CompletionTokensDetails{
			ReasoningTokens: usage.OutputTokenDetails.ReasoningTokens,
		}
	}

	return result
}
