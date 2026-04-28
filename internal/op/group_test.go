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

func TestGroupGetEnabledMapByEndpoint_PrefersDeepSeekConversationGroupForDeepSeekModels(t *testing.T) {
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
				Name:         "deepseek-chat",
				EndpointType: model.EndpointTypeChat,
				Items: []model.GroupItem{
					{ChannelID: 1, ModelName: "deepseek-chat"},
				},
			},
			11: {
				ID:           11,
				Name:         "deepseek-chat",
				EndpointType: model.EndpointTypeDeepSeek,
				Items: []model.GroupItem{
					{ChannelID: 2, ModelName: "deepseek-chat"},
				},
			},
		},
	)

	got, err := GroupGetEnabledMapByEndpoint(model.EndpointTypeChat, "deepseek-chat", context.Background())
	if err != nil {
		t.Fatalf("expected deepseek group match: %v", err)
	}
	if got.ID != 11 {
		t.Fatalf("expected deepseek endpoint group id 11, got %d", got.ID)
	}
	if got.EndpointType != model.EndpointTypeDeepSeek {
		t.Fatalf("expected deepseek endpoint group, got %q", got.EndpointType)
	}
	if len(got.Items) != 1 || got.Items[0].ChannelID != 2 {
		t.Fatalf("expected deepseek group items, got %+v", got.Items)
	}
}

func TestGroupGetEnabledMapByEndpoint_UsesTrimmedNameAndPriorityOrderedItems(t *testing.T) {
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
				Name:         " gpt-4.1 ",
				EndpointType: model.EndpointTypeChat,
				Items: []model.GroupItem{
					{ID: 2, ChannelID: 2, ModelName: " model-b ", Priority: 2, Weight: 0},
					{ID: 1, ChannelID: 1, ModelName: " model-a ", Priority: 1, Weight: 3},
				},
			},
		},
	)

	got, err := GroupGetEnabledMapByEndpoint(model.EndpointTypeChat, "gpt-4.1", context.Background())
	if err != nil {
		t.Fatalf("expected trimmed group name to match: %v", err)
	}

	if got.Name != "gpt-4.1" {
		t.Fatalf("expected trimmed group name, got %q", got.Name)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got.Items))
	}
	if got.Items[0].ChannelID != 1 || got.Items[0].ModelName != "model-a" || got.Items[0].Priority != 1 {
		t.Fatalf("unexpected first item: %+v", got.Items[0])
	}
	if got.Items[1].ChannelID != 2 || got.Items[1].ModelName != "model-b" || got.Items[1].Priority != 2 {
		t.Fatalf("unexpected second item: %+v", got.Items[1])
	}
	if got.Items[1].Weight != 1 {
		t.Fatalf("expected invalid weight to default to 1, got %+v", got.Items[1])
	}
}

func TestGroupGetEnabledMapByEndpoint_RegexMatchesAreStableByGroupID(t *testing.T) {
	restore := snapshotGroupLookupState()
	defer restore()

	seedGroupLookupState(
		map[int]model.Channel{
			1: {ID: 1, Enabled: true},
			2: {ID: 2, Enabled: true},
		},
		map[int]model.Group{
			20: {
				ID:           20,
				Name:         "fallback",
				EndpointType: model.EndpointTypeAll,
				MatchRegex:   "(?i)^gpt-.*$",
				Items: []model.GroupItem{
					{ChannelID: 2, ModelName: "gpt-4.1"},
				},
			},
			10: {
				ID:           10,
				Name:         "preferred",
				EndpointType: model.EndpointTypeAll,
				MatchRegex:   "(?i)^gpt-4.*$",
				Items: []model.GroupItem{
					{ChannelID: 1, ModelName: "gpt-4.1"},
				},
			},
		},
	)

	got, err := GroupGetEnabledMapByEndpoint(model.EndpointTypeChat, "gpt-4.1", context.Background())
	if err != nil {
		t.Fatalf("expected regex group match: %v", err)
	}

	if got.ID != 10 {
		t.Fatalf("expected lowest id regex group to win deterministically, got %d", got.ID)
	}
	if len(got.Items) != 1 || got.Items[0].ChannelID != 1 {
		t.Fatalf("unexpected matched group items: %+v", got.Items)
	}
}

func TestRebuildGroupIndexesFromCache_SetsRegexMatchTimeout(t *testing.T) {
	restore := snapshotGroupLookupState()
	defer restore()

	seedGroupLookupState(
		map[int]model.Channel{
			1: {ID: 1, Enabled: true},
		},
		map[int]model.Group{
			10: {
				ID:           10,
				Name:         "preferred",
				EndpointType: model.EndpointTypeAll,
				MatchRegex:   "(?i)^gpt-4.*$",
				Items: []model.GroupItem{
					{ChannelID: 1, ModelName: "gpt-4.1"},
				},
			},
		},
	)

	groupRegexMatchersLock.RLock()
	matchers := append([]compiledGroupMatcher(nil), groupRegexMatchersByEndpoint[model.EndpointTypeAll]...)
	groupRegexMatchersLock.RUnlock()

	if len(matchers) != 1 {
		t.Fatalf("regex matcher count = %d, want 1", len(matchers))
	}
	if matchers[0].re.MatchTimeout != groupRegexMatchTimeout {
		t.Fatalf("regex match timeout = %s, want %s", matchers[0].re.MatchTimeout, groupRegexMatchTimeout)
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
