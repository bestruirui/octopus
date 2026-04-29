package op

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/utils/semantic_cache"
)

func TestNormalizeNavOrder_AppendsMissingRoutesAndDropsUnknown(t *testing.T) {
	defaults := []string{"home", "channel", "group", "model", "analytics", "log", "alert", "ops", "setting", "user"}
	got := NormalizeNavOrder(`["group","group","unknown","setting"]`, defaults)
	want := []string{"group", "setting", "home", "channel", "model", "analytics", "log", "alert", "ops", "user"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeNavOrder() = %v, want %v", got, want)
	}
}

func TestBuildSemanticCacheEvaluationSummary_ComputesRates(t *testing.T) {
	stats := semantic_cache.RuntimeStats{
		EvaluatedRequests: 12,
		CacheHitResponses: 8,
		CacheMissRequests: 3,
		BypassedRequests:  1,
		StoredResponses:   3,
	}
	got := buildSemanticCacheEvaluationSummary(
		true, true, 3600, 98, 1000, 120, 80, 40, stats,
	)
	if got.HitRate != 66.66666666666666 {
		t.Fatalf("HitRate = %v", got.HitRate)
	}
	if got.UsageRate != 12 {
		t.Fatalf("UsageRate = %v", got.UsageRate)
	}
}

func TestBuildOpsCacheStatus_ComputesRates(t *testing.T) {
	got := buildOpsCacheStatus(true, true, 3600, 98, 100, 3, 1, 25)

	if !got.Enabled || !got.RuntimeEnabled {
		t.Fatalf("expected cache to be enabled at config and runtime levels: %+v", got)
	}
	if got.HitRate != 75 {
		t.Fatalf("hit rate = %v, want 75", got.HitRate)
	}
	if got.UsageRate != 25 {
		t.Fatalf("usage rate = %v, want 25", got.UsageRate)
	}
}

func TestBuildOpsQuotaSummary_ClassifiesAndSortsKeys(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	keys := []model.APIKey{
		{
			ID:                1,
			Name:              "Budget key",
			Enabled:           true,
			MaxCost:           5,
			RateLimitRPM:      100,
			PerModelQuotaJSON: `{"gpt-4.1":1000}`,
		},
		{
			ID:           2,
			Name:         "Expired key",
			Enabled:      true,
			ExpireAt:     now.Unix() - 60,
			MaxCost:      10,
			RateLimitTPM: 2000,
		},
		{
			ID:      3,
			Name:    "Open key",
			Enabled: false,
		},
	}
	stats := []model.StatsAPIKey{
		{
			APIKeyID: 1,
			StatsMetrics: model.StatsMetrics{
				InputCost:      2,
				OutputCost:     4,
				RequestSuccess: 2,
			},
		},
		{
			APIKeyID: 2,
			StatsMetrics: model.StatsMetrics{
				InputCost:     1,
				OutputCost:    1,
				RequestFailed: 1,
			},
		},
	}

	got := buildOpsQuotaSummary(keys, stats, now)

	if got.TotalKeyCount != 3 || got.EnabledKeyCount != 2 {
		t.Fatalf("unexpected quota summary counts: %+v", got)
	}
	if got.ExhaustedKeyCount != 1 || got.ExpiredKeyCount != 1 {
		t.Fatalf("unexpected exhausted/expired counts: %+v", got)
	}
	if got.PerModelQuotaKeyCount != 1 || got.ActiveUsageKeyCount != 2 {
		t.Fatalf("unexpected quota flags: %+v", got)
	}
	if len(got.Keys) != 3 {
		t.Fatalf("keys length = %d, want 3", len(got.Keys))
	}
	if got.Keys[0].Status != "exhausted" || got.Keys[0].APIKeyID != 1 {
		t.Fatalf("expected exhausted key first, got %+v", got.Keys[0])
	}
	if got.Keys[1].Status != "expired" || got.Keys[1].APIKeyID != 2 {
		t.Fatalf("expected expired key second, got %+v", got.Keys[1])
	}
	if got.Keys[2].Status != "disabled" || got.Keys[2].APIKeyID != 3 {
		t.Fatalf("expected disabled key last, got %+v", got.Keys[2])
	}
}

func TestBuildOpsHealthStatus_CountsAndLimitsFailingGroups(t *testing.T) {
	groupHealth := []model.AnalyticsGroupHealthItem{
		{GroupID: 1, GroupName: "g1", Status: "down", FailureCount: 5, HealthScore: 10},
		{GroupID: 2, GroupName: "g2", Status: "degraded", FailureCount: 3, HealthScore: 40},
		{GroupID: 3, GroupName: "g3", Status: "warning", FailureCount: 1, HealthScore: 70},
		{GroupID: 4, GroupName: "g4", Status: "empty", FailureCount: 0, HealthScore: 0},
		{GroupID: 5, GroupName: "g5", Status: "healthy", FailureCount: 0, HealthScore: 100},
		{GroupID: 6, GroupName: "g6", Status: "down", FailureCount: 8, HealthScore: 5},
		{GroupID: 7, GroupName: "g7", Status: "warning", FailureCount: 2, HealthScore: 65},
		{GroupID: 8, GroupName: "g8", Status: "degraded", FailureCount: 4, HealthScore: 30},
	}

	got := buildOpsHealthStatus(true, true, true, 9, groupHealth, time.Unix(1_700_000_000, 0))

	if got.RecentErrorCount != 9 || !got.DatabaseOK || !got.CacheOK || !got.TaskRuntimeOK {
		t.Fatalf("unexpected base health status: %+v", got)
	}
	if got.HealthyGroupCount != 1 || got.WarningGroupCount != 2 || got.DegradedGroupCount != 2 || got.DownGroupCount != 2 || got.EmptyGroupCount != 1 {
		t.Fatalf("unexpected group counts: %+v", got)
	}
	if len(got.FailingGroups) != opsFailingGroupLimit {
		t.Fatalf("failing group length = %d, want %d", len(got.FailingGroups), opsFailingGroupLimit)
	}
	if got.FailingGroups[0].GroupID != 1 || got.FailingGroups[1].GroupID != 2 {
		t.Fatalf("expected failing groups to preserve worst-first order, got %+v", got.FailingGroups)
	}
}

func TestBuildOpsAIRouteServices_ReturnsEmptySliceForNilConfigs(t *testing.T) {
	got := buildOpsAIRouteServices(nil)

	if got == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected no services, got %d", len(got))
	}
}

func TestRefreshSemanticCacheRuntime_ResetsDisabledOrIncompleteConfig(t *testing.T) {
	restore := snapshotSettingCache()
	defer restore()
	semantic_cache.Reset()
	semantic_cache.ResetRuntimeStats()
	defer semantic_cache.Reset()
	defer semantic_cache.ResetRuntimeStats()

	semantic_cache.ApplyRuntimeConfig(semantic_cache.RuntimeConfig{
		Enabled:          true,
		MaxEntries:       8,
		Threshold:        0.98,
		TTL:              time.Hour,
		EmbeddingBaseURL: "https://example.com",
		EmbeddingModel:   "text-embedding-3-small",
	})
	if !semantic_cache.Enabled() {
		t.Fatal("expected seeded runtime cache to be enabled")
	}

	seedDefaultSettingsForTest(map[model.SettingKey]string{
		model.SettingKeySemanticCacheEnabled: "false",
	})
	if err := RefreshSemanticCacheRuntime(); err != nil {
		t.Fatalf("RefreshSemanticCacheRuntime() disabled config error = %v", err)
	}
	if semantic_cache.RuntimeEnabled() {
		t.Fatal("expected disabled setting to clear semantic cache runtime")
	}

	semantic_cache.ApplyRuntimeConfig(semantic_cache.RuntimeConfig{
		Enabled:          true,
		MaxEntries:       8,
		Threshold:        0.98,
		TTL:              time.Hour,
		EmbeddingBaseURL: "https://example.com",
		EmbeddingModel:   "text-embedding-3-small",
	})
	seedDefaultSettingsForTest(map[model.SettingKey]string{
		model.SettingKeySemanticCacheEnabled:          "true",
		model.SettingKeySemanticCacheEmbeddingBaseURL: "",
		model.SettingKeySemanticCacheEmbeddingModel:   "text-embedding-3-small",
	})
	if err := RefreshSemanticCacheRuntime(); err != nil {
		t.Fatalf("RefreshSemanticCacheRuntime() incomplete config error = %v", err)
	}
	if semantic_cache.RuntimeEnabled() {
		t.Fatal("expected incomplete config to clear semantic cache runtime")
	}
}

func TestAnalyticsEvaluationGet_ReturnsSemanticCacheSummary(t *testing.T) {
	restore := snapshotSettingCache()
	defer restore()
	semantic_cache.Reset()
	semantic_cache.ResetRuntimeStats()
	defer semantic_cache.Reset()
	defer semantic_cache.ResetRuntimeStats()

	seedDefaultSettingsForTest(map[model.SettingKey]string{
		model.SettingKeySemanticCacheEnabled:                 "true",
		model.SettingKeySemanticCacheTTL:                     "7200",
		model.SettingKeySemanticCacheThreshold:               "97",
		model.SettingKeySemanticCacheMaxEntries:              "50",
		model.SettingKeySemanticCacheEmbeddingBaseURL:        "https://example.com/v1",
		model.SettingKeySemanticCacheEmbeddingAPIKey:         "test-key",
		model.SettingKeySemanticCacheEmbeddingModel:          "text-embedding-3-small",
		model.SettingKeySemanticCacheEmbeddingTimeoutSeconds: "12",
	})

	if err := RefreshSemanticCacheRuntime(); err != nil {
		t.Fatalf("RefreshSemanticCacheRuntime() error = %v", err)
	}

	embedding := []float64{1, 0}
	semantic_cache.Store("ns", "req-1", []byte(`{"ok":true}`), embedding)
	if _, ok := semantic_cache.Lookup("ns", embedding); !ok {
		t.Fatal("expected cached lookup hit")
	}
	if _, ok := semantic_cache.Lookup("ns", []float64{0, 1}); ok {
		t.Fatal("expected cache miss for different embedding")
	}

	for i := 0; i < 4; i++ {
		semantic_cache.RecordEvaluated()
	}
	semantic_cache.RecordHit()
	semantic_cache.RecordMiss()
	semantic_cache.RecordMiss()
	semantic_cache.RecordBypass()
	semantic_cache.RecordStored()
	semantic_cache.RecordStored()

	got, err := AnalyticsEvaluationGet(context.Background())
	if err != nil {
		t.Fatalf("AnalyticsEvaluationGet() error = %v", err)
	}

	summary := got.SemanticCache
	if !summary.Enabled || !summary.RuntimeEnabled {
		t.Fatalf("expected semantic cache to be enabled in summary: %+v", summary)
	}
	if summary.TTLSeconds != 7200 || summary.Threshold != 97 || summary.MaxEntries != 50 {
		t.Fatalf("unexpected config summary: %+v", summary)
	}
	if summary.CurrentEntries != 1 || summary.Hits != 1 || summary.Misses != 1 {
		t.Fatalf("unexpected cache stats summary: %+v", summary)
	}
	if summary.HitRate != 50 || summary.UsageRate != 2 {
		t.Fatalf("unexpected rates summary: %+v", summary)
	}
	if summary.EvaluatedRequests != 4 || summary.CacheHitResponses != 1 || summary.CacheMissRequests != 2 || summary.BypassedRequests != 1 || summary.StoredResponses != 2 {
		t.Fatalf("unexpected runtime stats summary: %+v", summary)
	}
}

func snapshotSettingCache() func() {
	snapshot := settingCache.GetAll()
	return func() {
		settingCache.Clear()
		for key, value := range snapshot {
			settingCache.Set(key, value)
		}
	}
}

func seedDefaultSettingsForTest(overrides map[model.SettingKey]string) {
	settingCache.Clear()
	for _, setting := range model.DefaultSettings() {
		value := setting.Value
		if override, ok := overrides[setting.Key]; ok {
			value = override
		}
		settingCache.Set(setting.Key, value)
	}
	for key, value := range overrides {
		if _, exists := settingCache.Get(key); exists {
			continue
		}
		settingCache.Set(key, value)
	}
}
