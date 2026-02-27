package anthropic

import (
	"context"
	"encoding/json"
	"testing"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func TestTransformRequest_AdaptiveThinking_WithEffort(t *testing.T) {
	req := MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 1024,
		Messages: []MessageParam{
			{Role: "user", Content: MessageContent{Content: strPtr("hello")}},
		},
		Thinking: &Thinking{
			Type: ThinkingTypeAdaptive,
		},
		OutputConfig: &OutputConfig{
			Effort: EffortMedium,
		},
		Stream: boolPtr(false),
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	inbound := &MessagesInbound{}
	result, err := inbound.TransformRequest(context.Background(), body)
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}

	if result.ReasoningEffort != EffortMedium {
		t.Errorf("expected reasoning effort %q, got %q", EffortMedium, result.ReasoningEffort)
	}
	if !result.AdaptiveThinking {
		t.Error("expected AdaptiveThinking to be true")
	}
}

func TestTransformRequest_AdaptiveThinking_DefaultHigh(t *testing.T) {
	req := MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 1024,
		Messages: []MessageParam{
			{Role: "user", Content: MessageContent{Content: strPtr("hello")}},
		},
		Thinking: &Thinking{
			Type: ThinkingTypeAdaptive,
		},
		Stream: boolPtr(false),
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	inbound := &MessagesInbound{}
	result, err := inbound.TransformRequest(context.Background(), body)
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}

	if result.ReasoningEffort != EffortHigh {
		t.Errorf("expected default reasoning effort %q, got %q", EffortHigh, result.ReasoningEffort)
	}
	if !result.AdaptiveThinking {
		t.Error("expected AdaptiveThinking to be true")
	}
}

func TestTransformRequest_Enabled_BudgetTokensNil(t *testing.T) {
	req := MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 1024,
		Messages: []MessageParam{
			{Role: "user", Content: MessageContent{Content: strPtr("hello")}},
		},
		Thinking: &Thinking{
			Type: ThinkingTypeEnabled,
			// BudgetTokens intentionally nil
		},
		Stream: boolPtr(false),
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	inbound := &MessagesInbound{}
	result, err := inbound.TransformRequest(context.Background(), body)
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}

	// When BudgetTokens is nil, reasoning should not be configured
	if result.ReasoningEffort != "" {
		t.Errorf("expected empty reasoning effort when BudgetTokens is nil, got %q", result.ReasoningEffort)
	}
	if result.ReasoningBudget != nil {
		t.Errorf("expected nil reasoning budget when BudgetTokens is nil, got %v", *result.ReasoningBudget)
	}
}

func TestTransformRequest_Enabled_WithBudgetTokens(t *testing.T) {
	req := MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 1024,
		Messages: []MessageParam{
			{Role: "user", Content: MessageContent{Content: strPtr("hello")}},
		},
		Thinking: &Thinking{
			Type:         ThinkingTypeEnabled,
			BudgetTokens: int64Ptr(32000),
		},
		Stream: boolPtr(false),
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	inbound := &MessagesInbound{}
	result, err := inbound.TransformRequest(context.Background(), body)
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}

	if result.ReasoningEffort != EffortHigh {
		t.Errorf("expected reasoning effort %q for budget 32000, got %q", EffortHigh, result.ReasoningEffort)
	}
	if result.ReasoningBudget == nil || *result.ReasoningBudget != 32000 {
		t.Errorf("expected reasoning budget 32000, got %v", result.ReasoningBudget)
	}
	if result.AdaptiveThinking {
		t.Error("expected AdaptiveThinking to be false for enabled type")
	}
}

func TestTransformRequest_Disabled(t *testing.T) {
	req := MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 1024,
		Messages: []MessageParam{
			{Role: "user", Content: MessageContent{Content: strPtr("hello")}},
		},
		Thinking: &Thinking{
			Type: ThinkingTypeDisabled,
		},
		Stream: boolPtr(false),
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	inbound := &MessagesInbound{}
	result, err := inbound.TransformRequest(context.Background(), body)
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}

	if result.ReasoningEffort != "" {
		t.Errorf("expected empty reasoning effort for disabled thinking, got %q", result.ReasoningEffort)
	}
	if result.AdaptiveThinking {
		t.Error("expected AdaptiveThinking to be false for disabled type")
	}
}

func strPtr(s string) *string {
	return &s
}
