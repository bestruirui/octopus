package balancer

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/utils/log"
	"gorm.io/gorm"
)

func LoadRuntimeState(ctx context.Context) error {
	dbConn := db.GetDB()
	if dbConn == nil {
		return nil
	}

	var autoStates []model.AutoStrategyState
	if err := dbConn.WithContext(ctx).Find(&autoStates).Error; err != nil {
		return err
	}

	var circuitStates []model.CircuitBreakerState
	if err := dbConn.WithContext(ctx).Find(&circuitStates).Error; err != nil {
		return err
	}

	clearAutoStats()
	clearCircuitBreaker()

	now := time.Now()
	timeWindow := getTimeWindow()
	sampleThreshold := getSampleThreshold()

	autoLoaded := 0
	for _, state := range autoStates {
		if state.ChannelID > 0 && !channelExists(state.ChannelID) {
			continue
		}

		stats := restoreChannelStats(state.Records, now, timeWindow, sampleThreshold)
		if stats == nil {
			continue
		}

		key := strings.TrimSpace(state.Key)
		if key == "" {
			key = statsKey(state.ChannelID, state.ModelName)
		}
		globalAutoStats.Store(key, stats)
		autoLoaded++
	}

	circuitLoaded := 0
	for _, state := range circuitStates {
		if state.ChannelID > 0 && state.ChannelKeyID > 0 && !channelKeyExists(state.ChannelID, state.ChannelKeyID) {
			continue
		}

		entry := restoreCircuitEntry(state)
		if entry == nil {
			continue
		}

		key := strings.TrimSpace(state.Key)
		if key == "" {
			key = circuitKey(state.ChannelID, state.ChannelKeyID, state.ModelName)
		}
		globalBreaker.Store(key, entry)
		circuitLoaded++
	}

	log.Infof("balancer runtime state loaded: auto=%d circuit=%d", autoLoaded, circuitLoaded)
	return nil
}

func SaveRuntimeState(ctx context.Context) error {
	dbConn := db.GetDB()
	if dbConn == nil {
		return nil
	}

	now := time.Now()
	autoStates := snapshotAutoStrategyStates(now)
	circuitStates := snapshotCircuitBreakerStates(now)

	return dbConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.AutoStrategyState{}).Error; err != nil {
			return err
		}
		if len(autoStates) > 0 {
			if err := tx.Create(&autoStates).Error; err != nil {
				return err
			}
		}

		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.CircuitBreakerState{}).Error; err != nil {
			return err
		}
		if len(circuitStates) > 0 {
			if err := tx.Create(&circuitStates).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func RuntimeStateSaveDBTask() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Debugf("balancer runtime state save db task started")
	startTime := time.Now()
	defer func() {
		log.Debugf("balancer runtime state save db task finished, save time: %s", time.Since(startTime))
	}()

	if err := SaveRuntimeState(ctx); err != nil {
		log.Errorf("balancer runtime state save db error: %v", err)
	}
}

func clearAutoStats() {
	globalAutoStats.Range(func(key, _ any) bool {
		globalAutoStats.Delete(key)
		return true
	})
}

func clearCircuitBreaker() {
	globalBreaker.Range(func(key, _ any) bool {
		globalBreaker.Delete(key)
		return true
	})
}

func channelExists(channelID int) bool {
	_, err := op.ChannelGet(channelID, context.Background())
	return err == nil
}

func channelKeyExists(channelID, keyID int) bool {
	channel, err := op.ChannelGet(channelID, context.Background())
	if err != nil {
		return false
	}
	for _, key := range channel.Keys {
		if key.ID == keyID {
			return true
		}
	}
	return false
}

func parseAutoStatsKey(key string) (int, string, bool) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return 0, "", false
	}

	channelID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", false
	}

	return channelID, parts[1], true
}

func parseCircuitStateKey(key string) (int, int, string, bool) {
	parts := strings.SplitN(key, ":", 3)
	if len(parts) != 3 {
		return 0, 0, "", false
	}

	channelID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, "", false
	}
	keyID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, "", false
	}

	return channelID, keyID, parts[2], true
}

func newChannelStatsWithCapacity(capacity int) *ChannelStats {
	if capacity < 1 {
		capacity = 1
	}
	return &ChannelStats{
		window:             make([]RequestRecord, capacity),
		cacheValidDuration: 5 * time.Second,
	}
}

func (cs *ChannelStats) snapshotRecords(now time.Time, timeWindow time.Duration, sampleThreshold int) []model.AutoStrategyRecord {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if cs.count == 0 || len(cs.window) == 0 {
		return nil
	}

	cutoff := now.Add(-timeWindow)
	start := 0
	if cs.count == len(cs.window) {
		start = cs.head
	}

	records := make([]model.AutoStrategyRecord, 0, cs.count)
	for i := 0; i < cs.count; i++ {
		idx := (start + i) % len(cs.window)
		record := cs.window[idx]
		if record.Timestamp.IsZero() || record.Timestamp.Before(cutoff) {
			continue
		}
		records = append(records, model.AutoStrategyRecord{
			Timestamp: record.Timestamp.UnixMilli(),
			Success:   record.Success,
		})
	}

	if sampleThreshold > 0 && len(records) > sampleThreshold {
		records = append([]model.AutoStrategyRecord(nil), records[len(records)-sampleThreshold:]...)
	}

	return records
}

func restoreChannelStats(records []model.AutoStrategyRecord, now time.Time, timeWindow time.Duration, sampleThreshold int) *ChannelStats {
	if sampleThreshold < 1 {
		sampleThreshold = 1
	}

	cutoff := now.Add(-timeWindow).UnixMilli()
	filtered := make([]model.AutoStrategyRecord, 0, len(records))
	for _, record := range records {
		if record.Timestamp <= 0 || record.Timestamp < cutoff {
			continue
		}
		filtered = append(filtered, record)
	}

	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) > sampleThreshold {
		filtered = filtered[len(filtered)-sampleThreshold:]
	}

	stats := newChannelStatsWithCapacity(sampleThreshold)
	for _, record := range filtered {
		stats.window[stats.head] = RequestRecord{
			Timestamp: time.UnixMilli(record.Timestamp),
			Success:   record.Success,
		}
		stats.head = (stats.head + 1) % len(stats.window)
		if stats.count < len(stats.window) {
			stats.count++
		}
	}

	return stats
}

func snapshotAutoStrategyStates(now time.Time) []model.AutoStrategyState {
	timeWindow := getTimeWindow()
	sampleThreshold := getSampleThreshold()
	updatedAt := now.UnixMilli()

	states := make([]model.AutoStrategyState, 0)
	globalAutoStats.Range(func(rawKey, value any) bool {
		key, ok := rawKey.(string)
		if !ok {
			return true
		}
		stats, ok := value.(*ChannelStats)
		if !ok || stats == nil {
			return true
		}

		channelID, modelName, ok := parseAutoStatsKey(key)
		if !ok || !channelExists(channelID) {
			return true
		}

		records := stats.snapshotRecords(now, timeWindow, sampleThreshold)
		if len(records) == 0 {
			return true
		}

		states = append(states, model.AutoStrategyState{
			Key:       key,
			ChannelID: channelID,
			ModelName: modelName,
			Records:   records,
			UpdatedAt: updatedAt,
		})
		return true
	})

	sort.Slice(states, func(i, j int) bool {
		return states[i].Key < states[j].Key
	})
	return states
}

func snapshotCircuitEntry(
	key string,
	channelID int,
	channelKeyID int,
	modelName string,
	entry *circuitEntry,
	updatedAt int64,
) (model.CircuitBreakerState, bool) {
	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.State == StateClosed && entry.ConsecutiveFailures == 0 && entry.TripCount == 0 && entry.LastFailureTime.IsZero() {
		return model.CircuitBreakerState{}, false
	}

	state := entry.State
	if state == StateHalfOpen {
		state = StateOpen
	}

	lastFailureTime := int64(0)
	if !entry.LastFailureTime.IsZero() {
		lastFailureTime = entry.LastFailureTime.UnixMilli()
	}

	return model.CircuitBreakerState{
		Key:                 key,
		ChannelID:           channelID,
		ChannelKeyID:        channelKeyID,
		ModelName:           modelName,
		State:               int(state),
		ConsecutiveFailures: entry.ConsecutiveFailures,
		LastFailureTime:     lastFailureTime,
		TripCount:           entry.TripCount,
		UpdatedAt:           updatedAt,
	}, true
}

func restoreCircuitEntry(state model.CircuitBreakerState) *circuitEntry {
	circuitState := CircuitState(state.State)
	switch circuitState {
	case StateHalfOpen:
		circuitState = StateOpen
	case StateClosed, StateOpen:
	default:
		circuitState = StateClosed
	}

	entry := &circuitEntry{
		State:               circuitState,
		ConsecutiveFailures: state.ConsecutiveFailures,
		TripCount:           state.TripCount,
	}
	if state.LastFailureTime > 0 {
		entry.LastFailureTime = time.UnixMilli(state.LastFailureTime)
	}

	if entry.State == StateClosed && entry.ConsecutiveFailures == 0 && entry.TripCount == 0 && entry.LastFailureTime.IsZero() {
		return nil
	}

	return entry
}

func snapshotCircuitBreakerStates(now time.Time) []model.CircuitBreakerState {
	updatedAt := now.UnixMilli()
	states := make([]model.CircuitBreakerState, 0)

	globalBreaker.Range(func(rawKey, value any) bool {
		key, ok := rawKey.(string)
		if !ok {
			return true
		}
		entry, ok := value.(*circuitEntry)
		if !ok || entry == nil {
			return true
		}

		channelID, channelKeyID, modelName, ok := parseCircuitStateKey(key)
		if !ok || !channelKeyExists(channelID, channelKeyID) {
			return true
		}

		state, ok := snapshotCircuitEntry(key, channelID, channelKeyID, modelName, entry, updatedAt)
		if !ok {
			return true
		}
		states = append(states, state)
		return true
	})

	sort.Slice(states, func(i, j int) bool {
		return states[i].Key < states[j].Key
	})
	return states
}
