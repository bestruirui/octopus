package op

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/db"
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

func TestBuildProviderBreakdown_PreservesHistoricalUsageFromStats(t *testing.T) {
	rows := map[int]*analyticsProviderAggregateRow{
		1: {
			ChannelID:   1,
			ChannelName: "alpha",
			analyticsAggregateMetrics: analyticsAggregateMetrics{
				InputTokens:    120,
				OutputTokens:   80,
				TotalCost:      3.5,
				RequestSuccess: 5,
				RequestFailed:  1,
			},
		},
	}

	got := buildProviderBreakdown(rows, map[int]model.Channel{
		1: {ID: 1, Name: "alpha", Enabled: true},
		2: {ID: 2, Name: "beta", Enabled: true},
	})

	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0].ChannelID != 1 {
		t.Fatalf("expected channel 1, got %+v", got[0])
	}
	if got[0].RequestCount != 6 {
		t.Fatalf("expected historical request count to be preserved, got %+v", got[0])
	}
	if got[0].TotalTokens != 200 || got[0].TotalCost != 3.5 {
		t.Fatalf("expected historical token/cost totals to be preserved, got %+v", got[0])
	}
}

func TestLoadAnalyticsProviderRows_UsesRangeBoundedRelayLogs(t *testing.T) {
	restoreRelayLogs := snapshotRelayLogCacheState()
	defer restoreRelayLogs()
	restoreSettings := snapshotAnalyticsSettingState()
	defer restoreSettings()

	settingCache.Set(model.SettingKeyRelayLogKeepEnabled, "false")

	relayLogCacheLock.Lock()
	relayLogCache = []model.RelayLog{
		{
			Time:         time.Now().AddDate(0, 0, -2).Unix(),
			ChannelId:    1,
			ChannelName:  "alpha",
			InputTokens:  30,
			OutputTokens: 10,
			Cost:         0.5,
		},
		{
			Time:         time.Now().AddDate(0, 0, -40).Unix(),
			ChannelId:    1,
			ChannelName:  "alpha",
			InputTokens:  300,
			OutputTokens: 100,
			Cost:         5,
		},
	}
	relayLogCacheLock.Unlock()

	rows, err := loadAnalyticsProviderRows(context.Background(), model.AnalyticsRange30D)
	if err != nil {
		t.Fatalf("loadAnalyticsProviderRows() error = %v", err)
	}

	row := rows[1]
	if row == nil {
		t.Fatal("expected provider row for channel 1")
	}
	if row.RequestSuccess != 1 || row.RequestFailed != 0 {
		t.Fatalf("expected only in-range request totals, got %+v", row)
	}
	if row.InputTokens != 30 || row.OutputTokens != 10 {
		t.Fatalf("expected only in-range token totals, got %+v", row)
	}
	if row.TotalCost != 0.5 {
		t.Fatalf("expected only in-range cost total, got %+v", row)
	}
}

func TestBuildAPIKeyBreakdown_KeepsDuplicateNamesSeparatedByID(t *testing.T) {
	rows := map[string]*analyticsAPIKeyAggregateRow{
		"id:11": {
			APIKeyID: 11,
			Name:     "shared",
			analyticsAggregateMetrics: analyticsAggregateMetrics{
				RequestSuccess: 3,
			},
		},
		"id:22": {
			APIKeyID: 22,
			Name:     "shared",
			analyticsAggregateMetrics: analyticsAggregateMetrics{
				RequestSuccess: 1,
				RequestFailed:  1,
			},
		},
	}

	got := buildAPIKeyBreakdown(rows)
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].APIKeyID == nil || got[1].APIKeyID == nil {
		t.Fatalf("expected API key ids to be present, got %+v", got)
	}
	if *got[0].APIKeyID == *got[1].APIKeyID {
		t.Fatalf("expected duplicate names to remain separate, got %+v", got)
	}
}

func snapshotAnalyticsSettingState() func() {
	oldSettings := settingCache.GetAll()

	return func() {
		settingCache.Clear()
		for key, value := range oldSettings {
			settingCache.Set(key, value)
		}
	}
}

func TestStatsRefreshCache_LoadsStatsModels(t *testing.T) {
	restoreStats := snapshotStatsPersistenceState()
	defer restoreStats()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name()))
	if err := db.InitDB("sqlite", dsn, false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	modelStat := model.StatsModel{
		ID:        101,
		Name:      "gpt-4o",
		ChannelID: 9,
		StatsMetrics: model.StatsMetrics{
			RequestSuccess: 7,
			RequestFailed:  2,
		},
	}
	if err := db.GetDB().Create(&modelStat).Error; err != nil {
		t.Fatalf("seed stats model: %v", err)
	}

	statsModelCache.Clear()

	if err := statsRefreshCache(context.Background()); err != nil {
		t.Fatalf("statsRefreshCache() error = %v", err)
	}

	got := StatsModelList()
	if len(got) != 1 {
		t.Fatalf("StatsModelList() len = %d, want 1", len(got))
	}
	if got[0].ID != 101 || got[0].Name != "gpt-4o" || got[0].ChannelID != 9 {
		t.Fatalf("unexpected stats model loaded: %+v", got[0])
	}
	if got[0].RequestSuccess != 7 || got[0].RequestFailed != 2 {
		t.Fatalf("unexpected stats model metrics loaded: %+v", got[0])
	}
}

func snapshotRelayLogCacheState() func() {
	relayLogCacheLock.Lock()
	oldLogs := make([]model.RelayLog, len(relayLogCache))
	copy(oldLogs, relayLogCache)
	relayLogCacheLock.Unlock()

	return func() {
		relayLogCacheLock.Lock()
		relayLogCache = make([]model.RelayLog, len(oldLogs))
		copy(relayLogCache, oldLogs)
		relayLogCacheLock.Unlock()
	}
}
