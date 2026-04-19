package op

import (
	"context"
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

func TestGroupGetEnabledMapByEndpoint_FallsBackAcrossConversationEndpoints(t *testing.T) {
	restore := snapshotGroupLookupState()
	defer restore()

	seedGroupLookupState(
		map[int]model.Channel{
			1: {ID: 1, Enabled: true},
			2: {ID: 2, Enabled: false},
		},
		map[int]model.Group{
			10: {
				ID:           10,
				Name:         "gpt-4.1",
				EndpointType: model.EndpointTypeChat,
				Items: []model.GroupItem{
					{ChannelID: 1, ModelName: "gpt-4.1"},
					{ChannelID: 2, ModelName: "gpt-4.1"},
				},
			},
		},
	)

	got, err := GroupGetEnabledMapByEndpoint(model.EndpointTypeResponses, "gpt-4.1", context.Background())
	if err != nil {
		t.Fatalf("expected responses lookup to fall back to chat group: %v", err)
	}

	if got.ID != 10 {
		t.Fatalf("expected fallback chat group id 10, got %d", got.ID)
	}
	if got.EndpointType != model.EndpointTypeChat {
		t.Fatalf("expected chat endpoint group, got %q", got.EndpointType)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected disabled channel items to be filtered out, got %d items", len(got.Items))
	}
	if got.Items[0].ChannelID != 1 {
		t.Fatalf("expected enabled channel 1, got %+v", got.Items[0])
	}
}

func TestGroupGetEnabledMapByEndpoint_PrefersExactConversationEndpointMatch(t *testing.T) {
	restore := snapshotGroupLookupState()
	defer restore()

	seedGroupLookupState(
		map[int]model.Channel{
			1: {ID: 1, Enabled: true},
			2: {ID: 2, Enabled: true},
		},
		map[int]model.Group{
			10: {
				ID:           10,
				Name:         "gpt-4.1",
				EndpointType: model.EndpointTypeChat,
				Items: []model.GroupItem{
					{ChannelID: 1, ModelName: "gpt-4.1"},
				},
			},
			11: {
				ID:           11,
				Name:         "gpt-4.1",
				EndpointType: model.EndpointTypeResponses,
				Items: []model.GroupItem{
					{ChannelID: 2, ModelName: "gpt-4.1"},
				},
			},
		},
	)

	got, err := GroupGetEnabledMapByEndpoint(model.EndpointTypeResponses, "gpt-4.1", context.Background())
	if err != nil {
		t.Fatalf("expected exact responses group match: %v", err)
	}

	if got.ID != 11 {
		t.Fatalf("expected exact responses group id 11, got %d", got.ID)
	}
	if len(got.Items) != 1 || got.Items[0].ChannelID != 2 {
		t.Fatalf("expected exact responses group items, got %+v", got.Items)
	}
}

func snapshotGroupLookupState() func() {
	oldGroups := groupCache.GetAll()
	oldChannels := channelCache.GetAll()

	return func() {
		seedGroupLookupState(oldChannels, oldGroups)
	}
}

func seedGroupLookupState(channels map[int]model.Channel, groups map[int]model.Group) {
	channelCache.Clear()
	for id, channel := range channels {
		channelCache.Set(id, channel)
	}

	groupCache.Clear()
	for id, group := range groups {
		groupCache.Set(id, group)
	}

	rebuildGroupIndexesFromCache()
}
