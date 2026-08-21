package op

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
	"github.com/charmbracelet/log"
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

	if err := persistStatsSnapshots(ctx, totalSnap, dailySnap, hourlyAll, channelIDs, modelIDs, apiKeyIDs); err != nil {
		restoreStatsDirty(channelIDs, modelIDs, apiKeyIDs)
		return err
	}
	return nil
}

// restoreStatsDirty 在统计持久化失败后恢复本批待写标记。
func restoreStatsDirty(channelIDs, modelIDs, apiKeyIDs []int) {
	statsChannelCacheNeedUpdateLock.Lock()
	for _, id := range channelIDs {
		statsChannelCacheNeedUpdate[id] = struct{}{}
	}
	statsChannelCacheNeedUpdateLock.Unlock()

	statsModelCacheNeedUpdateLock.Lock()
	for _, id := range modelIDs {
		statsModelCacheNeedUpdate[id] = struct{}{}
	}
	statsModelCacheNeedUpdateLock.Unlock()

	statsAPIKeyCacheNeedUpdateLock.Lock()
	for _, id := range apiKeyIDs {
		statsAPIKeyCacheNeedUpdate[id] = struct{}{}
	}
	statsAPIKeyCacheNeedUpdateLock.Unlock()
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

	if err := persistStatsSnapshots(ctx, totalSnap, dailyOverride, hourlyAll, channelIDs, modelIDs, apiKeyIDs); err != nil {
		restoreStatsDirty(channelIDs, modelIDs, apiKeyIDs)
		return err
	}
	return nil
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

// StatsChannelUpdate 累加仍然存在的渠道统计并标记为待持久化。
func StatsChannelUpdate(channelID int, metrics model.StatsMetrics) error {
	statsChannelCacheNeedUpdateLock.Lock()
	defer statsChannelCacheNeedUpdateLock.Unlock()
	if _, ok := channelCache.Get(channelID); !ok {
		return nil
	}
	channelCache, ok := statsChannelCache.Get(channelID)
	if !ok {
		channelCache = model.StatsChannel{
			ChannelID: channelID,
		}
	}
	channelCache.StatsMetrics.Add(metrics)
	statsChannelCache.Set(channelID, channelCache)
	statsChannelCacheNeedUpdate[channelID] = struct{}{}
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

// StatsModelUpdate 累加仍然存在的渠道模型统计并标记为待持久化。
func StatsModelUpdate(stats model.StatsModel) error {
	statsModelCacheNeedUpdateLock.Lock()
	defer statsModelCacheNeedUpdateLock.Unlock()
	if _, ok := channelCache.Get(stats.ChannelID); !ok {
		return nil
	}
	modelCache, ok := statsModelCache.Get(stats.ID)
	if !ok {
		modelCache = model.StatsModel{
			ID:        stats.ID,
			Name:      stats.Name,
			ChannelID: stats.ChannelID,
		}
	}
	modelCache.StatsMetrics.Add(stats.StatsMetrics)
	statsModelCache.Set(stats.ID, modelCache)
	statsModelCacheNeedUpdate[stats.ID] = struct{}{}
	return nil
}

func StatsAPIKeyUpdate(apiKeyID int, metrics model.StatsMetrics) error {
	statsAPIKeyCacheNeedUpdateLock.Lock()
	defer statsAPIKeyCacheNeedUpdateLock.Unlock()
	apiKeyCache, ok := statsAPIKeyCache.Get(apiKeyID)
	if !ok {
		apiKeyCache = model.StatsAPIKey{
			APIKeyID: apiKeyID,
		}
	}
	apiKeyCache.StatsMetrics.Add(metrics)
	statsAPIKeyCache.Set(apiKeyID, apiKeyCache)
	statsAPIKeyCacheNeedUpdate[apiKeyID] = struct{}{}
	return nil
}

func StatsChannelDel(id int) error {
	statsChannelCacheNeedUpdateLock.Lock()
	if _, ok := statsChannelCache.Get(id); !ok {
		statsChannelCacheNeedUpdateLock.Unlock()
		return nil
	}
	statsChannelCache.Del(id)
	delete(statsChannelCacheNeedUpdate, id)
	statsChannelCacheNeedUpdateLock.Unlock()
	return db.GetDB().Delete(&model.StatsChannel{}, id).Error
}

func StatsAPIKeyDel(id int) error {
	statsAPIKeyCacheNeedUpdateLock.Lock()
	if _, ok := statsAPIKeyCache.Get(id); !ok {
		statsAPIKeyCacheNeedUpdateLock.Unlock()
		return nil
	}
	statsAPIKeyCache.Del(id)
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
	if stats, ok := statsChannelCache.Get(id); ok {
		return stats
	}
	statsChannelCacheNeedUpdateLock.Lock()
	defer statsChannelCacheNeedUpdateLock.Unlock()
	stats, ok := statsChannelCache.Get(id)
	if !ok {
		tmp := model.StatsChannel{
			ChannelID: id,
		}
		statsChannelCache.Set(id, tmp)
		statsChannelCacheNeedUpdate[id] = struct{}{}
		return tmp
	}
	return stats
}

func StatsAPIKeyGet(id int) model.StatsAPIKey {
	if stats, ok := statsAPIKeyCache.Get(id); ok {
		return stats
	}
	statsAPIKeyCacheNeedUpdateLock.Lock()
	defer statsAPIKeyCacheNeedUpdateLock.Unlock()
	stats, ok := statsAPIKeyCache.Get(id)
	if !ok {
		tmp := model.StatsAPIKey{
			APIKeyID: id,
		}
		statsAPIKeyCache.Set(id, tmp)
		statsAPIKeyCacheNeedUpdate[id] = struct{}{}
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

// StatsGroupList 把模型统计按分组聚合返回。
func StatsGroupList() []model.StatsGroup {
	itemGroup := make(map[int]int)
	groupNames := make(map[int]string)
	for _, group := range groupCache.GetAll() {
		groupNames[group.ID] = group.Name
		for _, item := range group.Items {
			itemGroup[item.ID] = group.ID
		}
	}

	groupStats := make(map[int]*model.StatsGroup)
	for _, stats := range statsModelCache.GetAll() {
		groupID, ok := itemGroup[stats.ID]
		if !ok {
			continue
		}
		sg, ok := groupStats[groupID]
		if !ok {
			sg = &model.StatsGroup{GroupID: groupID, GroupName: groupNames[groupID]}
			groupStats[groupID] = sg
		}
		sg.StatsMetrics.Add(stats.StatsMetrics)
	}

	result := make([]model.StatsGroup, 0, len(groupStats))
	for _, sg := range groupStats {
		result = append(result, *sg)
	}
	return result
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
