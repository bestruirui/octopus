package balancer

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
)

// RequestRecord represents a single request outcome within the sliding window.
type RequestRecord struct {
	Timestamp time.Time
	Success   bool
}

// ChannelStats tracks request statistics for a single channel+model combination.
// It uses a ring buffer for the sliding window implementation.
type ChannelStats struct {
	mu sync.RWMutex

	// Ring buffer for sliding window
	window []RequestRecord
	head   int // Next write position
	count  int // Number of records written

	// Cached values to avoid frequent recomputation
	cachedSuccessRate  float64
	cachedTotalSamples int
	lastCacheUpdate    time.Time
	cacheValidDuration time.Duration
}

// Global storage for channel statistics.
// Key format: "channelID:modelName"
var globalAutoStats sync.Map // string -> *ChannelStats

// statsKey generates the key for channel stats storage.
func statsKey(channelID int, modelName string) string {
	return fmt.Sprintf("%d:%s", channelID, normalizeAutoStatsModelName(modelName))
}

func normalizeAutoStatsModelName(modelName string) string {
	return strings.ToLower(strings.TrimSpace(modelName))
}

// getOrCreateStats retrieves or creates a ChannelStats entry.
func getOrCreateStats(channelID int, modelName string) *ChannelStats {
	key := statsKey(channelID, modelName)
	if v, ok := globalAutoStats.Load(key); ok {
		return v.(*ChannelStats)
	}
	threshold := getSampleThreshold()
	entry := &ChannelStats{
		window:             make([]RequestRecord, threshold),
		cacheValidDuration: 5 * time.Second,
	}
	actual, _ := globalAutoStats.LoadOrStore(key, entry)
	return actual.(*ChannelStats)
}

// getMinSamples returns the minimum samples threshold before using success rate.
func getMinSamples() int {
	v, err := op.SettingGetInt(model.SettingKeyAutoStrategyMinSamples)
	if err != nil || v <= 0 {
		return 10
	}
	return v
}

// getTimeWindow returns the time window duration in seconds.
func getTimeWindow() time.Duration {
	v, err := op.SettingGetInt(model.SettingKeyAutoStrategyTimeWindow)
	if err != nil || v <= 0 {
		return 300 * time.Second
	}
	return time.Duration(v) * time.Second
}

// getSampleThreshold returns the sliding window size.
func getSampleThreshold() int {
	v, err := op.SettingGetInt(model.SettingKeyAutoStrategySampleThreshold)
	if err != nil || v <= 0 {
		return 100
	}
	return v
}

// Record records a request outcome to the sliding window.
func (cs *ChannelStats) Record(success bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Write to ring buffer
	cs.window[cs.head] = RequestRecord{
		Timestamp: time.Now(),
		Success:   success,
	}
	cs.head = (cs.head + 1) % len(cs.window)
	if cs.count < len(cs.window) {
		cs.count++
	}

	// Invalidate cache
	cs.lastCacheUpdate = time.Time{}
}

// GetStats returns the success rate and total samples within the time window.
func (cs *ChannelStats) GetStats(windowDuration time.Duration) (successRate float64, totalSamples int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	now := time.Now()

	// Return cached value if still valid
	if now.Sub(cs.lastCacheUpdate) < cs.cacheValidDuration && cs.cachedTotalSamples > 0 {
		return cs.cachedSuccessRate, cs.cachedTotalSamples
	}

	// Recompute stats
	cutoff := now.Add(-windowDuration)
	successCount := 0
	validCount := 0

	for i := 0; i < cs.count; i++ {
		record := cs.window[i]
		if record.Timestamp.After(cutoff) || record.Timestamp.Equal(cutoff) {
			validCount++
			if record.Success {
				successCount++
			}
		}
	}

	// Update cache
	if validCount > 0 {
		cs.cachedSuccessRate = float64(successCount) / float64(validCount)
	} else {
		cs.cachedSuccessRate = 0
	}
	cs.cachedTotalSamples = validCount
	cs.lastCacheUpdate = now

	return cs.cachedSuccessRate, cs.cachedTotalSamples
}

// RecordAutoSuccess records a successful request for the Auto strategy.
func RecordAutoSuccess(channelID int, modelName string) {
	stats := getOrCreateStats(channelID, modelName)
	stats.Record(true)
}

// RecordAutoFailure records a failed request for the Auto strategy.
func RecordAutoFailure(channelID int, modelName string) {
	stats := getOrCreateStats(channelID, modelName)
	stats.Record(false)
}

// GetAutoStats returns the success rate and total samples for a channel+model.
// This is primarily for debugging/testing purposes.
func GetAutoStats(channelID int, modelName string) (successRate float64, totalSamples int) {
	key := statsKey(channelID, modelName)
	v, ok := globalAutoStats.Load(key)
	if !ok {
		return 0, 0
	}
	stats := v.(*ChannelStats)
	return stats.GetStats(getTimeWindow())
}
