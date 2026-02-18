package openai

import (
	"github.com/samber/lo"

	"github.com/bestruirui/octopus/internal/transformer/model"
)

// convertResponsesInputToChatMessages is the dedicated conversion-layer entry for:
// OpenAI Responses input -> OpenAI Chat-style internal messages.
func convertResponsesInputToChatMessages(input *ResponsesInput) ([]model.Message, error) {
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
	return convertResponsesItemsToChatMessages(input.Items)
}

// convertResponsesItemsToChatMessages converts Responses input items into OpenAI Chat-style messages.
// It keeps full function_call -> function_call_output pairing by grouping consecutive function_call items
// into one assistant message with tool_calls, then appending each tool output message.
func convertResponsesItemsToChatMessages(items []ResponsesItem) ([]model.Message, error) {
	messages := make([]model.Message, 0, len(items))
	pendingToolCalls := make([]model.ToolCall, 0)

	flushPendingToolCalls := func() {
		if len(pendingToolCalls) == 0 {
			return
		}

		grouped := make([]model.ToolCall, len(pendingToolCalls))
		copy(grouped, pendingToolCalls)
		messages = append(messages, model.Message{
			Role:      "assistant",
			ToolCalls: grouped,
		})
		pendingToolCalls = pendingToolCalls[:0]
	}

	for _, item := range items {
		switch item.Type {
		case "function_call":
			pendingToolCalls = append(pendingToolCalls, convertFunctionCallItemToToolCall(item))
		case "function_call_output":
			flushPendingToolCalls()
			msg := convertFunctionCallOutputItemToToolMessage(item)
			if msg != nil {
				messages = append(messages, *msg)
			}
		default:
			flushPendingToolCalls()
			msg, err := convertItemToMessage(&item)
			if err != nil {
				return nil, err
			}
			if msg != nil {
				messages = append(messages, *msg)
			}
		}
	}

	flushPendingToolCalls()

	return messages, nil
}

func convertFunctionCallItemToToolCall(item ResponsesItem) model.ToolCall {
	callID := item.CallID
	if callID == "" {
		callID = item.ID
	}

	return model.ToolCall{
		ID:   callID,
		Type: "function",
		Function: model.FunctionCall{
			Name:      item.Name,
			Arguments: item.Arguments,
		},
	}
}

func convertFunctionCallOutputItemToToolMessage(item ResponsesItem) *model.Message {
	callID := item.CallID
	if callID == "" {
		callID = item.ID
	}

	var toolCallID *string
	if callID != "" {
		toolCallID = lo.ToPtr(callID)
	}

	content := model.MessageContent{}
	if item.Output != nil {
		content = convertInputToMessageContent(*item.Output)
	}

	return &model.Message{
		Role:       "tool",
		ToolCallID: toolCallID,
		Content:    content,
	}
}
