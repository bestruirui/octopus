package relay

import (
	"testing"

	appmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/transformer/inbound"
	transmodel "github.com/lingyuins/octopus/internal/transformer/model"
	"github.com/lingyuins/octopus/internal/utils/semantic_cache"
)

func TestBuildSemanticCacheText_ChatMessagesOnlyUsesText(t *testing.T) {
	userText := "hello"
	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4.1",
		Messages: []transmodel.Message{
			{
				Role: "user",
				Content: transmodel.MessageContent{
					Content: &userText,
				},
			},
		},
	}

	namespace, text, ok := buildSemanticCacheLookupInput(7, "chat", req)
	if !ok {
		t.Fatal("expected cacheable request")
	}
	if namespace != "7:chat:gpt-4.1" {
		t.Fatalf("namespace = %q", namespace)
	}
	if text != "user: hello" {
		t.Fatalf("text = %q", text)
	}
}

func TestBuildSemanticCacheLookupInput_BypassesStreamRequests(t *testing.T) {
	stream := true
	req := &transmodel.InternalLLMRequest{Model: "gpt-4.1", Stream: &stream}

	if _, _, ok := buildSemanticCacheLookupInput(1, "chat", req); ok {
		t.Fatal("expected stream request to bypass semantic cache")
	}
}

func TestSemanticCacheEndpointFamily_UsesHandlerInputs(t *testing.T) {
	if got := semanticCacheEndpointFamily(appmodel.EndpointTypeChat, inbound.InboundTypeOpenAIChat); got != "chat" {
		t.Fatalf("chat family = %q", got)
	}
	if got := semanticCacheEndpointFamily(appmodel.EndpointTypeResponses, inbound.InboundTypeOpenAIResponse); got != "responses" {
		t.Fatalf("responses family = %q", got)
	}
	if got := semanticCacheEndpointFamily(appmodel.EndpointTypeMessages, inbound.InboundTypeAnthropic); got != "" {
		t.Fatalf("anthropic family = %q, want empty", got)
	}
	if got := semanticCacheEndpointFamily(appmodel.EndpointTypeEmbeddings, inbound.InboundTypeOpenAIEmbedding); got != "" {
		t.Fatalf("embedding family = %q, want empty", got)
	}
}

func TestBuildSemanticCacheLookupInput_IgnoresNonTextPartsAndNormalizesWhitespace(t *testing.T) {
	text := "  hello\n\nworld\t "
	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4.1",
		Messages: []transmodel.Message{
			{
				Role: "user",
				Content: transmodel.MessageContent{
					MultipleContent: []transmodel.MessageContentPart{
						{Type: "text", Text: &text},
						{Type: "image_url", ImageURL: &transmodel.ImageURL{URL: "https://example.com/image.png"}},
						{Type: "input_audio", Audio: &transmodel.Audio{Format: "mp3", Data: "abc"}},
					},
				},
			},
		},
	}

	namespace, normalized, ok := buildSemanticCacheLookupInput(9, "responses", req)
	if !ok {
		t.Fatal("expected cacheable request")
	}
	if namespace != "9:responses:gpt-4.1" {
		t.Fatalf("namespace = %q", namespace)
	}
	if normalized != "user: hello world" {
		t.Fatalf("normalized = %q", normalized)
	}
}

func TestBuildSemanticCacheLookupInput_BypassesRequestsWithoutStableText(t *testing.T) {
	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4.1",
		Messages: []transmodel.Message{
			{
				Role: "user",
				Content: transmodel.MessageContent{
					MultipleContent: []transmodel.MessageContentPart{
						{Type: "image_url", ImageURL: &transmodel.ImageURL{URL: "https://example.com/image.png"}},
					},
				},
			},
		},
	}

	if _, _, ok := buildSemanticCacheLookupInput(1, "chat", req); ok {
		t.Fatal("expected non-text-only request to bypass semantic cache")
	}
}

func TestBuildSemanticCacheLookupInput_RecordsBypassStats(t *testing.T) {
	semantic_cache.ResetRuntimeStats()

	stream := true
	req := &transmodel.InternalLLMRequest{Model: "gpt-4.1", Stream: &stream}
	if _, _, ok := buildSemanticCacheLookupInput(1, "chat", req); ok {
		t.Fatal("expected stream request to bypass semantic cache")
	}

	stats := semantic_cache.GetRuntimeStats()
	if stats.BypassedRequests != 1 {
		t.Fatalf("bypassed_requests = %d, want 1", stats.BypassedRequests)
	}
}

func TestSemanticCacheRuntimeStatsCounters(t *testing.T) {
	semantic_cache.ResetRuntimeStats()

	semantic_cache.RecordEvaluated()
	semantic_cache.RecordHit()
	semantic_cache.RecordMiss()
	semantic_cache.RecordBypass()
	semantic_cache.RecordStored()

	stats := semantic_cache.GetRuntimeStats()
	if stats.EvaluatedRequests != 1 {
		t.Fatalf("evaluated_requests = %d, want 1", stats.EvaluatedRequests)
	}
	if stats.CacheHitResponses != 1 {
		t.Fatalf("cache_hit_responses = %d, want 1", stats.CacheHitResponses)
	}
	if stats.CacheMissRequests != 1 {
		t.Fatalf("cache_miss_requests = %d, want 1", stats.CacheMissRequests)
	}
	if stats.BypassedRequests != 1 {
		t.Fatalf("bypassed_requests = %d, want 1", stats.BypassedRequests)
	}
	if stats.StoredResponses != 1 {
		t.Fatalf("stored_responses = %d, want 1", stats.StoredResponses)
	}
}
