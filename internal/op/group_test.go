package op

import (
	"testing"

	"github.com/lingyuins/octopus/internal/model"
)

func TestNormalizeGroupItemsDedupesAndReorders(t *testing.T) {
	items := []model.GroupItem{
		{ChannelID: 7, ModelName: " model-a ", Priority: 9, Weight: 0},
		{ChannelID: 7, ModelName: "model-a", Priority: 1, Weight: 8},
		{ChannelID: 0, ModelName: "skip-me", Priority: 2, Weight: 1},
		{ChannelID: 9, ModelName: "model-b", Priority: 3, Weight: 2},
		{ChannelID: 9, ModelName: "   ", Priority: 4, Weight: 2},
	}

	got := normalizeGroupItems(items)
	if len(got) != 2 {
		t.Fatalf("expected 2 items after normalization, got %d", len(got))
	}

	if got[0].ChannelID != 7 || got[0].ModelName != "model-a" {
		t.Fatalf("unexpected first item: %+v", got[0])
	}
	if got[0].Priority != 1 || got[0].Weight != 1 {
		t.Fatalf("unexpected first item priority/weight: %+v", got[0])
	}

	if got[1].ChannelID != 9 || got[1].ModelName != "model-b" {
		t.Fatalf("unexpected second item: %+v", got[1])
	}
	if got[1].Priority != 2 || got[1].Weight != 2 {
		t.Fatalf("unexpected second item priority/weight: %+v", got[1])
	}
}
