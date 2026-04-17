package op

import (
	"testing"

	"github.com/lingyuins/octopus/internal/model"
)

func TestNormalizeAIRouteEntriesMergesSameRequestedModel(t *testing.T) {
	routes := []model.AIRouteEntry{
		{
			RequestedModel: " gpt-4o ",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "gpt-4o", Priority: 1, Weight: 100},
				{ChannelID: 1, UpstreamModel: "gpt-4o", Priority: 2, Weight: 50},
			},
		},
		{
			RequestedModel: "GPT-4O",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 2, UpstreamModel: "chatgpt-4o-latest", Priority: 1, Weight: 80},
				{ChannelID: 0, UpstreamModel: "ignored", Priority: 1, Weight: 100},
			},
		},
		{
			RequestedModel: "   ",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 3, UpstreamModel: "should-be-skipped", Priority: 1, Weight: 100},
			},
		},
	}

	got := normalizeAIRouteEntries(routes)
	if len(got) != 1 {
		t.Fatalf("normalizeAIRouteEntries() len = %d, want 1", len(got))
	}

	if got[0].RequestedModel != "gpt-4o" {
		t.Fatalf("normalizeAIRouteEntries()[0].RequestedModel = %q, want %q", got[0].RequestedModel, "gpt-4o")
	}

	if len(got[0].Items) != 2 {
		t.Fatalf("normalizeAIRouteEntries()[0].Items len = %d, want 2", len(got[0].Items))
	}

	if got[0].Items[0].ChannelID != 1 || got[0].Items[0].UpstreamModel != "gpt-4o" {
		t.Fatalf("normalizeAIRouteEntries()[0].Items[0] = %+v, want channel 1 / gpt-4o", got[0].Items[0])
	}

	if got[0].Items[1].ChannelID != 2 || got[0].Items[1].UpstreamModel != "chatgpt-4o-latest" {
		t.Fatalf("normalizeAIRouteEntries()[0].Items[1] = %+v, want channel 2 / chatgpt-4o-latest", got[0].Items[1])
	}
}

func TestDedupeAIRouteItemsPreservesFirstOccurrence(t *testing.T) {
	items := []model.AIRouteItemSpec{
		{ChannelID: 1, UpstreamModel: " model-a ", Priority: 5, Weight: 20},
		{ChannelID: 1, UpstreamModel: "MODEL-A", Priority: 1, Weight: 100},
		{ChannelID: 2, UpstreamModel: "model-a", Priority: 1, Weight: 100},
		{ChannelID: 3, UpstreamModel: "", Priority: 1, Weight: 100},
	}

	got := dedupeAIRouteItems(items)
	if len(got) != 2 {
		t.Fatalf("dedupeAIRouteItems() len = %d, want 2", len(got))
	}

	if got[0].ChannelID != 1 || got[0].UpstreamModel != "model-a" || got[0].Priority != 5 || got[0].Weight != 20 {
		t.Fatalf("dedupeAIRouteItems()[0] = %+v, want first trimmed occurrence", got[0])
	}

	if got[1].ChannelID != 2 || got[1].UpstreamModel != "model-a" {
		t.Fatalf("dedupeAIRouteItems()[1] = %+v, want channel 2 / model-a", got[1])
	}
}
