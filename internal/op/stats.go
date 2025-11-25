package op

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/bestruirui/octopus/internal/utils/timeo"
)

var statsDailyCache model.StatsDaily
var statsDailyCacheLock sync.RWMutex

func StatsSaveDBTask() {
	interval, err := SettingGetInt(model.SettingKeyStatsSaveInterval)
	if err != nil {
		return
	}
	for {
		time.Sleep(time.Duration(interval) * time.Minute)
		if err := StatsSaveDB(); err != nil {
			log.Errorf("stats save db error: %v", err)
		}
	}
}

func StatsSaveDB() error {
	return db.GetDB().WithContext(context.Background()).Save(&statsDailyCache).Error
}

func StatsDailyUpdate(ctx context.Context, stats model.StatsDaily) error {
	statsDailyCacheLock.Lock()
	defer statsDailyCacheLock.Unlock()
	if timeo.ToMidnight(statsDailyCache.Date) != timeo.GetMidnight() {
		if err := StatsSaveDB(); err != nil {
			return err
		}
		statsDailyCache = model.StatsDaily{
			Date: timeo.GetMidnight(),
		}
	}
	statsDailyCache.InputToken += stats.InputToken
	statsDailyCache.OutputToken += stats.OutputToken
	statsDailyCache.RequestCount += stats.RequestCount
	statsDailyCache.Money += stats.Money
	statsDailyCache.WaitTime += stats.WaitTime
	return nil
}

func StatsGetToday(ctx context.Context) model.StatsDaily {
	statsDailyCacheLock.RLock()
	defer statsDailyCacheLock.RUnlock()
	return statsDailyCache
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
	statsDailyCacheLock.Lock()
	defer statsDailyCacheLock.Unlock()
	result := db.GetDB().WithContext(ctx).Last(&statsDailyCache)
	if result.RowsAffected == 0 {
		statsDailyCache = model.StatsDaily{
			Date: timeo.GetMidnight(),
		}
		return nil
	}

	if result.Error != nil {
		return result.Error
	}
	fmt.Println(statsDailyCache)
	fmt.Printf("今日 %v\n", timeo.GetMidnight())
	fmt.Printf("数据库 %v \n", timeo.ToMidnight(statsDailyCache.Date))

	if timeo.ToMidnight(statsDailyCache.Date) != timeo.GetMidnight() {
		statsDailyCache = model.StatsDaily{
			Date: timeo.GetMidnight(),
		}
	}
	fmt.Println(statsDailyCache)

	return nil
}
