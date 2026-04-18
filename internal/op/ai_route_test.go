package op

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/model"
)

func TestNormalizeAIRouteEntriesMergesSameRequestedModel(t *testing.T) {
	routes := []model.AIRouteEntry{
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: " gpt-4o ",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "gpt-4o", Priority: 1, Weight: 100},
				{ChannelID: 1, UpstreamModel: "gpt-4o", Priority: 2, Weight: 50},
			},
		},
		{
			EndpointType:   model.EndpointTypeAll,
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

func TestNormalizeAIRouteEntriesKeepsDifferentEndpointTypesSeparated(t *testing.T) {
	routes := []model.AIRouteEntry{
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: "common-model",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "gpt-4o"},
			},
		},
		{
			EndpointType:   model.EndpointTypeEmbeddings,
			RequestedModel: "common-model",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 2, UpstreamModel: "text-embedding-3-large"},
			},
		},
	}

	got := normalizeAIRouteEntries(routes)
	if len(got) != 2 {
		t.Fatalf("normalizeAIRouteEntries() len = %d, want 2", len(got))
	}

	if got[0].EndpointType == got[1].EndpointType {
		t.Fatalf("normalizeAIRouteEntries() endpoint types should stay separated, got %+v", got)
	}
}

func TestBuildAIRoutePromptBucketsSplitsCapabilities(t *testing.T) {
	inputs := []model.AIRouteModelInput{
		{ChannelID: 1, ChannelName: "chat", Provider: "openai", Model: "gpt-4o"},
		{ChannelID: 1, ChannelName: "chat", Provider: "openai", Model: "gpt-4o"},
		{ChannelID: 2, ChannelName: "embed", Provider: "openai", Model: "text-embedding-3-large"},
		{ChannelID: 3, ChannelName: "image", Provider: "openai", Model: "gpt-image-1"},
		{ChannelID: 4, ChannelName: "chat2", Provider: "anthropic", Model: "claude-sonnet-4.5"},
	}

	got := buildAIRoutePromptBuckets(inputs, "")
	if len(got) != 3 {
		t.Fatalf("buildAIRoutePromptBuckets() len = %d, want 3", len(got))
	}

	if got[0].PromptEndpointType != model.EndpointTypeChat || got[0].GroupEndpointType != model.EndpointTypeAll {
		t.Fatalf("buildAIRoutePromptBuckets()[0] = %+v, want chat bucket mapped to *", got[0])
	}
	if len(got[0].ModelInputs) != 2 {
		t.Fatalf("chat bucket len = %d, want 2", len(got[0].ModelInputs))
	}
	if got[1].PromptEndpointType != model.EndpointTypeEmbeddings || got[1].GroupEndpointType != model.EndpointTypeEmbeddings {
		t.Fatalf("buildAIRoutePromptBuckets()[1] = %+v, want embeddings bucket", got[1])
	}
	if len(got[1].ModelInputs) != 1 || got[1].ModelInputs[0].Model != "text-embedding-3-large" {
		t.Fatalf("embeddings bucket = %+v, want text-embedding-3-large", got[1].ModelInputs)
	}
	if got[2].PromptEndpointType != model.EndpointTypeImageGeneration || got[2].GroupEndpointType != model.EndpointTypeImageGeneration {
		t.Fatalf("buildAIRoutePromptBuckets()[2] = %+v, want image bucket", got[2])
	}
}

func TestSummarizeAIRouteErrorBodyCollapsesHTML(t *testing.T) {
	got := summarizeAIRouteErrorBody("<html><body>504 Gateway Time-out</body></html>")
	if got != "upstream returned an HTML error page" {
		t.Fatalf("summarizeAIRouteErrorBody() = %q, want %q", got, "upstream returned an HTML error page")
	}
}

func TestDetectAIRoutePromptEndpointTypeForGroupPrefersNonChatWhenStable(t *testing.T) {
	group := model.Group{
		EndpointType: model.EndpointTypeAll,
		Items: []model.GroupItem{
			{ModelName: "text-embedding-3-large"},
			{ModelName: "text-embedding-3-small"},
		},
	}

	got := detectAIRoutePromptEndpointTypeForGroup(group)
	if got != model.EndpointTypeEmbeddings {
		t.Fatalf("detectAIRoutePromptEndpointTypeForGroup() = %q, want %q", got, model.EndpointTypeEmbeddings)
	}
}

func TestBuildAIRoutePromptBucketsSplitsLargeBucket(t *testing.T) {
	inputs := make([]model.AIRouteModelInput, 0, aiRouteMaxModelsPerRequest+5)
	for i := 0; i < aiRouteMaxModelsPerRequest+5; i++ {
		inputs = append(inputs, model.AIRouteModelInput{
			ChannelID:   i + 1,
			ChannelName: "chat",
			Provider:    "openai",
			Model:       fmt.Sprintf("gpt-4o-variant-%03d", i),
		})
	}

	got := buildAIRoutePromptBuckets(inputs, model.EndpointTypeChat)
	if len(got) != 2 {
		t.Fatalf("buildAIRoutePromptBuckets() len = %d, want 2", len(got))
	}

	total := 0
	for _, bucket := range got {
		if len(bucket.ModelInputs) > aiRouteMaxModelsPerRequest {
			t.Fatalf("bucket len = %d, want <= %d", len(bucket.ModelInputs), aiRouteMaxModelsPerRequest)
		}
		total += len(bucket.ModelInputs)
	}

	if total != len(inputs) {
		t.Fatalf("buildAIRoutePromptBuckets() total = %d, want %d", total, len(inputs))
	}
}

func TestFormatAIRouteTimeout(t *testing.T) {
	got := formatAIRouteTimeout(3 * time.Minute)
	if got != "180s" {
		t.Fatalf("formatAIRouteTimeout() = %q, want %q", got, "180s")
	}
}

func TestIsAIRouteTimeoutErrorDetectsContextDeadlineExceeded(t *testing.T) {
	if !isAIRouteTimeoutError(context.DeadlineExceeded) {
		t.Fatal("isAIRouteTimeoutError() = false, want true")
	}
}
