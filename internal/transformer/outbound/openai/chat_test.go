package openai

import (
	"context"
	"encoding/json"
	"io"
	"strings"
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

func TestChatOutboundTransformRequest_PreservesReasoningContentForDeepSeekToolContinuation(t *testing.T) {
	outbound := &ChatOutbound{}
	reasoning := "need one more tool round"
	content := ""
	toolCallID := "call_1"

	request := &model.InternalLLMRequest{
		Model: "deepseek-v4-pro",
		Messages: []model.Message{
			{
				Role: "assistant",
				Content: model.MessageContent{
					Content: &content,
				},
				ReasoningContent: &reasoning,
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
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://api.deepseek.com/v1", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		Messages []struct {
			ReasoningContent *string `json:"reasoning_content,omitempty"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	if got.Messages[0].ReasoningContent == nil || *got.Messages[0].ReasoningContent != reasoning {
		t.Fatalf("expected reasoning_content %q, got %#v", reasoning, got.Messages[0].ReasoningContent)
	}
	if request.Messages[0].ReasoningContent == nil || *request.Messages[0].ReasoningContent != reasoning {
		t.Fatalf("expected original request reasoning_content to stay intact")
	}
}

func TestChatOutboundTransformRequest_ClearsReasoningContentForDeepSeekFollowUpTurn(t *testing.T) {
	outbound := &ChatOutbound{}
	reasoning := "finished reasoning from the prior turn"
	content := "final answer"

	request := &model.InternalLLMRequest{
		Model: "deepseek-v4-pro",
		Messages: []model.Message{
			{
				Role: "assistant",
				Content: model.MessageContent{
					Content: &content,
				},
				ReasoningContent: &reasoning,
			},
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://api.deepseek.com/v1", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		Messages []struct {
			ReasoningContent *string `json:"reasoning_content,omitempty"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	if got.Messages[0].ReasoningContent != nil {
		t.Fatalf("expected reasoning_content to be cleared for DeepSeek follow-up turns, got %#v", got.Messages[0].ReasoningContent)
	}
	if request.Messages[0].ReasoningContent == nil || *request.Messages[0].ReasoningContent != reasoning {
		t.Fatalf("expected original request reasoning_content to stay intact")
	}
}

func TestChatOutboundTransformRequest_PreservesReasoningContentForDeepSeekEndpointCategory(t *testing.T) {
	outbound := &ChatOutbound{}
	reasoning := "preserve via deepseek endpoint category"
	content := "final answer"
	toolCallID := "call_123"
	toolCallName := "search"
	toolCallArgs := "{}"

	request := &model.InternalLLMRequest{
		Model: "custom-chat-model",
		Messages: []model.Message{
			{
				Role: "assistant",
				Content: model.MessageContent{
					Content: &content,
				},
				ReasoningContent: &reasoning,
				ToolCalls: []model.ToolCall{
					{
						ID:   toolCallID,
						Type: "function",
						Function: model.FunctionCall{
							Name:      toolCallName,
							Arguments: toolCallArgs,
						},
					},
				},
			},
		},
		TransformerMetadata: map[string]string{
			model.TransformerMetadataGroupEndpointType: "deepseek",
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://proxy.example.com/v1", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		Messages []struct {
			ReasoningContent *string `json:"reasoning_content,omitempty"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	if got.Messages[0].ReasoningContent == nil || *got.Messages[0].ReasoningContent != reasoning {
		t.Fatalf("expected reasoning_content %q to be preserved for deepseek endpoint category, got %#v", reasoning, got.Messages[0].ReasoningContent)
	}
}

func TestChatOutboundTransformRequest_ClearsReasoningContentForDeepSeekReasonerFollowUpTurn(t *testing.T) {
	outbound := &ChatOutbound{}
	reasoning := "reasoner chain of thought"
	content := "final answer"

	request := &model.InternalLLMRequest{
		Model: "deepseek-reasoner",
		Messages: []model.Message{
			{
				Role: "assistant",
				Content: model.MessageContent{
					Content: &content,
				},
				ReasoningContent: &reasoning,
			},
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://api.deepseek.com/v1", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		Messages []struct {
			ReasoningContent *string `json:"reasoning_content,omitempty"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	if got.Messages[0].ReasoningContent != nil {
		t.Fatalf("expected reasoning_content to be cleared for deepseek-reasoner follow-up turns, got %q", *got.Messages[0].ReasoningContent)
	}
}

func TestChatOutboundTransformRequest_ClearsReasoningContentForGenericOpenAICompat(t *testing.T) {
	outbound := &ChatOutbound{}
	reasoning := "should not be sent to generic openai compat"
	content := ""

	request := &model.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []model.Message{
			{
				Role: "assistant",
				Content: model.MessageContent{
					Content: &content,
				},
				ReasoningContent: &reasoning,
			},
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://api.openai.com/v1", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		Messages []struct {
			ReasoningContent *string `json:"reasoning_content,omitempty"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	if got.Messages[0].ReasoningContent != nil {
		t.Fatalf("expected reasoning_content to be cleared for generic openai compat, got %q", *got.Messages[0].ReasoningContent)
	}
}

func TestChatOutboundTransformRequest_NormalizesReasoningAliasForDeepSeekToolContinuation(t *testing.T) {
	outbound := &ChatOutbound{}
	reasoning := "provider-specific reasoning alias"
	content := ""
	toolCallID := "call_2"

	request := &model.InternalLLMRequest{
		Model: "deepseek-chat",
		Messages: []model.Message{
			{
				Role: "assistant",
				Content: model.MessageContent{
					Content: &content,
				},
				Reasoning: &reasoning,
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
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://api.deepseek.com/v1", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		Messages []struct {
			ReasoningContent *string `json:"reasoning_content,omitempty"`
			Reasoning        *string `json:"reasoning,omitempty"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	if got.Messages[0].ReasoningContent == nil || *got.Messages[0].ReasoningContent != reasoning {
		t.Fatalf("expected reasoning_content %q, got %#v", reasoning, got.Messages[0].ReasoningContent)
	}
	if got.Messages[0].Reasoning != nil {
		t.Fatalf("expected reasoning alias to be normalized away for DeepSeek, got %q", *got.Messages[0].Reasoning)
	}
	if request.Messages[0].Reasoning == nil || *request.Messages[0].Reasoning != reasoning {
		t.Fatalf("expected original request reasoning to stay intact")
	}
}

func TestChatOutboundTransformRequest_ClearsReasoningAliasForGenericOpenAICompat(t *testing.T) {
	outbound := &ChatOutbound{}
	reasoning := "should not be sent as reasoning alias"
	content := ""

	request := &model.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []model.Message{
			{
				Role: "assistant",
				Content: model.MessageContent{
					Content: &content,
				},
				Reasoning: &reasoning,
			},
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://api.openai.com/v1", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		Messages []struct {
			Reasoning *string `json:"reasoning,omitempty"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	if got.Messages[0].Reasoning != nil {
		t.Fatalf("expected reasoning alias to be cleared for generic openai compat, got %q", *got.Messages[0].Reasoning)
	}
}

func TestChatOutboundTransformRequest_OmitsNoneReasoningEffort(t *testing.T) {
	outbound := &ChatOutbound{}
	stream := false

	request := &model.InternalLLMRequest{
		Model:             "mimo-v2.5-pro",
		ReasoningEffort:   "none",
		Store:             &stream,
		ParallelToolCalls: &stream,
		Messages: []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					Content: loPtr("Use the get_current_time tool."),
				},
			},
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://token-plan-cn.xiaomimimo.com/v1", "tp-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		ReasoningEffort *string `json:"reasoning_effort,omitempty"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if got.ReasoningEffort != nil {
		t.Fatalf("expected reasoning_effort to be omitted, got %q", *got.ReasoningEffort)
	}
	if request.ReasoningEffort != "none" {
		t.Fatalf("expected original request reasoning_effort to stay intact, got %q", request.ReasoningEffort)
	}
}

func TestChatOutboundTransformRequest_MapsDeepSeekThinkingControls(t *testing.T) {
	outbound := &ChatOutbound{}
	stream := false

	request := &model.InternalLLMRequest{
		Model:           "DeepSeek-V4-Pro",
		ReasoningEffort: "xhigh",
		Store:           &stream,
		ExtraBody:       json.RawMessage(`{"thinking":{"type":"enabled"},"foo":"bar"}`),
		Messages: []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					Content: loPtr("hello"),
				},
			},
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://api.deepseek.com/v1", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		ReasoningEffort string         `json:"reasoning_effort,omitempty"`
		ExtraBody       map[string]any `json:"extra_body,omitempty"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if got.ReasoningEffort != "max" {
		t.Fatalf("expected deepseek reasoning_effort max, got %q", got.ReasoningEffort)
	}

	thinking, ok := got.ExtraBody["thinking"].(map[string]any)
	if !ok {
		t.Fatalf("expected extra_body.thinking to be preserved, got %#v", got.ExtraBody)
	}
	if thinking["type"] != "enabled" {
		t.Fatalf("expected extra_body.thinking.type enabled, got %#v", thinking["type"])
	}
	if got.ExtraBody["foo"] != "bar" {
		t.Fatalf("expected extra_body custom fields to be preserved, got %#v", got.ExtraBody["foo"])
	}
}

func TestChatOutboundTransformRequest_DisablesDeepSeekThinkingWhenReasoningEffortNone(t *testing.T) {
	outbound := &ChatOutbound{}
	stream := false

	request := &model.InternalLLMRequest{
		Model:           "DeepSeek-V4-Pro",
		ReasoningEffort: "none",
		Store:           &stream,
		Messages: []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					Content: loPtr("hello"),
				},
			},
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://api.deepseek.com/v1", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	bodyText := string(body)
	if strings.Contains(bodyText, `"reasoning_effort":"none"`) {
		t.Fatalf("expected invalid reasoning_effort none to be omitted, got %s", bodyText)
	}

	var got struct {
		ReasoningEffort *string `json:"reasoning_effort,omitempty"`
		ExtraBody       struct {
			Thinking struct {
				Type string `json:"type"`
			} `json:"thinking"`
		} `json:"extra_body,omitempty"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if got.ReasoningEffort != nil {
		t.Fatalf("expected reasoning_effort to be omitted, got %q", *got.ReasoningEffort)
	}
	if got.ExtraBody.Thinking.Type != "disabled" {
		t.Fatalf("expected deepseek thinking to be disabled, got %q", got.ExtraBody.Thinking.Type)
	}
}

func TestChatOutboundTransformRequest_DeepSeekThinkingDisabledOverridesReasoningEffort(t *testing.T) {
	outbound := &ChatOutbound{}
	stream := false

	request := &model.InternalLLMRequest{
		Model:           "deepseek-v4-pro",
		ReasoningEffort: "high",
		Store:           &stream,
		ExtraBody:       json.RawMessage(`{"thinking":{"type":"disabled"}}`),
		Messages: []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					Content: loPtr("hello"),
				},
			},
		},
	}

	httpReq, err := outbound.TransformRequest(context.Background(), request, "https://api.deepseek.com/v1", "sk-test")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var got struct {
		ReasoningEffort *string `json:"reasoning_effort,omitempty"`
		ExtraBody       struct {
			Thinking struct {
				Type string `json:"type"`
			} `json:"thinking"`
		} `json:"extra_body,omitempty"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal outbound body: %v", err)
	}

	if got.ReasoningEffort != nil {
		t.Fatalf("expected reasoning_effort to be omitted when thinking disabled, got %q", *got.ReasoningEffort)
	}
	if got.ExtraBody.Thinking.Type != "disabled" {
		t.Fatalf("expected deepseek thinking type disabled, got %q", got.ExtraBody.Thinking.Type)
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

func loPtr[T any](v T) *T {
	return &v
}
