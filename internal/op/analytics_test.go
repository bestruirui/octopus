package op

import (
	"testing"

	"github.com/lingyuins/octopus/internal/model"
)

func TestBuildAnalyticsOverview_NoData(t *testing.T) {
	got := buildAnalyticsOverview(model.StatsMetrics{}, 0, 0, 0, 0)
	if got.RequestCount != 0 || got.TotalTokens != 0 || got.TotalCost != 0 {
		t.Fatalf("unexpected non-zero overview: %+v", got)
	}
	if got.ProviderCount != 0 || got.APIKeyCount != 0 || got.ModelCount != 0 {
		t.Fatalf("unexpected counts: %+v", got)
	}
}

func TestBuildProviderBreakdown_SortsByRequestsDesc(t *testing.T) {
	rows := map[int]*analyticsProviderAggregateRow{
		2: {
			ChannelID:   2,
			ChannelName: "beta",
			analyticsAggregateMetrics: analyticsAggregateMetrics{
				InputTokens:    40,
				OutputTokens:   60,
				TotalCost:      2,
				RequestSuccess: 2,
				RequestFailed:  1,
			},
		},
		1: {
			ChannelID:   1,
			ChannelName: "alpha",
			analyticsAggregateMetrics: analyticsAggregateMetrics{
				InputTokens:    10,
				OutputTokens:   20,
				TotalCost:      5,
				RequestSuccess: 1,
				RequestFailed:  0,
			},
		},
	}

	got := buildProviderBreakdown(rows, map[int]model.Channel{
		1: {ID: 1, Name: "alpha", Enabled: true},
		2: {ID: 2, Name: "beta", Enabled: false},
	})

	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].ChannelID != 2 {
		t.Fatalf("expected channel 2 first by request count, got %+v", got[0])
	}
	if got[0].RequestCount != 3 || got[0].TotalTokens != 100 {
		t.Fatalf("unexpected aggregate for first item: %+v", got[0])
	}
	if got[0].Enabled {
		t.Fatalf("expected channel 2 to be disabled: %+v", got[0])
	}
	if got[1].ChannelID != 1 || got[1].ChannelName != "alpha" {
		t.Fatalf("expected channel 1 second, got %+v", got[1])
	}
}
