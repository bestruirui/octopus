package relay

import (
	"testing"

	transmodel "github.com/lingyuins/octopus/internal/transformer/model"
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
