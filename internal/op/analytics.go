package op

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/utils/semantic_cache"
)

const analyticsRouteHealthFailureWindow = 24 * time.Hour

type analyticsAggregateMetrics struct {
	InputTokens    int64
	OutputTokens   int64
	TotalCost      float64
	RequestSuccess int64
	RequestFailed  int64
}

type analyticsSummaryRow struct {
	analyticsAggregateMetrics
	RequestCount  int64
	FallbackCount int64
}

type analyticsProviderAggregateRow struct {
	ChannelID   int
	ChannelName string
	analyticsAggregateMetrics
}

type analyticsModelAggregateRow struct {
	ModelName string
	analyticsAggregateMetrics
}

type analyticsAPIKeyAggregateRow struct {
	APIKeyID int
	Name     string
	analyticsAggregateMetrics
}

type analyticsFailureAggregateRow struct {
	ChannelID        int
	RequestModelName string
	ActualModelName  string
	FailureCount     int64
	LastFailureAt    int64
}

func AnalyticsOverviewGet(ctx context.Context, r model.AnalyticsRange) (*model.AnalyticsOverview, error) {
	daily, err := StatsGetDaily(ctx)
	if err != nil {
		return nil, err
	}
	mergedDaily := mergeAnalyticsDailyWithToday(daily, StatsTodayGet())
	metrics := aggregateAnalyticsDailyMetrics(mergedDaily, r, time.Now())

	channels, err := ChannelList(ctx)
	if err != nil {
		return nil, err
	}
	apiKeys, err := APIKeyList(ctx)
	if err != nil {
		return nil, err
	}

	providerCount := 0
	modelNames := make(map[string]struct{})
	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}
		providerCount++
		for _, modelName := range splitAnalyticsChannelModels(channel) {
			modelNames[modelName] = struct{}{}
		}
	}

	apiKeyCount := 0
	for _, apiKey := range apiKeys {
		if apiKey.Enabled {
			apiKeyCount++
		}
	}

	logSummary, err := loadAnalyticsSummary(ctx, r)
	if err != nil {
		return nil, err
	}
	fallbackRate := 0.0
	if logSummary.RequestCount > 0 {
		fallbackRate = (float64(logSummary.FallbackCount) / float64(logSummary.RequestCount)) * 100
	}

	overview := buildAnalyticsOverview(metrics, providerCount, apiKeyCount, len(modelNames), fallbackRate)
	return &overview, nil
}

func AnalyticsUtilizationGet(ctx context.Context, r model.AnalyticsRange) (*model.AnalyticsUtilization, error) {
	providerBreakdown, err := AnalyticsProviderBreakdownGet(ctx, r)
	if err != nil {
		return nil, err
	}
	modelBreakdown, err := AnalyticsModelBreakdownGet(ctx, r)
	if err != nil {
		return nil, err
	}
	apiKeyBreakdown, err := AnalyticsAPIKeyBreakdownGet(ctx, r)
	if err != nil {
		return nil, err
	}

	return &model.AnalyticsUtilization{
		ProviderBreakdown: providerBreakdown,
		ModelBreakdown:    modelBreakdown,
		APIKeyBreakdown:   apiKeyBreakdown,
	}, nil
}

func AnalyticsProviderBreakdownGet(ctx context.Context, r model.AnalyticsRange) ([]model.AnalyticsProviderBreakdownItem, error) {
	rows, err := loadAnalyticsProviderRows(ctx, r)
	if err != nil {
		return nil, err
	}

	channels, err := ChannelList(ctx)
	if err != nil {
		return nil, err
	}
	channelByID := make(map[int]model.Channel, len(channels))
	for _, channel := range channels {
		channelByID[channel.ID] = channel
	}

	return buildProviderBreakdown(rows, channelByID), nil
}

func AnalyticsModelBreakdownGet(ctx context.Context, r model.AnalyticsRange) ([]model.AnalyticsModelBreakdownItem, error) {
	rows, err := loadAnalyticsModelRows(ctx, r)
	if err != nil {
		return nil, err
	}
	return buildModelBreakdown(rows), nil
}

func AnalyticsAPIKeyBreakdownGet(ctx context.Context, r model.AnalyticsRange) ([]model.AnalyticsAPIKeyBreakdownItem, error) {
	rows, err := loadAnalyticsAPIKeyRows(ctx, r)
	if err != nil {
		return nil, err
	}
	return buildAPIKeyBreakdown(rows), nil
}

func AnalyticsGroupHealthGet(ctx context.Context) ([]model.AnalyticsGroupHealthItem, error) {
	groups, err := GroupList(ctx)
	if err != nil {
		return nil, err
	}
	channels, err := ChannelList(ctx)
	if err != nil {
		return nil, err
	}

	channelByID := make(map[int]model.Channel, len(channels))
	for _, channel := range channels {
		channelByID[channel.ID] = channel
	}

	failures, err := loadAnalyticsFailureRows(ctx, time.Now().Add(-analyticsRouteHealthFailureWindow))
	if err != nil {
		return nil, err
	}

	return buildGroupHealth(groups, channelByID, failures), nil
}

func AnalyticsEvaluationGet(_ context.Context) (*model.AnalyticsEvaluationSummary, error) {
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

	hits, misses, currentEntries := semantic_cache.Stats()
	summary := &model.AnalyticsEvaluationSummary{
		SemanticCache: buildSemanticCacheEvaluationSummary(
			enabled,
			semantic_cache.RuntimeEnabled(),
			ttlSeconds,
			threshold,
			maxEntries,
			currentEntries,
			hits,
			misses,
			semantic_cache.GetRuntimeStats(),
		),
	}
	return summary, nil
}

func mergeAnalyticsDailyWithToday(daily []model.StatsDaily, today model.StatsDaily) []model.StatsDaily {
	if today.Date == "" {
		return daily
	}

	merged := make([]model.StatsDaily, 0, len(daily)+1)
	replaced := false
	for _, item := range daily {
		if item.Date == today.Date {
			merged = append(merged, today)
			replaced = true
			continue
		}
		merged = append(merged, item)
	}
	if !replaced {
		merged = append(merged, today)
	}
	return merged
}

func aggregateAnalyticsDailyMetrics(daily []model.StatsDaily, r model.AnalyticsRange, now time.Time) model.StatsMetrics {
	startDate := analyticsStartDate(r, now)
	var metrics model.StatsMetrics
	for _, item := range daily {
		if startDate != "" && item.Date < startDate {
			continue
		}
		metrics.Add(item.StatsMetrics)
	}
	return metrics
}

func buildAnalyticsOverview(metrics model.StatsMetrics, providerCount, apiKeyCount, modelCount int, fallbackRate float64) model.AnalyticsOverview {
	requestCount := metrics.RequestSuccess + metrics.RequestFailed
	successRate := 0.0
	if requestCount > 0 {
		successRate = (float64(metrics.RequestSuccess) / float64(requestCount)) * 100
	}

	return model.AnalyticsOverview{
		AnalyticsMetrics: model.AnalyticsMetrics{
			RequestCount: requestCount,
			TotalTokens:  metrics.InputToken + metrics.OutputToken,
			InputTokens:  metrics.InputToken,
			OutputTokens: metrics.OutputToken,
			TotalCost:    metrics.InputCost + metrics.OutputCost,
			SuccessRate:  successRate,
		},
		ProviderCount: providerCount,
		APIKeyCount:   apiKeyCount,
		ModelCount:    modelCount,
		FallbackRate:  fallbackRate,
	}
}

func buildProviderBreakdown(rows map[int]*analyticsProviderAggregateRow, channelByID map[int]model.Channel) []model.AnalyticsProviderBreakdownItem {
	items := make([]model.AnalyticsProviderBreakdownItem, 0, len(rows))
	for channelID, row := range rows {
		if row == nil {
			continue
		}

		requestCount := row.RequestSuccess + row.RequestFailed
		successRate := 0.0
		if requestCount > 0 {
			successRate = (float64(row.RequestSuccess) / float64(requestCount)) * 100
		}

		channelName := strings.TrimSpace(row.ChannelName)
		enabled := false
		if channel, ok := channelByID[channelID]; ok {
			if channelName == "" {
				channelName = channel.Name
			}
			enabled = channel.Enabled
		}
		if channelName == "" {
			channelName = "Unknown Channel"
		}

		items = append(items, model.AnalyticsProviderBreakdownItem{
			ChannelID:   channelID,
			ChannelName: channelName,
			Enabled:     enabled,
			AnalyticsMetrics: model.AnalyticsMetrics{
				RequestCount: requestCount,
				TotalTokens:  row.InputTokens + row.OutputTokens,
				InputTokens:  row.InputTokens,
				OutputTokens: row.OutputTokens,
				TotalCost:    row.TotalCost,
				SuccessRate:  successRate,
			},
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].RequestCount != items[j].RequestCount {
			return items[i].RequestCount > items[j].RequestCount
		}
		if items[i].TotalCost != items[j].TotalCost {
			return items[i].TotalCost > items[j].TotalCost
		}
		return items[i].ChannelName < items[j].ChannelName
	})

	return items
}

func buildModelBreakdown(rows map[string]*analyticsModelAggregateRow) []model.AnalyticsModelBreakdownItem {
	items := make([]model.AnalyticsModelBreakdownItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		modelName := strings.TrimSpace(row.ModelName)
		if modelName == "" {
			continue
		}

		requestCount := row.RequestSuccess + row.RequestFailed
		successRate := 0.0
		if requestCount > 0 {
			successRate = (float64(row.RequestSuccess) / float64(requestCount)) * 100
		}

		items = append(items, model.AnalyticsModelBreakdownItem{
			ModelName: modelName,
			AnalyticsMetrics: model.AnalyticsMetrics{
				RequestCount: requestCount,
				TotalTokens:  row.InputTokens + row.OutputTokens,
				InputTokens:  row.InputTokens,
				OutputTokens: row.OutputTokens,
				TotalCost:    row.TotalCost,
				SuccessRate:  successRate,
			},
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].RequestCount != items[j].RequestCount {
			return items[i].RequestCount > items[j].RequestCount
		}
		if items[i].TotalCost != items[j].TotalCost {
			return items[i].TotalCost > items[j].TotalCost
		}
		return items[i].ModelName < items[j].ModelName
	})

	return items
}

func buildAPIKeyBreakdown(rows map[string]*analyticsAPIKeyAggregateRow) []model.AnalyticsAPIKeyBreakdownItem {
	items := make([]model.AnalyticsAPIKeyBreakdownItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		name := strings.TrimSpace(row.Name)
		if name == "" {
			if row.APIKeyID > 0 {
				name = "Key #" + strconv.Itoa(row.APIKeyID)
			} else {
				name = "Unknown Key"
			}
		}

		requestCount := row.RequestSuccess + row.RequestFailed
		successRate := 0.0
		if requestCount > 0 {
			successRate = (float64(row.RequestSuccess) / float64(requestCount)) * 100
		}

		item := model.AnalyticsAPIKeyBreakdownItem{
			Name: name,
			AnalyticsMetrics: model.AnalyticsMetrics{
				RequestCount: requestCount,
				TotalTokens:  row.InputTokens + row.OutputTokens,
				InputTokens:  row.InputTokens,
				OutputTokens: row.OutputTokens,
				TotalCost:    row.TotalCost,
				SuccessRate:  successRate,
			},
		}
		if row.APIKeyID > 0 {
			id := row.APIKeyID
			item.APIKeyID = &id
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].RequestCount != items[j].RequestCount {
			return items[i].RequestCount > items[j].RequestCount
		}
		if items[i].TotalCost != items[j].TotalCost {
			return items[i].TotalCost > items[j].TotalCost
		}
		return items[i].Name < items[j].Name
	})

	return items
}

func buildGroupHealth(groups []model.Group, channelByID map[int]model.Channel, failures map[string]*analyticsFailureAggregateRow) []model.AnalyticsGroupHealthItem {
	items := make([]model.AnalyticsGroupHealthItem, 0, len(groups))
	for _, group := range groups {
		itemCount := len(group.Items)
		enabledItemCount := 0
		disabledItemCount := 0
		failureCount := int64(0)
		lastFailureAt := int64(0)

		seenFailureKeys := make(map[string]struct{})
		for _, item := range group.Items {
			channel, ok := channelByID[item.ChannelID]
			if ok && channel.Enabled {
				enabledItemCount++
			} else {
				disabledItemCount++
			}

			for _, key := range []string{
				makeAnalyticsFailureKey(item.ChannelID, item.ModelName, item.ModelName),
				makeAnalyticsFailureKey(item.ChannelID, item.ModelName, group.Name),
			} {
				if _, ok := seenFailureKeys[key]; ok {
					continue
				}
				seenFailureKeys[key] = struct{}{}
				failure, ok := failures[key]
				if !ok || failure == nil {
					continue
				}
				failureCount += failure.FailureCount
				if failure.LastFailureAt > lastFailureAt {
					lastFailureAt = failure.LastFailureAt
				}
			}
		}

		status := "healthy"
		score := 100
		switch {
		case itemCount == 0:
			status = "empty"
			score = 0
		case enabledItemCount == 0:
			status = "down"
			score = 20
		default:
			score -= (disabledItemCount * 40) / itemCount
			if failureCount > 0 {
				penalty := int(failureCount * 12)
				if penalty > 48 {
					penalty = 48
				}
				score -= penalty
			}
			if disabledItemCount > 0 || failureCount >= 3 {
				status = "degraded"
			} else if failureCount > 0 {
				status = "warning"
			}
		}

		if score < 0 {
			score = 0
		}

		items = append(items, model.AnalyticsGroupHealthItem{
			GroupID:           group.ID,
			GroupName:         group.Name,
			EndpointType:      group.EndpointType,
			ItemCount:         itemCount,
			EnabledItemCount:  enabledItemCount,
			DisabledItemCount: disabledItemCount,
			FailureCount:      failureCount,
			LastFailureAt:     lastFailureAt,
			HealthScore:       score,
			Status:            status,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].HealthScore != items[j].HealthScore {
			return items[i].HealthScore < items[j].HealthScore
		}
		if items[i].FailureCount != items[j].FailureCount {
			return items[i].FailureCount > items[j].FailureCount
		}
		return items[i].GroupName < items[j].GroupName
	})

	return items
}

func loadAnalyticsSummary(ctx context.Context, r model.AnalyticsRange) (*analyticsSummaryRow, error) {
	startUnix := analyticsRangeStartUnix(r, time.Now())
	row := &analyticsSummaryRow{}

	keepEnabled, err := SettingGetBool(model.SettingKeyRelayLogKeepEnabled)
	if err != nil {
		return nil, err
	}

	if keepEnabled {
		query := db.GetDB().WithContext(ctx).
			Model(&model.RelayLog{}).
			Select(`
				COUNT(*) AS request_count,
				COALESCE(SUM(CASE WHEN total_attempts > 1 THEN 1 ELSE 0 END), 0) AS fallback_count
			`)
		if startUnix != nil {
			query = query.Where("time >= ?", *startUnix)
		}
		if err := query.Scan(row).Error; err != nil {
			return nil, err
		}
	}

	relayLogCacheLock.Lock()
	for _, logItem := range relayLogCache {
		if startUnix != nil && logItem.Time < *startUnix {
			continue
		}
		row.RequestCount++
		if logItem.TotalAttempts > 1 {
			row.FallbackCount++
		}
	}
	relayLogCacheLock.Unlock()

	return row, nil
}

func loadAnalyticsProviderRows(ctx context.Context, r model.AnalyticsRange) (map[int]*analyticsProviderAggregateRow, error) {
	startUnix := analyticsRangeStartUnix(r, time.Now())
	rows := make(map[int]*analyticsProviderAggregateRow)

	keepEnabled, err := SettingGetBool(model.SettingKeyRelayLogKeepEnabled)
	if err != nil {
		return nil, err
	}

	if keepEnabled {
		var dbRows []analyticsProviderAggregateRow
		query := db.GetDB().WithContext(ctx).
			Model(&model.RelayLog{}).
			Select(`
				channel_id,
				channel_name,
				COALESCE(SUM(input_tokens), 0) AS input_tokens,
				COALESCE(SUM(output_tokens), 0) AS output_tokens,
				COALESCE(SUM(cost), 0) AS total_cost,
				COALESCE(SUM(CASE WHEN error = '' THEN 1 ELSE 0 END), 0) AS request_success,
				COALESCE(SUM(CASE WHEN error <> '' THEN 1 ELSE 0 END), 0) AS request_failed
			`).
			Group("channel_id, channel_name")
		if startUnix != nil {
			query = query.Where("time >= ?", *startUnix)
		}
		if err := query.Scan(&dbRows).Error; err != nil {
			return nil, err
		}
		for _, row := range dbRows {
			rowCopy := row
			rows[row.ChannelID] = &rowCopy
		}
	}

	relayLogCacheLock.Lock()
	for _, logItem := range relayLogCache {
		if startUnix != nil && logItem.Time < *startUnix {
			continue
		}
		row, ok := rows[logItem.ChannelId]
		if !ok {
			row = &analyticsProviderAggregateRow{
				ChannelID:   logItem.ChannelId,
				ChannelName: logItem.ChannelName,
			}
			rows[logItem.ChannelId] = row
		}
		row.InputTokens += int64(logItem.InputTokens)
		row.OutputTokens += int64(logItem.OutputTokens)
		row.TotalCost += logItem.Cost
		if logItem.Error == "" {
			row.RequestSuccess++
		} else {
			row.RequestFailed++
		}
		if row.ChannelName == "" {
			row.ChannelName = logItem.ChannelName
		}
	}
	relayLogCacheLock.Unlock()

	return rows, nil
}

func loadAnalyticsModelRows(ctx context.Context, r model.AnalyticsRange) (map[string]*analyticsModelAggregateRow, error) {
	startUnix := analyticsRangeStartUnix(r, time.Now())
	rows := make(map[string]*analyticsModelAggregateRow)

	keepEnabled, err := SettingGetBool(model.SettingKeyRelayLogKeepEnabled)
	if err != nil {
		return nil, err
	}

	if keepEnabled {
		var dbRows []analyticsModelAggregateRow
		modelExpr := "COALESCE(NULLIF(actual_model_name, ''), request_model_name)"
		query := db.GetDB().WithContext(ctx).
			Model(&model.RelayLog{}).
			Select(`
				` + modelExpr + ` AS model_name,
				COALESCE(SUM(input_tokens), 0) AS input_tokens,
				COALESCE(SUM(output_tokens), 0) AS output_tokens,
				COALESCE(SUM(cost), 0) AS total_cost,
				COALESCE(SUM(CASE WHEN error = '' THEN 1 ELSE 0 END), 0) AS request_success,
				COALESCE(SUM(CASE WHEN error <> '' THEN 1 ELSE 0 END), 0) AS request_failed
			`).
			Group(modelExpr)
		if startUnix != nil {
			query = query.Where("time >= ?", *startUnix)
		}
		if err := query.Scan(&dbRows).Error; err != nil {
			return nil, err
		}
		for _, row := range dbRows {
			modelName := strings.TrimSpace(row.ModelName)
			if modelName == "" {
				continue
			}
			rowCopy := row
			rowCopy.ModelName = modelName
			rows[modelName] = &rowCopy
		}
	}

	relayLogCacheLock.Lock()
	for _, logItem := range relayLogCache {
		if startUnix != nil && logItem.Time < *startUnix {
			continue
		}
		modelName := strings.TrimSpace(logItem.ActualModelName)
		if modelName == "" {
			modelName = strings.TrimSpace(logItem.RequestModelName)
		}
		if modelName == "" {
			continue
		}

		row, ok := rows[modelName]
		if !ok {
			row = &analyticsModelAggregateRow{ModelName: modelName}
			rows[modelName] = row
		}
		row.InputTokens += int64(logItem.InputTokens)
		row.OutputTokens += int64(logItem.OutputTokens)
		row.TotalCost += logItem.Cost
		if logItem.Error == "" {
			row.RequestSuccess++
		} else {
			row.RequestFailed++
		}
	}
	relayLogCacheLock.Unlock()

	return rows, nil
}

func loadAnalyticsAPIKeyRows(ctx context.Context, r model.AnalyticsRange) (map[string]*analyticsAPIKeyAggregateRow, error) {
	startUnix := analyticsRangeStartUnix(r, time.Now())
	rows := make(map[string]*analyticsAPIKeyAggregateRow)

	keepEnabled, err := SettingGetBool(model.SettingKeyRelayLogKeepEnabled)
	if err != nil {
		return nil, err
	}

	if keepEnabled {
		var dbRows []analyticsAPIKeyAggregateRow
		query := db.GetDB().WithContext(ctx).
			Model(&model.RelayLog{}).
			Select(`
				request_api_key_id AS api_key_id,
				request_api_key_name AS name,
				COALESCE(SUM(input_tokens), 0) AS input_tokens,
				COALESCE(SUM(output_tokens), 0) AS output_tokens,
				COALESCE(SUM(cost), 0) AS total_cost,
				COALESCE(SUM(CASE WHEN error = '' THEN 1 ELSE 0 END), 0) AS request_success,
				COALESCE(SUM(CASE WHEN error <> '' THEN 1 ELSE 0 END), 0) AS request_failed
			`).
			Group("request_api_key_id, request_api_key_name")
		if startUnix != nil {
			query = query.Where("time >= ?", *startUnix)
		}
		if err := query.Scan(&dbRows).Error; err != nil {
			return nil, err
		}
		for _, row := range dbRows {
			rowCopy := row
			rowCopy.Name = strings.TrimSpace(row.Name)
			rows[makeAnalyticsAPIKeyAggregateKey(row.APIKeyID, rowCopy.Name)] = &rowCopy
		}
	}

	relayLogCacheLock.Lock()
	for _, logItem := range relayLogCache {
		if startUnix != nil && logItem.Time < *startUnix {
			continue
		}
		apiKeyID := logItem.RequestAPIKeyID
		keyName := strings.TrimSpace(logItem.RequestAPIKeyName)
		aggregateKey := makeAnalyticsAPIKeyAggregateKey(apiKeyID, keyName)
		row, ok := rows[aggregateKey]
		if !ok {
			row = &analyticsAPIKeyAggregateRow{
				APIKeyID: apiKeyID,
				Name:     keyName,
			}
			rows[aggregateKey] = row
		}
		row.InputTokens += int64(logItem.InputTokens)
		row.OutputTokens += int64(logItem.OutputTokens)
		row.TotalCost += logItem.Cost
		if logItem.Error == "" {
			row.RequestSuccess++
		} else {
			row.RequestFailed++
		}
		if row.Name == "" {
			row.Name = keyName
		}
	}
	relayLogCacheLock.Unlock()

	return rows, nil
}

func makeAnalyticsAPIKeyAggregateKey(apiKeyID int, name string) string {
	if apiKeyID > 0 {
		return "id:" + strconv.Itoa(apiKeyID)
	}
	return "name:" + strings.TrimSpace(name)
}

func loadAnalyticsFailureRows(ctx context.Context, since time.Time) (map[string]*analyticsFailureAggregateRow, error) {
	startUnix := since.Unix()
	rows := make(map[string]*analyticsFailureAggregateRow)

	keepEnabled, err := SettingGetBool(model.SettingKeyRelayLogKeepEnabled)
	if err != nil {
		return nil, err
	}

	if keepEnabled {
		var dbRows []analyticsFailureAggregateRow
		query := db.GetDB().WithContext(ctx).
			Model(&model.RelayLog{}).
			Select(`
				channel_id,
				request_model_name,
				actual_model_name,
				COUNT(*) AS failure_count,
				MAX(time) AS last_failure_at
			`).
			Where("error <> ''").
			Where("time >= ?", startUnix).
			Group("channel_id, request_model_name, actual_model_name")
		if err := query.Scan(&dbRows).Error; err != nil {
			return nil, err
		}
		for _, row := range dbRows {
			key := makeAnalyticsFailureKey(row.ChannelID, row.ActualModelName, row.RequestModelName)
			rowCopy := row
			rows[key] = &rowCopy
		}
	}

	relayLogCacheLock.Lock()
	for _, logItem := range relayLogCache {
		if logItem.Error == "" || logItem.Time < startUnix {
			continue
		}
		key := makeAnalyticsFailureKey(logItem.ChannelId, logItem.ActualModelName, logItem.RequestModelName)
		row, ok := rows[key]
		if !ok {
			row = &analyticsFailureAggregateRow{
				ChannelID:        logItem.ChannelId,
				RequestModelName: logItem.RequestModelName,
				ActualModelName:  logItem.ActualModelName,
			}
			rows[key] = row
		}
		row.FailureCount++
		if logItem.Time > row.LastFailureAt {
			row.LastFailureAt = logItem.Time
		}
	}
	relayLogCacheLock.Unlock()

	return rows, nil
}

func analyticsRangeStartUnix(r model.AnalyticsRange, now time.Time) *int64 {
	startDate := analyticsStartTime(r, now)
	if startDate == nil {
		return nil
	}
	unix := startDate.Unix()
	return &unix
}

func analyticsStartDate(r model.AnalyticsRange, now time.Time) string {
	start := analyticsStartTime(r, now)
	if start == nil {
		return ""
	}
	return start.Format("20060102")
}

func analyticsStartTime(r model.AnalyticsRange, now time.Time) *time.Time {
	location := now.Location()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)

	switch r {
	case model.AnalyticsRange1D:
		return &dayStart
	case model.AnalyticsRange7D:
		start := dayStart.AddDate(0, 0, -6)
		return &start
	case model.AnalyticsRange30D:
		start := dayStart.AddDate(0, 0, -29)
		return &start
	case model.AnalyticsRange90D:
		start := dayStart.AddDate(0, 0, -89)
		return &start
	case model.AnalyticsRangeYTD:
		start := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, location)
		return &start
	case model.AnalyticsRangeAll:
		return nil
	default:
		start := dayStart.AddDate(0, 0, -6)
		return &start
	}
}

func splitAnalyticsChannelModels(channel model.Channel) []string {
	parts := strings.Split(channel.Model, ",")
	if strings.TrimSpace(channel.CustomModel) != "" {
		parts = append(parts, strings.Split(channel.CustomModel, ",")...)
	}

	seen := make(map[string]struct{}, len(parts))
	models := make([]string, 0, len(parts))
	for _, part := range parts {
		modelName := strings.TrimSpace(part)
		if modelName == "" {
			continue
		}
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		models = append(models, modelName)
	}
	return models
}

func makeAnalyticsFailureKey(channelID int, actualModelName, requestModelName string) string {
	actualModelName = strings.TrimSpace(actualModelName)
	if actualModelName == "" {
		actualModelName = strings.TrimSpace(requestModelName)
	}
	return strings.Join([]string{
		strconv.Itoa(channelID),
		actualModelName,
		strings.TrimSpace(requestModelName),
	}, "\x00")
}
