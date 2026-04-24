package openai

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/lingyuins/octopus/internal/transformer/model"
)

func TestChatOutboundTransformRequest_NormalizesOpenAICompatMessages(t *testing.T) {
	outbound := &ChatOutbound{}
	toolCallID := "call_1"
	first := "first block"
	second := "second block"
	toolFirst := "tool output 1"
	toolSecond := "tool output 2"

	request := &model.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					MultipleContent: []model.MessageContentPart{
						{Type: "text", Text: &first},
						{Type: "text", Text: &second},
					},
				},
			},
			{
				Role: "assistant",
				ToolCalls: []model.ToolCall{
					{
						ID:   toolCallID,
						Type: "function",
						Function: model.FunctionCall{
							Name:      "lookup_weather",
							Arguments: `{"city":"Shanghai"}`,
						},
					},
				},
			},
			{
				Role:       "tool",
				ToolCallID: &toolCallID,
				Content: model.MessageContent{
					MultipleContent: []model.MessageContentPart{
						{Type: "text", Text: &toolFirst},
						{Type: "text", Text: &toolSecond},
					},
				},
			},
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://nullapi.example.com", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		Messages []struct {
			Role      string           `json:"role"`
			Content   json.RawMessage  `json:"content"`
			ToolCalls []model.ToolCall `json:"tool_calls,omitempty"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if len(got.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(got.Messages))
	}

	assertJSONEncodedString(t, got.Messages[0].Content, "first block\nsecond block")
	assertJSONEncodedString(t, got.Messages[1].Content, "")
	assertJSONEncodedString(t, got.Messages[2].Content, "tool output 1\ntool output 2")

	if len(got.Messages[1].ToolCalls) != 1 {
		t.Fatalf("expected assistant tool_calls to be preserved, got %d", len(got.Messages[1].ToolCalls))
	}

	if request.Messages[0].Content.Content != nil {
		t.Fatalf("expected original request user content to remain unflattened")
	}
	if len(request.Messages[0].Content.MultipleContent) != 2 {
		t.Fatalf("expected original request user content parts to stay intact")
	}
	if request.Messages[1].Content.Content != nil {
		t.Fatalf("expected original request assistant content to remain nil")
	}
}

func TestChatOutboundTransformRequest_PreservesMixedMultiPartContent(t *testing.T) {
	outbound := &ChatOutbound{}
	text := "look at this image"
	imageURL := "https://example.com/image.png"

	request := &model.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					MultipleContent: []model.MessageContentPart{
						{Type: "text", Text: &text},
						{Type: "image_url", ImageURL: &model.ImageURL{URL: imageURL}},
					},
				},
			},
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://nullapi.example.com", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		Messages []struct {
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}

	var contentParts []map[string]any
	if err := json.Unmarshal(got.Messages[0].Content, &contentParts); err != nil {
		t.Fatalf("expected mixed multipart content to stay as array, got error: %v", err)
	}
	if len(contentParts) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(contentParts))
	}
}

func assertJSONEncodedString(t *testing.T, raw json.RawMessage, want string) {
	t.Helper()

	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("expected JSON string %q, got %s (err=%v)", want, string(raw), err)
	}
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
