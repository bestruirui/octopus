package op

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/lingyuins/octopus/internal/conf"
	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/utils/semantic_cache"
)

const (
	opsHealthErrorWindow           = 24 * time.Hour
	opsFailingGroupLimit           = 6
	semanticCacheDefaultTTLSeconds = 3600
	semanticCacheDefaultThreshold  = 98
	semanticCacheDefaultMaxEntries = 1000
	semanticCacheDefaultTimeoutSec = 10
)

type opsQuotaUsage struct {
	RequestCount int64
	TotalCost    float64
}

func OpsCacheStatusGet(ctx context.Context) (*model.OpsCacheStatus, error) {
	enabled, err := SettingGetBool(model.SettingKeySemanticCacheEnabled)
	if err != nil {
		return nil, err
	}
	ttlSeconds, err := SettingGetInt(model.SettingKeySemanticCacheTTL)
	if err != nil {
		return nil, err
	}
	threshold, err := SettingGetInt(model.SettingKeySemanticCacheThreshold)
	if err != nil {
		return nil, err
	}
	maxEntries, err := SettingGetInt(model.SettingKeySemanticCacheMaxEntries)
	if err != nil {
		return nil, err
	}

	hits, misses, size := semantic_cache.Stats()
	status := buildOpsCacheStatus(enabled, semantic_cache.RuntimeEnabled(), ttlSeconds, threshold, maxEntries, hits, misses, size)
	return &status, nil
}

func RefreshSemanticCacheRuntime() error {
	cfg, ok, err := buildSemanticCacheRuntimeConfigFromSettings()
	if err != nil {
		return err
	}
	if !ok {
		semantic_cache.Reset()
		return nil
	}
	semantic_cache.ApplyRuntimeConfig(cfg)
	return nil
}

func OpsQuotaSummaryGet(ctx context.Context) (*model.OpsQuotaSummary, error) {
	apiKeys, err := APIKeyList(ctx)
	if err != nil {
		return nil, err
	}

	summary := buildOpsQuotaSummary(apiKeys, StatsAPIKeyList(), time.Now())
	return &summary, nil
}

func OpsHealthStatusGet(ctx context.Context) (*model.OpsHealthStatus, error) {
	cacheStatus, err := OpsCacheStatusGet(ctx)
	if err != nil {
		return nil, err
	}

	recentErrorCount, err := loadOpsRecentErrorCount(ctx, time.Now().Add(-opsHealthErrorWindow))
	if err != nil {
		return nil, err
	}

	groupHealth, err := AnalyticsGroupHealthGet(ctx)
	if err != nil {
		return nil, err
	}

	status := buildOpsHealthStatus(
		pingDatabase(ctx),
		!cacheStatus.Enabled || cacheStatus.RuntimeEnabled,
		opsTaskRuntimeOK(),
		recentErrorCount,
		groupHealth,
		time.Now(),
	)
	return &status, nil
}

func OpsSystemSummaryGet(ctx context.Context) (*model.OpsSystemSummary, error) {
	proxyURL, err := SettingGetString(model.SettingKeyProxyURL)
	if err != nil {
		return nil, err
	}
	publicAPIBaseURL, err := SettingGetString(model.SettingKeyPublicAPIBaseURL)
	if err != nil {
		return nil, err
	}
	relayLogKeepEnabled, err := SettingGetBool(model.SettingKeyRelayLogKeepEnabled)
	if err != nil {
		return nil, err
	}
	relayLogKeepDays, err := SettingGetInt(model.SettingKeyRelayLogKeepPeriod)
	if err != nil {
		return nil, err
	}
	statsSaveIntervalMinutes, err := SettingGetInt(model.SettingKeyStatsSaveInterval)
	if err != nil {
		return nil, err
	}
	syncLLMIntervalHours, err := SettingGetInt(model.SettingKeySyncLLMInterval)
	if err != nil {
		return nil, err
	}
	modelInfoUpdateIntervalHours, err := SettingGetInt(model.SettingKeyModelInfoUpdateInterval)
	if err != nil {
		return nil, err
	}
	aiRouteGroupID, err := SettingGetInt(model.SettingKeyAIRouteGroupID)
	if err != nil {
		return nil, err
	}
	aiRouteTimeoutSeconds, err := SettingGetInt(model.SettingKeyAIRouteTimeoutSeconds)
	if err != nil {
		return nil, err
	}
	aiRouteParallelism, err := SettingGetInt(model.SettingKeyAIRouteParallelism)
	if err != nil {
		return nil, err
	}

	channels, err := ChannelList(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := GroupList(ctx)
	if err != nil {
		return nil, err
	}
	apiKeys, err := APIKeyList(ctx)
	if err != nil {
		return nil, err
	}

	aiRouteServices, legacyMode := loadOpsAIRouteServicesSummary()
	enabledServiceCount := 0
	for _, service := range aiRouteServices {
		if service.Enabled {
			enabledServiceCount++
		}
	}

	summary := &model.OpsSystemSummary{
		Version:                      conf.Version,
		Commit:                       conf.Commit,
		BuildTime:                    conf.BuildTime,
		Repo:                         conf.Repo,
		DatabaseType:                 conf.AppConfig.Database.Type,
		PublicAPIBaseURL:             strings.TrimSpace(publicAPIBaseURL),
		ProxyURL:                     strings.TrimSpace(proxyURL),
		RelayLogKeepEnabled:          relayLogKeepEnabled,
		RelayLogKeepDays:             relayLogKeepDays,
		StatsSaveIntervalMinutes:     statsSaveIntervalMinutes,
		SyncLLMIntervalHours:         syncLLMIntervalHours,
		ModelInfoUpdateIntervalHours: modelInfoUpdateIntervalHours,
		ImportEnabled:                true,
		ExportEnabled:                true,
		AIRouteGroupID:               aiRouteGroupID,
		AIRouteTimeoutSeconds:        aiRouteTimeoutSeconds,
		AIRouteParallelism:           aiRouteParallelism,
		AIRouteLegacyMode:            legacyMode,
		AIRouteServiceCount:          len(aiRouteServices),
		AIRouteEnabledServiceCount:   enabledServiceCount,
		AIRouteServices:              aiRouteServices,
		ChannelCount:                 len(channels),
		GroupCount:                   len(groups),
		APIKeyCount:                  len(apiKeys),
	}
	return summary, nil
}

func buildOpsCacheStatus(
	enabled bool,
	runtimeEnabled bool,
	ttlSeconds int,
	threshold int,
	maxEntries int,
	hits int64,
	misses int64,
	size int,
) model.OpsCacheStatus {
	totalLookups := hits + misses
	hitRate := 0.0
	if totalLookups > 0 {
		hitRate = (float64(hits) / float64(totalLookups)) * 100
	}

	usageRate := 0.0
	if maxEntries > 0 {
		usageRate = (float64(size) / float64(maxEntries)) * 100
	}

	return model.OpsCacheStatus{
		Enabled:        enabled,
		RuntimeEnabled: runtimeEnabled,
		TTLSeconds:     ttlSeconds,
		Threshold:      threshold,
		MaxEntries:     maxEntries,
		CurrentEntries: size,
		Hits:           hits,
		Misses:         misses,
		HitRate:        hitRate,
		UsageRate:      usageRate,
	}
}

func buildSemanticCacheRuntimeConfigFromSettings() (semantic_cache.RuntimeConfig, bool, error) {
	enabled, err := SettingGetBool(model.SettingKeySemanticCacheEnabled)
	if err != nil {
		return semantic_cache.RuntimeConfig{}, false, err
	}
	if !enabled {
		return semantic_cache.RuntimeConfig{}, false, nil
	}

	ttlSeconds, err := SettingGetInt(model.SettingKeySemanticCacheTTL)
	if err != nil || ttlSeconds <= 0 {
		ttlSeconds = semanticCacheDefaultTTLSeconds
	}

	thresholdRaw, err := SettingGetInt(model.SettingKeySemanticCacheThreshold)
	if err != nil || thresholdRaw < 0 || thresholdRaw > 100 {
		thresholdRaw = semanticCacheDefaultThreshold
	}

	maxEntries, err := SettingGetInt(model.SettingKeySemanticCacheMaxEntries)
	if err != nil || maxEntries <= 0 {
		maxEntries = semanticCacheDefaultMaxEntries
	}

	baseURL, err := SettingGetString(model.SettingKeySemanticCacheEmbeddingBaseURL)
	if err != nil {
		return semantic_cache.RuntimeConfig{}, false, err
	}
	modelName, err := SettingGetString(model.SettingKeySemanticCacheEmbeddingModel)
	if err != nil {
		return semantic_cache.RuntimeConfig{}, false, err
	}
	baseURL = strings.TrimSpace(baseURL)
	modelName = strings.TrimSpace(modelName)
	if baseURL == "" || modelName == "" {
		return semantic_cache.RuntimeConfig{}, false, nil
	}

	apiKey, err := SettingGetString(model.SettingKeySemanticCacheEmbeddingAPIKey)
	if err != nil {
		return semantic_cache.RuntimeConfig{}, false, err
	}

	timeoutSeconds, err := SettingGetInt(model.SettingKeySemanticCacheEmbeddingTimeoutSeconds)
	if err != nil || timeoutSeconds <= 0 {
		timeoutSeconds = semanticCacheDefaultTimeoutSec
	}

	return semantic_cache.RuntimeConfig{
		Enabled:          true,
		MaxEntries:       maxEntries,
		Threshold:        float64(thresholdRaw) / 100.0,
		TTL:              time.Duration(ttlSeconds) * time.Second,
		EmbeddingBaseURL: baseURL,
		EmbeddingAPIKey:  strings.TrimSpace(apiKey),
		EmbeddingModel:   modelName,
		EmbeddingTimeout: time.Duration(timeoutSeconds) * time.Second,
	}, true, nil
}

func buildOpsQuotaSummary(apiKeys []model.APIKey, stats []model.StatsAPIKey, now time.Time) model.OpsQuotaSummary {
	usageByKeyID := make(map[int]opsQuotaUsage, len(stats))
	for _, stat := range stats {
		usageByKeyID[stat.APIKeyID] = opsQuotaUsage{
			RequestCount: stat.RequestSuccess + stat.RequestFailed,
			TotalCost:    stat.InputCost + stat.OutputCost,
		}
	}

	items := make([]model.OpsQuotaKeyItem, 0, len(apiKeys))
	summary := model.OpsQuotaSummary{
		TotalKeyCount: len(apiKeys),
	}

	nowUnix := now.Unix()
	for _, apiKey := range apiKeys {
		usage := usageByKeyID[apiKey.ID]
		expired := apiKey.ExpireAt > 0 && apiKey.ExpireAt <= nowUnix
		hasPerModelQuota := hasPerModelQuota(apiKey.PerModelQuotaJSON)
		supportedModelCount := countCommaSeparated(apiKey.SupportedModels)
		limited := apiKey.RateLimitRPM > 0 || apiKey.RateLimitTPM > 0 || apiKey.MaxCost > 0 || hasPerModelQuota
		exhausted := apiKey.MaxCost > 0 && usage.TotalCost >= apiKey.MaxCost

		status := "open"
		switch {
		case !apiKey.Enabled:
			status = "disabled"
		case expired:
			status = "expired"
		case exhausted:
			status = "exhausted"
		case limited:
			status = "limited"
		}

		if apiKey.Enabled {
			summary.EnabledKeyCount++
		}
		if apiKey.Enabled && !expired {
			summary.AvailableKeyCount++
		}
		if expired {
			summary.ExpiredKeyCount++
		}
		if limited {
			summary.LimitedKeyCount++
		} else {
			summary.UnlimitedKeyCount++
		}
		if exhausted {
			summary.ExhaustedKeyCount++
		}
		if hasPerModelQuota {
			summary.PerModelQuotaKeyCount++
		}
		if usage.RequestCount > 0 {
			summary.ActiveUsageKeyCount++
		}
		if apiKey.RateLimitRPM > 0 {
			summary.TotalRPM += apiKey.RateLimitRPM
		}
		if apiKey.RateLimitTPM > 0 {
			summary.TotalTPM += apiKey.RateLimitTPM
		}
		if apiKey.MaxCost > 0 {
			summary.TotalMaxCost += apiKey.MaxCost
		}

		name := strings.TrimSpace(apiKey.Name)
		if name == "" {
			name = "Key"
		}
		items = append(items, model.OpsQuotaKeyItem{
			APIKeyID:            apiKey.ID,
			Name:                name,
			Enabled:             apiKey.Enabled,
			Expired:             expired,
			Status:              status,
			SupportedModelCount: supportedModelCount,
			HasPerModelQuota:    hasPerModelQuota,
			RateLimitRPM:        apiKey.RateLimitRPM,
			RateLimitTPM:        apiKey.RateLimitTPM,
			MaxCost:             apiKey.MaxCost,
			RequestCount:        usage.RequestCount,
			TotalCost:           usage.TotalCost,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		leftRank := opsQuotaStatusRank(items[i].Status)
		rightRank := opsQuotaStatusRank(items[j].Status)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if items[i].RequestCount != items[j].RequestCount {
			return items[i].RequestCount > items[j].RequestCount
		}
		if items[i].TotalCost != items[j].TotalCost {
			return items[i].TotalCost > items[j].TotalCost
		}
		return items[i].Name < items[j].Name
	})

	summary.Keys = items
	return summary
}

func buildOpsHealthStatus(
	databaseOK bool,
	cacheOK bool,
	taskRuntimeOK bool,
	recentErrorCount int64,
	groupHealth []model.AnalyticsGroupHealthItem,
	now time.Time,
) model.OpsHealthStatus {
	status := model.OpsHealthStatus{
		DatabaseOK:       databaseOK,
		CacheOK:          cacheOK,
		TaskRuntimeOK:    taskRuntimeOK,
		RecentErrorCount: recentErrorCount,
		CheckedAt:        now.Unix(),
	}

	failingGroups := make([]model.OpsHealthGroupItem, 0, opsFailingGroupLimit)
	for _, group := range groupHealth {
		switch group.Status {
		case "healthy":
			status.HealthyGroupCount++
		case "warning":
			status.WarningGroupCount++
		case "degraded":
			status.DegradedGroupCount++
		case "down":
			status.DownGroupCount++
		case "empty":
			status.EmptyGroupCount++
		}

		if group.Status == "healthy" {
			continue
		}
		if len(failingGroups) >= opsFailingGroupLimit {
			continue
		}
		failingGroups = append(failingGroups, model.OpsHealthGroupItem{
			GroupID:      group.GroupID,
			GroupName:    group.GroupName,
			EndpointType: group.EndpointType,
			Status:       group.Status,
			FailureCount: group.FailureCount,
			HealthScore:  group.HealthScore,
		})
	}

	status.FailingGroups = failingGroups
	return status
}

func pingDatabase(ctx context.Context) bool {
	gormDB := db.GetDB()
	if gormDB == nil {
		return false
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return false
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return false
	}
	return true
}

func opsTaskRuntimeOK() bool {
	statsSaveIntervalMinutes, err := SettingGetInt(model.SettingKeyStatsSaveInterval)
	if err != nil || statsSaveIntervalMinutes < 1 {
		return false
	}
	syncLLMIntervalHours, err := SettingGetInt(model.SettingKeySyncLLMInterval)
	if err != nil || syncLLMIntervalHours < 1 {
		return false
	}
	modelInfoUpdateIntervalHours, err := SettingGetInt(model.SettingKeyModelInfoUpdateInterval)
	if err != nil || modelInfoUpdateIntervalHours < 1 {
		return false
	}
	if _, err := SettingGetBool(model.SettingKeyRelayLogKeepEnabled); err != nil {
		return false
	}
	return true
}

func loadOpsRecentErrorCount(ctx context.Context, since time.Time) (int64, error) {
	startUnix := since.Unix()
	var errorCount int64

	keepEnabled, err := SettingGetBool(model.SettingKeyRelayLogKeepEnabled)
	if err != nil {
		return 0, err
	}

	if keepEnabled {
		if err := db.GetDB().WithContext(ctx).
			Model(&model.RelayLog{}).
			Where("error <> ''").
			Where("time >= ?", startUnix).
			Count(&errorCount).Error; err != nil {
			return 0, err
		}
	}

	relayLogCacheLock.Lock()
	for _, logItem := range relayLogCache {
		if logItem.Error != "" && logItem.Time >= startUnix {
			errorCount++
		}
	}
	relayLogCacheLock.Unlock()

	return errorCount, nil
}

func loadOpsAIRouteServicesSummary() ([]model.OpsAIRouteServiceSummary, bool) {
	rawServices, _ := SettingGetString(model.SettingKeyAIRouteServices)
	rawServices = strings.TrimSpace(rawServices)
	if rawServices != "" && rawServices != "[]" {
		var configs []model.AIRouteServiceConfig
		if err := json.Unmarshal([]byte(rawServices), &configs); err == nil {
			return buildOpsAIRouteServices(configs), false
		}
	}

	baseURL, _ := SettingGetString(model.SettingKeyAIRouteBaseURL)
	apiKey, _ := SettingGetString(model.SettingKeyAIRouteAPIKey)
	modelName, _ := SettingGetString(model.SettingKeyAIRouteModel)
	if strings.TrimSpace(baseURL) == "" && strings.TrimSpace(apiKey) == "" && strings.TrimSpace(modelName) == "" {
		return buildOpsAIRouteServices(nil), false
	}

	enabled := strings.TrimSpace(baseURL) != "" && strings.TrimSpace(apiKey) != "" && strings.TrimSpace(modelName) != ""
	configs := []model.AIRouteServiceConfig{{
		Name:    "legacy",
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   modelName,
		Enabled: &enabled,
	}}
	return buildOpsAIRouteServices(configs), true
}

func buildOpsAIRouteServices(configs []model.AIRouteServiceConfig) []model.OpsAIRouteServiceSummary {
	services := make([]model.OpsAIRouteServiceSummary, 0, len(configs))
	for i, cfg := range configs {
		services = append(services, model.OpsAIRouteServiceSummary{
			Name:    normalizeAIRouteServiceName(cfg, i+1),
			BaseURL: strings.TrimSpace(cfg.BaseURL),
			Model:   strings.TrimSpace(cfg.Model),
			Enabled: cfg.IsEnabled(),
		})
	}

	sort.SliceStable(services, func(i, j int) bool {
		if services[i].Enabled != services[j].Enabled {
			return services[i].Enabled
		}
		return services[i].Name < services[j].Name
	})
	return services
}

func hasPerModelQuota(raw string) bool {
	value := strings.TrimSpace(raw)
	return value != "" && value != "{}" && value != "null"
}

func countCommaSeparated(raw string) int {
	if strings.TrimSpace(raw) == "" {
		return 0
	}

	seen := make(map[string]struct{})
	count := 0
	for _, part := range strings.Split(raw, ",") {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		count++
	}
	return count
}

func opsQuotaStatusRank(status string) int {
	switch status {
	case "exhausted":
		return 0
	case "expired":
		return 1
	case "disabled":
		return 2
	case "limited":
		return 3
	default:
		return 4
	}
}
