package authropic

import (
	"testing"

	anthropicModel "github.com/bestruirui/octopus/internal/transformer/inbound/anthropic"
	"github.com/bestruirui/octopus/internal/transformer/model"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func strPtr(s string) *string {
	return &s
}

func TestConvertToAnthropicRequest_AdaptiveThinking(t *testing.T) {
	req := &model.InternalLLMRequest{
		Model:            "claude-sonnet-4-20250514",
		MaxTokens:        int64Ptr(1024),
		Stream:           boolPtr(false),
		ReasoningEffort:  anthropicModel.EffortMedium,
		AdaptiveThinking: true,
		Messages: []model.Message{
			{Role: "user", Content: model.MessageContent{Content: strPtr("hello")}},
		},
	}

	result := convertToAnthropicRequest(req)

	if result.Thinking == nil {
		t.Fatal("expected Thinking to be set")
	}
	if result.Thinking.Type != anthropicModel.ThinkingTypeAdaptive {
		t.Errorf("expected thinking type %q, got %q", anthropicModel.ThinkingTypeAdaptive, result.Thinking.Type)
	}
	if result.Thinking.BudgetTokens != nil {
		t.Error("expected BudgetTokens to be nil for adaptive thinking")
	}
	if result.OutputConfig == nil {
		t.Fatal("expected OutputConfig to be set")
	}
	if result.OutputConfig.Effort != anthropicModel.EffortMedium {
		t.Errorf("expected effort %q, got %q", anthropicModel.EffortMedium, result.OutputConfig.Effort)
	}
}

func TestConvertToAnthropicRequest_EnabledThinking(t *testing.T) {
	req := &model.InternalLLMRequest{
		Model:            "claude-sonnet-4-20250514",
		MaxTokens:        int64Ptr(1024),
		Stream:           boolPtr(false),
		ReasoningEffort:  anthropicModel.EffortHigh,
		AdaptiveThinking: false,
		Messages: []model.Message{
			{Role: "user", Content: model.MessageContent{Content: strPtr("hello")}},
		},
	}

	result := convertToAnthropicRequest(req)

	if result.Thinking == nil {
		t.Fatal("expected Thinking to be set")
	}
	if result.Thinking.Type != anthropicModel.ThinkingTypeEnabled {
		t.Errorf("expected thinking type %q, got %q", anthropicModel.ThinkingTypeEnabled, result.Thinking.Type)
	}
	if result.Thinking.BudgetTokens == nil {
		t.Fatal("expected BudgetTokens to be set")
	}
	if *result.Thinking.BudgetTokens != 32768 {
		t.Errorf("expected budget tokens 32768 for high effort, got %d", *result.Thinking.BudgetTokens)
	}
	if result.OutputConfig != nil {
		t.Error("expected OutputConfig to be nil for enabled thinking")
	}
}

func TestConvertToAnthropicRequest_EnabledThinking_WithBudget(t *testing.T) {
	budget := int64(50000)
	req := &model.InternalLLMRequest{
		Model:           "claude-sonnet-4-20250514",
		MaxTokens:       int64Ptr(1024),
		Stream:          boolPtr(false),
		ReasoningEffort: anthropicModel.EffortHigh,
		ReasoningBudget: &budget,
		Messages: []model.Message{
			{Role: "user", Content: model.MessageContent{Content: strPtr("hello")}},
		},
	}

	result := convertToAnthropicRequest(req)

	if result.Thinking == nil {
		t.Fatal("expected Thinking to be set")
	}
	if result.Thinking.BudgetTokens == nil {
		t.Fatal("expected BudgetTokens to be set")
	}
	if *result.Thinking.BudgetTokens != 50000 {
		t.Errorf("expected budget tokens 50000 (from original budget), got %d", *result.Thinking.BudgetTokens)
	}
}

func TestConvertToAnthropicRequest_NoThinking(t *testing.T) {
	req := &model.InternalLLMRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: int64Ptr(1024),
		Stream:    boolPtr(false),
		Messages: []model.Message{
			{Role: "user", Content: model.MessageContent{Content: strPtr("hello")}},
		},
	}

	result := convertToAnthropicRequest(req)

	if result.Thinking != nil {
		t.Error("expected Thinking to be nil when ReasoningEffort is empty")
	}
	if result.OutputConfig != nil {
		t.Error("expected OutputConfig to be nil when ReasoningEffort is empty")
	}
}

func TestGetThinkingBudget(t *testing.T) {
	tests := []struct {
		name     string
		effort   string
		budget   *int64
		expected int64
	}{
		{
			name:     "low effort",
			effort:   anthropicModel.EffortLow,
			expected: 1024,
		},
		{
			name:     "medium effort",
			effort:   anthropicModel.EffortMedium,
			expected: 8192,
		},
		{
			name:     "high effort",
			effort:   anthropicModel.EffortHigh,
			expected: 32768,
		},
		{
			name:     "unknown effort defaults to medium",
			effort:   "unknown",
			expected: 8192,
		},
		{
			name:     "explicit budget overrides effort",
			effort:   anthropicModel.EffortLow,
			budget:   int64Ptr(99999),
			expected: 99999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getThinkingBudget(tt.effort, tt.budget)
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if *result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, *result)
			}
		})
	}
}
