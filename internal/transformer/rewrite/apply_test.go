package rewrite

import (
	"testing"

	appmodel "github.com/lingyuins/octopus/internal/model"
	transmodel "github.com/lingyuins/octopus/internal/transformer/model"
	"github.com/lingyuins/octopus/internal/transformer/outbound"
)

func TestResolve_DisabledConfigReturnsNotEnabled(t *testing.T) {
	got, enabled, err := Resolve(outbound.OutboundTypeOpenAIChat, nil)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if enabled {
		t.Fatal("Resolve() enabled = true, want false")
	}
	if got != nil {
		t.Fatalf("Resolve() config = %#v, want nil", got)
	}
}

func TestResolve_OpenAIChatCompatDefaults(t *testing.T) {
	got, enabled, err := Resolve(outbound.OutboundTypeOpenAIChat, &appmodel.RequestRewriteConfig{
		Enabled: true,
		Profile: appmodel.RequestRewriteProfileOpenAIChatCompat,
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !enabled {
		t.Fatal("Resolve() enabled = false, want true")
	}
	if !got.FlattenTextBlockArrays || !got.NilContentAsEmptyString || !got.EnsureAssistantContentWithToolCalls {
		t.Fatalf("Resolve() defaults not applied: %#v", got)
	}
	if got.ToolRoleStrategy != appmodel.ToolRoleStrategyKeep {
		t.Fatalf("Resolve() ToolRoleStrategy = %q, want %q", got.ToolRoleStrategy, appmodel.ToolRoleStrategyKeep)
	}
	if got.SystemMessageStrategy != appmodel.SystemMessageStrategyKeep {
		t.Fatalf("Resolve() SystemMessageStrategy = %q, want %q", got.SystemMessageStrategy, appmodel.SystemMessageStrategyKeep)
	}
}

func TestResolve_RejectsUnsupportedChannelType(t *testing.T) {
	_, _, err := Resolve(outbound.OutboundTypeOpenAIResponse, &appmodel.RequestRewriteConfig{
		Enabled: true,
		Profile: appmodel.RequestRewriteProfileOpenAIChatCompat,
	})
	if err == nil {
		t.Fatal("Resolve() error = nil, want non-nil")
	}
}

func TestResolve_AcceptsMimoChannelType(t *testing.T) {
	got, enabled, err := Resolve(outbound.OutboundTypeMimo, &appmodel.RequestRewriteConfig{
		Enabled: true,
		Profile: appmodel.RequestRewriteProfileOpenAIChatCompat,
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !enabled {
		t.Fatal("Resolve() enabled = false, want true")
	}
	if got == nil {
		t.Fatal("Resolve() config = nil, want non-nil")
	}
}

func TestApply_FlattensTextOnlyBlockArray(t *testing.T) {
	first := "first"
	second := "second"
	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []transmodel.Message{
			{
				Role: "user",
				Content: transmodel.MessageContent{
					MultipleContent: []transmodel.MessageContentPart{
						{Type: "text", Text: &first},
						{Type: "text", Text: &second},
					},
				},
			},
		},
	}

	got, err := Apply(req, defaultEffectiveConfig())
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if got.Messages[0].Content.Content == nil || *got.Messages[0].Content.Content != "first\nsecond" {
		t.Fatalf("Apply() content = %#v, want flattened string", got.Messages[0].Content)
	}
	if req.Messages[0].Content.Content != nil {
		t.Fatal("Apply() mutated original request content")
	}
}

func TestApply_PreservesMixedMultipartContent(t *testing.T) {
	text := "hello"
	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []transmodel.Message{
			{
				Role: "user",
				Content: transmodel.MessageContent{
					MultipleContent: []transmodel.MessageContentPart{
						{Type: "text", Text: &text},
						{Type: "image_url", ImageURL: &transmodel.ImageURL{URL: "https://example.com/image.png"}},
					},
				},
			},
		},
	}

	got, err := Apply(req, defaultEffectiveConfig())
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got.Messages[0].Content.Content != nil {
		t.Fatalf("Apply() unexpectedly flattened mixed content: %#v", got.Messages[0].Content)
	}
	if len(got.Messages[0].Content.MultipleContent) != 2 {
		t.Fatalf("Apply() content parts len = %d, want 2", len(got.Messages[0].Content.MultipleContent))
	}
}

func TestApply_EnsuresAssistantContentWhenToolCallsPresent(t *testing.T) {
	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []transmodel.Message{
			{
				Role: "assistant",
				ToolCalls: []transmodel.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: transmodel.FunctionCall{
							Name:      "lookup_weather",
							Arguments: `{"city":"Shanghai"}`,
						},
					},
				},
			},
		},
	}

	got, err := Apply(req, defaultEffectiveConfig())
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got.Messages[0].Content.Content == nil || *got.Messages[0].Content.Content != "" {
		t.Fatalf("Apply() content = %#v, want empty string", got.Messages[0].Content)
	}
}

func TestApply_EnsuresNilContentAsEmptyString(t *testing.T) {
	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []transmodel.Message{
			{Role: "user"},
		},
	}

	got, err := Apply(req, defaultEffectiveConfig())
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got.Messages[0].Content.Content == nil || *got.Messages[0].Content.Content != "" {
		t.Fatalf("Apply() content = %#v, want empty string", got.Messages[0].Content)
	}
}

func TestApply_MergeSystemMessagesPreservesOrder(t *testing.T) {
	systemA := "system a"
	systemB := "system b"
	userText := "user text"
	cfg := defaultEffectiveConfig()
	cfg.SystemMessageStrategy = appmodel.SystemMessageStrategyMerge

	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []transmodel.Message{
			{Role: "system", Content: transmodel.MessageContent{Content: &systemA}},
			{Role: "system", Content: transmodel.MessageContent{Content: &systemB}},
			{Role: "user", Content: transmodel.MessageContent{Content: &userText}},
		},
	}

	got, err := Apply(req, cfg)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("Apply() messages len = %d, want 2", len(got.Messages))
	}
	if got.Messages[0].Role != "system" || got.Messages[0].Content.Content == nil || *got.Messages[0].Content.Content != "system a\n\nsystem b" {
		t.Fatalf("Apply() merged system = %#v", got.Messages[0])
	}
	if got.Messages[1].Role != "user" {
		t.Fatalf("Apply() second role = %q, want user", got.Messages[1].Role)
	}
}

func TestApply_StringifyToolRoleToUser(t *testing.T) {
	toolID := "call_1"
	text := "tool output"
	cfg := defaultEffectiveConfig()
	cfg.ToolRoleStrategy = appmodel.ToolRoleStrategyStringifyToUser

	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []transmodel.Message{
			{
				Role:       "tool",
				ToolCallID: &toolID,
				Content: transmodel.MessageContent{
					Content: &text,
				},
			},
		},
	}

	got, err := Apply(req, cfg)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got.Messages[0].Role != "user" {
		t.Fatalf("Apply() role = %q, want user", got.Messages[0].Role)
	}
	if got.Messages[0].ToolCallID != nil {
		t.Fatal("Apply() should clear tool_call_id when stringifying tool role")
	}
	if got.Messages[0].Content.Content == nil || *got.Messages[0].Content.Content != "Tool result (call_1):\ntool output" {
		t.Fatalf("Apply() content = %#v", got.Messages[0].Content)
	}
}

func TestApply_DoesNotMutateOriginalRequest(t *testing.T) {
	first := "first"
	second := "second"
	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []transmodel.Message{
			{
				Role: "user",
				Content: transmodel.MessageContent{
					MultipleContent: []transmodel.MessageContentPart{
						{Type: "text", Text: &first},
						{Type: "text", Text: &second},
					},
				},
			},
		},
	}

	_, err := Apply(req, defaultEffectiveConfig())
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if req.Messages[0].Content.Content != nil {
		t.Fatal("Apply() mutated original request content pointer")
	}
	if len(req.Messages[0].Content.MultipleContent) != 2 {
		t.Fatalf("Apply() mutated original request parts len = %d, want 2", len(req.Messages[0].Content.MultipleContent))
	}
}

func TestApply_IsIdempotent(t *testing.T) {
	first := "first"
	second := "second"
	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []transmodel.Message{
			{
				Role: "assistant",
				Content: transmodel.MessageContent{
					MultipleContent: []transmodel.MessageContentPart{
						{Type: "text", Text: &first},
						{Type: "text", Text: &second},
					},
				},
				ToolCalls: []transmodel.ToolCall{
					{ID: "call_1", Type: "function"},
				},
			},
		},
	}

	firstPass, err := Apply(req, defaultEffectiveConfig())
	if err != nil {
		t.Fatalf("Apply() first pass error = %v", err)
	}
	secondPass, err := Apply(firstPass, defaultEffectiveConfig())
	if err != nil {
		t.Fatalf("Apply() second pass error = %v", err)
	}

	got := ""
	if secondPass.Messages[0].Content.Content != nil {
		got = *secondPass.Messages[0].Content.Content
	}
	if got != "first\nsecond" {
		t.Fatalf("Apply() second pass content = %q, want %q", got, "first\nsecond")
	}
}

func defaultEffectiveConfig() *EffectiveConfig {
	return &EffectiveConfig{
		Profile:                             appmodel.RequestRewriteProfileOpenAIChatCompat,
		FlattenTextBlockArrays:              true,
		NilContentAsEmptyString:             true,
		EnsureAssistantContentWithToolCalls: true,
		ToolRoleStrategy:                    appmodel.ToolRoleStrategyKeep,
		SystemMessageStrategy:               appmodel.SystemMessageStrategyKeep,
	}
}
