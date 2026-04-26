package relay

import (
	"testing"

	appmodel "github.com/lingyuins/octopus/internal/model"
	transmodel "github.com/lingyuins/octopus/internal/transformer/model"
	"github.com/lingyuins/octopus/internal/transformer/outbound"
)

func TestPrepareInternalRequestForOutbound_IsScopedPerChannelAttempt(t *testing.T) {
	first := "first"
	second := "second"
	baseRequest := &transmodel.InternalLLMRequest{
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

	rewriteChannel := &appmodel.Channel{
		Type: outbound.OutboundTypeOpenAIChat,
		RequestRewrite: &appmodel.RequestRewriteConfig{
			Enabled: true,
			Profile: appmodel.RequestRewriteProfileOpenAIChatCompat,
		},
	}
	plainChannel := &appmodel.Channel{
		Type: outbound.OutboundTypeOpenAIChat,
	}

	rewritten, err := prepareInternalRequestForOutbound(rewriteChannel, baseRequest, appmodel.EndpointTypeDeepSeek)
	if err != nil {
		t.Fatalf("prepareInternalRequestForOutbound() rewrite channel error = %v", err)
	}
	plain, err := prepareInternalRequestForOutbound(plainChannel, baseRequest, appmodel.EndpointTypeChat)
	if err != nil {
		t.Fatalf("prepareInternalRequestForOutbound() plain channel error = %v", err)
	}

	if rewritten.Messages[0].Content.Content == nil || *rewritten.Messages[0].Content.Content != "first\nsecond" {
		t.Fatalf("rewritten content = %#v, want flattened string", rewritten.Messages[0].Content)
	}
	if plain.Messages[0].Content.Content != nil {
		t.Fatalf("plain channel content = %#v, want original multipart content", plain.Messages[0].Content)
	}
	if len(plain.Messages[0].Content.MultipleContent) != 2 {
		t.Fatalf("plain channel content parts len = %d, want 2", len(plain.Messages[0].Content.MultipleContent))
	}
	if baseRequest.Messages[0].Content.Content != nil {
		t.Fatal("base request was mutated across channel attempts")
	}
	if rewritten.TransformerMetadata[transmodel.TransformerMetadataGroupEndpointType] != appmodel.EndpointTypeDeepSeek {
		t.Fatalf("rewritten transformer metadata = %#v, want deepseek endpoint type", rewritten.TransformerMetadata)
	}
	if plain.TransformerMetadata[transmodel.TransformerMetadataGroupEndpointType] != appmodel.EndpointTypeChat {
		t.Fatalf("plain transformer metadata = %#v, want chat endpoint type", plain.TransformerMetadata)
	}
}
