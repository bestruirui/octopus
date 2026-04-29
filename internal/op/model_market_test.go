package op

import (
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/model"
)

func TestBuildModelMarket_AggregatesChannelsKeysAndStats(t *testing.T) {
	items, summary := buildModelMarket(
		[]model.LLMInfo{
			{
				Name: "gpt-5.2",
				LLMPrice: model.LLMPrice{
					Input:      1,
					Output:     2,
					CacheRead:  0.1,
					CacheWrite: 0.2,
				},
			},
		},
		[]model.LLMChannel{
			{Name: "gpt-5.2", ChannelID: 1, ChannelName: "NMapi", Enabled: true},
			{Name: "gpt-5.2", ChannelID: 2, ChannelName: "Ygxz", Enabled: false},
		},
		map[int]model.Channel{
			1: {ID: 1, Enabled: true, Keys: []model.ChannelKey{{Enabled: true}, {Enabled: true}, {Enabled: false}}},
			2: {ID: 2, Enabled: false, Keys: []model.ChannelKey{{Enabled: true}}},
		},
		[]model.StatsModel{
			{ID: 1, Name: "gpt-5.2", ChannelID: 1, StatsMetrics: model.StatsMetrics{WaitTime: 3000, RequestSuccess: 9, RequestFailed: 1}},
			{ID: 2, Name: "gpt-5.2", ChannelID: 2, StatsMetrics: model.StatsMetrics{WaitTime: 1000, RequestSuccess: 1, RequestFailed: 1}},
		},
		time.Date(2026, 4, 29, 10, 0, 0, 0, time.FixedZone("CST", 8*3600)),
	)

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ChannelCount != 2 {
		t.Fatalf("ChannelCount = %d, want 2", items[0].ChannelCount)
	}
	if items[0].EnabledKeyCount != 3 {
		t.Fatalf("EnabledKeyCount = %d, want 3", items[0].EnabledKeyCount)
	}
	if items[0].AverageLatencyMS != 333 {
		t.Fatalf("AverageLatencyMS = %d, want 333", items[0].AverageLatencyMS)
	}
	if items[0].SuccessRate != 0.8333333333333334 {
		t.Fatalf("SuccessRate = %v, want 0.8333333333333334", items[0].SuccessRate)
	}
	if summary.UniqueChannelCount != 2 {
		t.Fatalf("UniqueChannelCount = %d, want 2", summary.UniqueChannelCount)
	}
}

func TestBuildModelMarket_SortsItemsByName(t *testing.T) {
	items, _ := buildModelMarket(
		[]model.LLMInfo{
			{Name: "z-model"},
			{Name: "a-model"},
		},
		nil,
		nil,
		nil,
		time.Time{},
	)

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Name != "a-model" || items[1].Name != "z-model" {
		t.Fatalf("unexpected item order: %+v", items)
	}
}

func TestBuildModelMarket_UsesEmptyChannelsSliceWhenModelHasNoChannels(t *testing.T) {
	items, _ := buildModelMarket(
		[]model.LLMInfo{
			{Name: "standalone-model"},
		},
		nil,
		nil,
		nil,
		time.Time{},
	)

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Channels == nil {
		t.Fatal("Channels = nil, want empty slice")
	}
	if len(items[0].Channels) != 0 {
		t.Fatalf("len(Channels) = %d, want 0", len(items[0].Channels))
	}
}
