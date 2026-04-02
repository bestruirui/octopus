package op

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/bestruirui/octopus/internal/utils/xstrings"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var statsDailyCache model.StatsDaily
var statsDailyCacheLock sync.RWMutex

var statsTotalCache model.StatsTotal
var statsTotalCacheLock sync.RWMutex

var statsHourlyCache [24]model.StatsHourly
var statsHourlyCacheLock sync.RWMutex

var statsChannelCache = cache.New[int, model.StatsChannel](16)
var statsChannelCacheNeedUpdate = make(map[int]struct{})
var statsChannelCacheNeedUpdateLock sync.Mutex

var statsModelCache = cache.New[int, model.StatsModel](16)
var statsModelCacheNeedUpdate = make(map[int]struct{})
var statsModelCacheNeedUpdateLock sync.Mutex

var statsAPIKeyCache = cache.New[int, model.StatsAPIKey](16)
var statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
var statsAPIKeyCacheNeedUpdateLock sync.Mutex

func StatsSaveDBTask() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	log.Debugf("stats save db task started")
	startTime := time.Now()
	defer func() {
		log.Debugf("stats save db task finished, save time: %s", time.Since(startTime))
	}()
	if err := StatsSaveDB(ctx); err != nil {
		log.Errorf("stats save db error: %v", err)
		return
	}
}

func StatsSaveDB(ctx context.Context) error {
	statsTotalCacheLock.RLock()
	totalSnap := statsTotalCache
	statsTotalCacheLock.RUnlock()
	if totalSnap.ID == 0 {
		totalSnap.ID = 1
	}

	statsDailyCacheLock.RLock()
	dailySnap := statsDailyCache
	statsDailyCacheLock.RUnlock()

	statsHourlyCacheLock.RLock()
	hourlyAll := statsHourlyCache
	statsHourlyCacheLock.RUnlock()

	statsChannelCacheNeedUpdateLock.Lock()
	channelIDs := make([]int, 0, len(statsChannelCacheNeedUpdate))
	for id := range statsChannelCacheNeedUpdate {
		channelIDs = append(channelIDs, id)
	}
	statsChannelCacheNeedUpdate = make(map[int]struct{})
	statsChannelCacheNeedUpdateLock.Unlock()

	statsModelCacheNeedUpdateLock.Lock()
	modelIDs := make([]int, 0, len(statsModelCacheNeedUpdate))
	for id := range statsModelCacheNeedUpdate {
		modelIDs = append(modelIDs, id)
	}
	statsModelCacheNeedUpdate = make(map[int]struct{})
	statsModelCacheNeedUpdateLock.Unlock()

	statsAPIKeyCacheNeedUpdateLock.Lock()
	apiKeyIDs := make([]int, 0, len(statsAPIKeyCacheNeedUpdate))
	for id := range statsAPIKeyCacheNeedUpdate {
		apiKeyIDs = append(apiKeyIDs, id)
	}
	statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
	statsAPIKeyCacheNeedUpdateLock.Unlock()

	return persistStatsSnapshots(ctx, totalSnap, dailySnap, hourlyAll, channelIDs, modelIDs, apiKeyIDs)
}

func persistStatsSnapshots(
	ctx context.Context,
	totalSnap model.StatsTotal,
	dailySnap model.StatsDaily,
	hourlyAll [24]model.StatsHourly,
	channelIDs []int,
	modelIDs []int,
	apiKeyIDs []int,
) error {
	dbConn := db.GetDB().WithContext(ctx)

	if result := dbConn.Save(&totalSnap); result.Error != nil {
		return result.Error
	}
	if result := dbConn.Save(&dailySnap); result.Error != nil {
		return result.Error
	}

	todayDate := time.Now().Format("20060102")
	hourlyStats := make([]model.StatsHourly, 0, 24)
	for hour := 0; hour < 24; hour++ {
		if hourlyAll[hour].Date == todayDate {
			hourlyStats = append(hourlyStats, hourlyAll[hour])
		}
	}
	if len(hourlyStats) > 0 {
		if result := dbConn.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "hour"}},
			UpdateAll: true,
		}).Create(&hourlyStats); result.Error != nil {
			return result.Error
		}
	}

	for _, id := range channelIDs {
		ch, ok := statsChannelCache.Get(id)
		if !ok {
			continue
		}
		if result := dbConn.Save(&ch); result.Error != nil {
			return result.Error
		}
	}

	for _, id := range modelIDs {
		m, ok := statsModelCache.Get(id)
		if !ok {
			continue
		}
		if result := dbConn.Save(&m); result.Error != nil {
			return result.Error
		}
	}

	for _, id := range apiKeyIDs {
		ak, ok := statsAPIKeyCache.Get(id)
		if !ok {
			continue
		}
		if result := dbConn.Save(&ak); result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func statsSaveDBWithDailyOverride(ctx context.Context, dailyOverride model.StatsDaily) error {
	statsTotalCacheLock.RLock()
	totalSnap := statsTotalCache
	statsTotalCacheLock.RUnlock()
	if totalSnap.ID == 0 {
		totalSnap.ID = 1
	}

	statsHourlyCacheLock.RLock()
	hourlyAll := statsHourlyCache
	statsHourlyCacheLock.RUnlock()

	statsChannelCacheNeedUpdateLock.Lock()
	channelIDs := make([]int, 0, len(statsChannelCacheNeedUpdate))
	for id := range statsChannelCacheNeedUpdate {
		channelIDs = append(channelIDs, id)
	}
	statsChannelCacheNeedUpdate = make(map[int]struct{})
	statsChannelCacheNeedUpdateLock.Unlock()

	statsModelCacheNeedUpdateLock.Lock()
	modelIDs := make([]int, 0, len(statsModelCacheNeedUpdate))
	for id := range statsModelCacheNeedUpdate {
		modelIDs = append(modelIDs, id)
	}
	statsModelCacheNeedUpdate = make(map[int]struct{})
	statsModelCacheNeedUpdateLock.Unlock()

	statsAPIKeyCacheNeedUpdateLock.Lock()
	apiKeyIDs := make([]int, 0, len(statsAPIKeyCacheNeedUpdate))
	for id := range statsAPIKeyCacheNeedUpdate {
		apiKeyIDs = append(apiKeyIDs, id)
	}
	statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
	statsAPIKeyCacheNeedUpdateLock.Unlock()

	return persistStatsSnapshots(ctx, totalSnap, dailyOverride, hourlyAll, channelIDs, modelIDs, apiKeyIDs)
}

func StatsDailyUpdate(ctx context.Context, metrics model.StatsMetrics) error {
	today := time.Now().Format("20060102")

	statsDailyCacheLock.Lock()
	if statsDailyCache.Date == today {
		statsDailyCache.StatsMetrics.Add(metrics)
		statsDailyCacheLock.Unlock()
		return nil
	}

	prevDaily := statsDailyCache
	statsDailyCache = model.StatsDaily{Date: today}
	statsDailyCache.StatsMetrics.Add(metrics)
	statsDailyCacheLock.Unlock()

	return statsSaveDBWithDailyOverride(ctx, prevDaily)
}

func StatsTotalUpdate(metrics model.StatsMetrics) error {
	statsTotalCacheLock.Lock()
	defer statsTotalCacheLock.Unlock()
	if statsTotalCache.ID == 0 {
		statsTotalCache.ID = 1
	}
	statsTotalCache.StatsMetrics.Add(metrics)
	return nil
}

func StatsChannelUpdate(channelID int, metrics model.StatsMetrics) error {
	channelCache, ok := statsChannelCache.Get(channelID)
	if !ok {
		channelCache = model.StatsChannel{
			ChannelID: channelID,
		}
	}
	channelCache.StatsMetrics.Add(metrics)
	statsChannelCache.Set(channelID, channelCache)
	statsChannelCacheNeedUpdateLock.Lock()
	statsChannelCacheNeedUpdate[channelID] = struct{}{}
	statsChannelCacheNeedUpdateLock.Unlock()
	return nil
}

func StatsHourlyUpdate(metrics model.StatsMetrics) error {
	now := time.Now()
	nowHour := now.Hour()
	todayDate := time.Now().Format("20060102")

	statsHourlyCacheLock.Lock()
	defer statsHourlyCacheLock.Unlock()

	if statsHourlyCache[nowHour].Date != todayDate {
		statsHourlyCache[nowHour] = model.StatsHourly{
			Hour: nowHour,
			Date: todayDate,
		}
	}

	statsHourlyCache[nowHour].StatsMetrics.Add(metrics)
	return nil
}

func StatsModelUpdate(stats model.StatsModel) error {
	modelCache, ok := statsModelCache.Get(stats.ID)
	if !ok {
		modelCache = model.StatsModel{
			ID: stats.ID,
			Name: stats.Name,
			ChannelID: stats.ChannelID,
		}
	}
	if stats.Name != "" {
		modelCache.Name = stats.Name
	}
	if stats.ChannelID != 0 {
		modelCache.ChannelID = stats.ChannelID
	}
	modelCache.StatsMetrics.Add(stats.StatsMetrics)
	statsModelCache.Set(stats.ID, modelCache)
	statsModelCacheNeedUpdateLock.Lock()
	statsModelCacheNeedUpdate[stats.ID] = struct{}{}
	statsModelCacheNeedUpdateLock.Unlock()
	return nil
}

func StatsModelGet(id int) model.StatsModel {
	stats, ok := statsModelCache.Get(id)
	if !ok {
		return model.StatsModel{ID: id}
	}
	return stats
}

func StatsModelList() []model.StatsModel {
	models := make([]model.StatsModel, 0, statsModelCache.Len())
	for _, v := range statsModelCache.GetAll() {
		models = append(models, v)
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].ChannelID == models[j].ChannelID {
			return models[i].Name < models[j].Name
		}
		return models[i].ChannelID < models[j].ChannelID
	})
	return models
}

func StatsModelKey(channelID int, modelName string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d:%s", channelID, modelName)))
	return int(h.Sum32())
}

func ComputeHealthScore(metrics model.StatsMetrics, baseDelay int, enabledKeys, totalKeys int) float64 {
	totalRequests := metrics.RequestSuccess + metrics.RequestFailed
	if totalRequests == 0 {
		score := 70.0
		if baseDelay > 0 {
			score -= clamp(float64(baseDelay)/40.0, 0, 15)
		}
		if totalKeys > 0 {
			score += clamp(float64(enabledKeys)/float64(totalKeys)*10.0, 0, 10)
		}
		return clamp(score, 0, 100)
	}

	successRate := float64(metrics.RequestSuccess) / float64(totalRequests)
	avgWait := 0.0
	if totalRequests > 0 {
		avgWait = float64(metrics.WaitTime) / float64(totalRequests)
	}

	score := successRate * 70.0
	score += clamp(20.0-avgWait/150.0, 0, 20)
	if totalKeys > 0 {
		score += clamp(float64(enabledKeys)/float64(totalKeys)*10.0, 0, 10)
	}
	if baseDelay > 0 {
		score -= clamp(float64(baseDelay)/50.0, 0, 10)
	}

	return clamp(score, 0, 100)
}

func HealthGradeByScore(score float64) model.HealthGrade {
	switch {
	case score >= 90:
		return model.HealthGradeExcellent
	case score >= 75:
		return model.HealthGradeGood
	case score >= 55:
		return model.HealthGradeWarning
	default:
		return model.HealthGradeCritical
	}
}

func IsHealthyScore(score float64) bool {
	return score >= 75
}

func StatsChannelHealthList(includeModels bool, ctx context.Context) ([]model.StatsChannelHealth, error) {
	channels, err := ChannelList(ctx)
	if err != nil {
		return nil, err
	}

	modelStats := StatsModelList()
	modelsByChannel := make(map[int][]model.StatsModel)
	for _, item := range modelStats {
		if item.ChannelID == 0 || item.Name == "" {
			continue
		}
		modelsByChannel[item.ChannelID] = append(modelsByChannel[item.ChannelID], item)
	}

	result := make([]model.StatsChannelHealth, 0, len(channels))
	for _, ch := range channels {
		stats := StatsChannelGet(ch.ID)
		enabledKeys := 0
		for _, key := range ch.Keys {
			if key.Enabled && key.ChannelKey != "" {
				enabledKeys++
			}
		}

		modelNames := xstrings.SplitTrimCompact(",", ch.Model, ch.CustomModel)
		modelSet := make(map[string]struct{}, len(modelNames))
		for _, name := range modelNames {
			modelSet[name] = struct{}{}
		}
		for _, item := range modelsByChannel[ch.ID] {
			modelSet[item.Name] = struct{}{}
		}

		baseDelay := 0
		if url := ch.GetBaseUrl(); url != "" {
			for _, bu := range ch.BaseUrls {
				if bu.URL == url {
					baseDelay = bu.Delay
					break
				}
			}
		}

		channelHealth := model.StatsChannelHealth{
			ChannelID:      ch.ID,
			ChannelName:    ch.Name,
			Enabled:        ch.Enabled,
			Type:           int(ch.Type),
			RequestSuccess: stats.RequestSuccess,
			RequestFailed:  stats.RequestFailed,
			RequestCount:   stats.RequestSuccess + stats.RequestFailed,
			AvgWaitTime:    avgWaitTime(stats.StatsMetrics),
			BaseURLDelay:   baseDelay,
			EnabledKeys:    enabledKeys,
			TotalKeys:      len(ch.Keys),
			TotalModels:    len(modelSet),
		}
		if channelHealth.RequestCount > 0 {
			channelHealth.SuccessRate = float64(channelHealth.RequestSuccess) / float64(channelHealth.RequestCount)
		}
		channelHealth.Score = ComputeHealthScore(stats.StatsMetrics, baseDelay, enabledKeys, len(ch.Keys))
		channelHealth.Grade = HealthGradeByScore(channelHealth.Score)

		if includeModels {
			channelHealth.Models = make([]model.StatsModelHealth, 0, len(modelSet))
			for modelName := range modelSet {
				item := StatsModelGet(StatsModelKey(ch.ID, modelName))
				requestCount := item.RequestSuccess + item.RequestFailed
				mh := model.StatsModelHealth{
					ChannelID:      ch.ID,
					ModelName:      modelName,
					RequestSuccess: item.RequestSuccess,
					RequestFailed:  item.RequestFailed,
					RequestCount:   requestCount,
					AvgWaitTime:    avgWaitTime(item.StatsMetrics),
				}
				if requestCount > 0 {
					mh.SuccessRate = float64(item.RequestSuccess) / float64(requestCount)
				}
				mh.Score = ComputeHealthScore(item.StatsMetrics, baseDelay, enabledKeys, len(ch.Keys))
				mh.Grade = HealthGradeByScore(mh.Score)
				mh.Healthy = IsHealthyScore(mh.Score)
				if mh.Healthy {
					channelHealth.HealthyModels++
				}
				channelHealth.Models = append(channelHealth.Models, mh)
			}
			sort.Slice(channelHealth.Models, func(i, j int) bool {
				if channelHealth.Models[i].Score == channelHealth.Models[j].Score {
					return channelHealth.Models[i].ModelName < channelHealth.Models[j].ModelName
				}
				return channelHealth.Models[i].Score > channelHealth.Models[j].Score
			})
		}

		result = append(result, channelHealth)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].ChannelID < result[j].ChannelID
		}
		return result[i].Score > result[j].Score
	})
	return result, nil
}

func avgWaitTime(metrics model.StatsMetrics) int64 {
	totalRequests := metrics.RequestSuccess + metrics.RequestFailed
	if totalRequests == 0 {
		return 0
	}
	return metrics.WaitTime / totalRequests
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func StatsAPIKeyUpdate(apiKeyID int, metrics model.StatsMetrics) error {
	apiKeyCache, ok := statsAPIKeyCache.Get(apiKeyID)
	if !ok {
		apiKeyCache = model.StatsAPIKey{
			APIKeyID: apiKeyID,
		}
	}
	apiKeyCache.StatsMetrics.Add(metrics)
	statsAPIKeyCache.Set(apiKeyID, apiKeyCache)
	statsAPIKeyCacheNeedUpdateLock.Lock()
	statsAPIKeyCacheNeedUpdate[apiKeyID] = struct{}{}
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	return nil
}

func StatsChannelDel(id int) error {
	if _, ok := statsChannelCache.Get(id); !ok {
		return nil
	}
	statsChannelCache.Del(id)
	statsChannelCacheNeedUpdateLock.Lock()
	delete(statsChannelCacheNeedUpdate, id)
	statsChannelCacheNeedUpdateLock.Unlock()
	return db.GetDB().Delete(&model.StatsChannel{}, id).Error
}

func StatsAPIKeyDel(id int) error {
	if _, ok := statsAPIKeyCache.Get(id); !ok {
		return nil
	}
	statsAPIKeyCache.Del(id)
	statsAPIKeyCacheNeedUpdateLock.Lock()
	delete(statsAPIKeyCacheNeedUpdate, id)
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	return db.GetDB().Delete(&model.StatsAPIKey{}, id).Error
}

func StatsTotalGet() model.StatsTotal {
	statsTotalCacheLock.RLock()
	defer statsTotalCacheLock.RUnlock()
	return statsTotalCache
}

func StatsTodayGet() model.StatsDaily {
	statsDailyCacheLock.RLock()
	defer statsDailyCacheLock.RUnlock()
	return statsDailyCache
}

func StatsChannelGet(id int) model.StatsChannel {
	stats, ok := statsChannelCache.Get(id)
	if !ok {
		tmp := model.StatsChannel{
			ChannelID: id,
		}
		statsChannelCache.Set(id, tmp)
		statsChannelCacheNeedUpdateLock.Lock()
		statsChannelCacheNeedUpdate[id] = struct{}{}
		statsChannelCacheNeedUpdateLock.Unlock()
		return tmp
	}
	return stats
}

func StatsAPIKeyGet(id int) model.StatsAPIKey {
	stats, ok := statsAPIKeyCache.Get(id)
	if !ok {
		tmp := model.StatsAPIKey{
			APIKeyID: id,
		}
		statsAPIKeyCache.Set(id, tmp)
		statsAPIKeyCacheNeedUpdateLock.Lock()
		statsAPIKeyCacheNeedUpdate[id] = struct{}{}
		statsAPIKeyCacheNeedUpdateLock.Unlock()
		return tmp
	}
	return stats
}

func StatsAPIKeyList() []model.StatsAPIKey {
	apiKeys := make([]model.StatsAPIKey, 0, statsAPIKeyCache.Len())
	for _, v := range statsAPIKeyCache.GetAll() {
		apiKeys = append(apiKeys, v)
	}
	return apiKeys
}

func StatsHourlyGet() []model.StatsHourly {
	now := time.Now()
	currentHour := now.Hour()
	todayDate := time.Now().Format("20060102")

	statsHourlyCacheLock.RLock()
	defer statsHourlyCacheLock.RUnlock()

	result := make([]model.StatsHourly, 0, currentHour+1)

	for hour := 0; hour <= currentHour; hour++ {
		if statsHourlyCache[hour].Date == todayDate {
			result = append(result, statsHourlyCache[hour])
		} else {
			result = append(result, model.StatsHourly{
				Hour: hour,
				Date: todayDate,
			})
		}
	}

	return result
}

func StatsGetDaily(ctx context.Context) ([]model.StatsDaily, error) {
	var statsDaily []model.StatsDaily
	result := db.GetDB().WithContext(ctx).Find(&statsDaily)
	if result.Error != nil {
		return nil, result.Error
	}
	return statsDaily, nil
}

func statsRefreshCache(ctx context.Context) error {
	dbConn := db.GetDB().WithContext(ctx)
	today := time.Now().Format("20060102")

	var loadedDaily model.StatsDaily
	result := dbConn.Last(&loadedDaily)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get daily stats: %v", result.Error)
	}
	if result.RowsAffected == 0 || loadedDaily.Date != today {
		loadedDaily = model.StatsDaily{Date: today}
	}

	var loadedTotal model.StatsTotal
	result = dbConn.First(&loadedTotal)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get total stats: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		loadedTotal = model.StatsTotal{ID: 1}
	} else if loadedTotal.ID == 0 {
		loadedTotal.ID = 1
	}

	var loadedChannels []model.StatsChannel
	result = dbConn.Find(&loadedChannels)
	if result.Error != nil {
		return fmt.Errorf("failed to get channels: %v", result.Error)
	}

	var loadedModels []model.StatsModel
	result = dbConn.Find(&loadedModels)
	if result.Error != nil {
		return fmt.Errorf("failed to get model stats: %v", result.Error)
	}

	var loadedHourly []model.StatsHourly
	result = dbConn.Find(&loadedHourly)
	if result.Error != nil {
		return fmt.Errorf("failed to get hourly stats: %v", result.Error)
	}

	statsDailyCacheLock.Lock()
	statsDailyCache = loadedDaily
	statsDailyCacheLock.Unlock()

	statsTotalCacheLock.Lock()
	statsTotalCache = loadedTotal
	statsTotalCacheLock.Unlock()

	statsChannelCache.Clear()
	statsChannelCacheNeedUpdateLock.Lock()
	statsChannelCacheNeedUpdate = make(map[int]struct{})
	statsChannelCacheNeedUpdateLock.Unlock()
	for _, v := range loadedChannels {
		statsChannelCache.Set(v.ChannelID, v)
	}

	statsModelCache.Clear()
	statsModelCacheNeedUpdateLock.Lock()
	statsModelCacheNeedUpdate = make(map[int]struct{})
	statsModelCacheNeedUpdateLock.Unlock()
	for _, v := range loadedModels {
		statsModelCache.Set(v.ID, v)
	}

	var loadedAPIKeys []model.StatsAPIKey
	result = dbConn.Find(&loadedAPIKeys)
	if result.Error != nil {
		return fmt.Errorf("failed to get api key stats: %v", result.Error)
	}

	statsAPIKeyCache.Clear()
	statsAPIKeyCacheNeedUpdateLock.Lock()
	statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	for _, v := range loadedAPIKeys {
		statsAPIKeyCache.Set(v.APIKeyID, v)
	}

	statsHourlyCacheLock.Lock()
	statsHourlyCache = [24]model.StatsHourly{}
	for _, v := range loadedHourly {
		if v.Hour >= 0 && v.Hour < 24 {
			statsHourlyCache[v.Hour] = v
		}
	}
	statsHourlyCacheLock.Unlock()

	return nil
}
