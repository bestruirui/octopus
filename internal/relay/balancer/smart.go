package balancer

import (
	"fmt"
	"sync"
	"time"
)

const (
	smartStatsBuckets = 24 * 60

	smartWeightManual = 0.30
	smartWeight1h     = 0.40
	smartWeight24h    = 0.30

	smartRatePriorSuccess = 1.0
	smartRatePriorFailure = 1.0
)

type smartMinuteBucket struct {
	minute  int64
	success uint32
	failure uint32
}

type smartRollingStats struct {
	mu      sync.Mutex
	buckets [smartStatsBuckets]smartMinuteBucket
}

var (
	smartChannelStats sync.Map // key: channelID:modelName -> *smartRollingStats
	smartNowFunc      = time.Now
)

func smartStatsKey(channelID int, modelName string) string {
	return fmt.Sprintf("%d:%s", channelID, modelName)
}

func getOrCreateSmartStats(channelID int, modelName string) *smartRollingStats {
	key := smartStatsKey(channelID, modelName)
	if v, ok := smartChannelStats.Load(key); ok {
		return v.(*smartRollingStats)
	}
	entry := &smartRollingStats{}
	actual, _ := smartChannelStats.LoadOrStore(key, entry)
	return actual.(*smartRollingStats)
}

func recordSmartOutcome(channelID int, modelName string, success bool) {
	if channelID <= 0 || modelName == "" {
		return
	}
	stats := getOrCreateSmartStats(channelID, modelName)
	stats.add(smartNowFunc(), success)
}

// RecordSmartOutcome records per-channel per-model request outcomes for smart weighted selection.
func RecordSmartOutcome(channelID int, modelName string, success bool) {
	recordSmartOutcome(channelID, modelName, success)
}

func getSmartSuccessRates(channelID int, modelName string) (float64, float64) {
	stats := getOrCreateSmartStats(channelID, modelName)
	now := smartNowFunc()
	return stats.successRate(now, 60), stats.successRate(now, 24*60)
}

func (s *smartRollingStats) add(now time.Time, success bool) {
	minute := now.Unix() / 60
	idx := int(minute % smartStatsBuckets)
	if idx < 0 {
		idx += smartStatsBuckets
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	b := &s.buckets[idx]
	if b.minute != minute {
		b.minute = minute
		b.success = 0
		b.failure = 0
	}
	if success {
		b.success++
		return
	}
	b.failure++
}

func (s *smartRollingStats) successRate(now time.Time, windowMinutes int) float64 {
	currentMinute := now.Unix() / 60
	var successCount int64
	var totalCount int64

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := 0; i < windowMinutes; i++ {
		minute := currentMinute - int64(i)
		idx := int(minute % smartStatsBuckets)
		if idx < 0 {
			idx += smartStatsBuckets
		}
		b := s.buckets[idx]
		if b.minute != minute {
			continue
		}
		successCount += int64(b.success)
		totalCount += int64(b.success + b.failure)
	}

	return (float64(successCount) + smartRatePriorSuccess) / (float64(totalCount) + smartRatePriorSuccess + smartRatePriorFailure)
}

func resetSmartStatsForTest() {
	smartChannelStats = sync.Map{}
}
