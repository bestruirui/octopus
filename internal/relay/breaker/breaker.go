package breaker

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bestruirui/octopus/internal/model"
)

const shardCount = 64

type Decision string

const (
	DecisionDisabled    Decision = "disabled"
	DecisionClosedAllow Decision = "closed_allow"
	DecisionSkipOpen    Decision = "skip_open"
	DecisionProbeAllow  Decision = "probe_allowed"
	DecisionProbeDenied Decision = "probe_denied"
	DecisionRecordFail  Decision = "record_failure"
	DecisionProbeFailed Decision = "probe_failed"
	DecisionAllOpen     Decision = "all_open"
)

type AttemptAcquire struct {
	Key           string
	Allowed       bool
	Decision      Decision
	StateBefore   model.CircuitBreakerState
	StateAfter    model.CircuitBreakerState
	TripCount     int
	OpenUntil     time.Time
	ProbeGranted  bool
	ProbeInFlight bool
}

type RecordResult struct {
	Decision      Decision
	StateAfter    model.CircuitBreakerState
	TripCount     int
	OpenUntil     time.Time
	ProbeInFlight bool
}

type Snapshot struct {
	Key                 string
	State               model.CircuitBreakerState
	ConsecutiveFailures int
	TripCount           int
	OpenUntil           time.Time
	LastFailureAt       time.Time
	LastFailureReason   string
	LastTripAt          time.Time
	ProbeInFlight       bool
}

type breakerState struct {
	mu sync.Mutex

	state               model.CircuitBreakerState
	consecutiveFailures int
	tripCount           int
	openUntil           time.Time
	lastFailureAt       time.Time
	lastFailureReason   string
	lastTripAt          time.Time
	probeInFlight       uint32
}

type shard struct {
	mu sync.RWMutex
	m  map[string]*breakerState
}

type Manager struct {
	shards [shardCount]shard
}

var globalManager = NewManager()

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewManager() *Manager {
	m := &Manager{}
	for i := 0; i < shardCount; i++ {
		m.shards[i].m = make(map[string]*breakerState)
	}
	return m
}

func GetManager() *Manager {
	return globalManager
}

func BuildKey(channelID int, modelName string) string {
	return fmt.Sprintf("%d:%s", channelID, strings.TrimSpace(modelName))
}

func (m *Manager) FilterAvailable(items []model.GroupItem, now time.Time, cfg model.CircuitBreakerConfig) ([]model.GroupItem, time.Time) {
	filtered := make([]model.GroupItem, 0, len(items))
	if !cfg.Enabled {
		filtered = append(filtered, items...)
		return filtered, time.Time{}
	}

	var earliest time.Time
	for _, item := range items {
		snap := m.Snapshot(BuildKey(item.ChannelID, item.ModelName))
		switch snap.State {
		case model.CircuitBreakerStateOpen:
			if now.Before(snap.OpenUntil) {
				if earliest.IsZero() || snap.OpenUntil.Before(earliest) {
					earliest = snap.OpenUntil
				}
				continue
			}
		case model.CircuitBreakerStateHalfOpen:
			if snap.ProbeInFlight {
				retryAt := snap.OpenUntil
				if retryAt.IsZero() || retryAt.Before(now) {
					retryAt = now.Add(1 * time.Second)
				}
				if earliest.IsZero() || retryAt.Before(earliest) {
					earliest = retryAt
				}
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered, earliest
}

func (m *Manager) Acquire(key string, now time.Time, cfg model.CircuitBreakerConfig) AttemptAcquire {
	if !cfg.Enabled {
		return AttemptAcquire{
			Key:         key,
			Allowed:     true,
			Decision:    DecisionDisabled,
			StateBefore: model.CircuitBreakerStateClosed,
			StateAfter:  model.CircuitBreakerStateClosed,
		}
	}
	s := m.getOrCreateState(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	result := AttemptAcquire{
		Key:         key,
		Allowed:     false,
		Decision:    DecisionSkipOpen,
		StateBefore: s.state,
		StateAfter:  s.state,
		TripCount:   s.tripCount,
		OpenUntil:   s.openUntil,
	}

	switch s.state {
	case model.CircuitBreakerStateClosed:
		result.Allowed = true
		result.Decision = DecisionClosedAllow
		return result
	case model.CircuitBreakerStateOpen:
		if now.Before(s.openUntil) {
			return result
		}
		if atomic.CompareAndSwapUint32(&s.probeInFlight, 0, 1) {
			s.state = model.CircuitBreakerStateHalfOpen
			result.Allowed = true
			result.Decision = DecisionProbeAllow
			result.StateAfter = s.state
			result.ProbeGranted = true
			result.ProbeInFlight = true
			return result
		}
		result.Decision = DecisionProbeDenied
		result.ProbeInFlight = true
		return result
	case model.CircuitBreakerStateHalfOpen:
		if atomic.CompareAndSwapUint32(&s.probeInFlight, 0, 1) {
			result.Allowed = true
			result.Decision = DecisionProbeAllow
			result.ProbeGranted = true
			result.ProbeInFlight = true
			return result
		}
		result.Decision = DecisionProbeDenied
		result.ProbeInFlight = true
		return result
	default:
		s.state = model.CircuitBreakerStateClosed
		result.Allowed = true
		result.Decision = DecisionClosedAllow
		result.StateAfter = s.state
		return result
	}
}

func (m *Manager) RecordSuccess(key string, acquire AttemptAcquire, cfg model.CircuitBreakerConfig, now time.Time) RecordResult {
	s := m.getOrCreateState(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	if !cfg.Enabled {
		atomic.StoreUint32(&s.probeInFlight, 0)
		return m.recordResultFromState(s, DecisionDisabled)
	}

	if acquire.ProbeGranted || s.state == model.CircuitBreakerStateHalfOpen {
		s.state = model.CircuitBreakerStateClosed
	}
	s.consecutiveFailures = 0
	atomic.StoreUint32(&s.probeInFlight, 0)
	return m.recordResultFromState(s, acquire.Decision)
}

func (m *Manager) RecordFailure(key, reason string, acquire AttemptAcquire, cfg model.CircuitBreakerConfig, now time.Time) RecordResult {
	s := m.getOrCreateState(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	applyDecay(s, now, time.Duration(cfg.DecayWindowMS)*time.Millisecond)
	s.lastFailureAt = now
	s.lastFailureReason = truncateReason(reason)
	s.consecutiveFailures++

	shouldTrip := s.consecutiveFailures >= cfg.FailureThreshold || acquire.ProbeGranted || s.state == model.CircuitBreakerStateHalfOpen
	decision := DecisionRecordFail
	if shouldTrip {
		s.tripCount++
		s.openUntil = now.Add(cooldownForTrip(s.tripCount, cfg))
		s.state = model.CircuitBreakerStateOpen
		s.lastTripAt = now
		s.consecutiveFailures = 0
		if acquire.ProbeGranted || acquire.StateBefore == model.CircuitBreakerStateHalfOpen {
			decision = DecisionProbeFailed
		}
	}
	atomic.StoreUint32(&s.probeInFlight, 0)
	return m.recordResultFromState(s, decision)
}

func (m *Manager) RecordNonTrippable(key string, acquire AttemptAcquire, cfg model.CircuitBreakerConfig) RecordResult {
	s := m.getOrCreateState(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.StoreUint32(&s.probeInFlight, 0)
	if cfg.Enabled && (acquire.ProbeGranted || s.state == model.CircuitBreakerStateHalfOpen) {
		s.state = model.CircuitBreakerStateClosed
		s.consecutiveFailures = 0
	}
	return m.recordResultFromState(s, acquire.Decision)
}

func (m *Manager) Snapshot(key string) Snapshot {
	s := m.getState(key)
	if s == nil {
		return Snapshot{
			Key:   key,
			State: model.CircuitBreakerStateClosed,
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return Snapshot{
		Key:                 key,
		State:               s.state,
		ConsecutiveFailures: s.consecutiveFailures,
		TripCount:           s.tripCount,
		OpenUntil:           s.openUntil,
		LastFailureAt:       s.lastFailureAt,
		LastFailureReason:   s.lastFailureReason,
		LastTripAt:          s.lastTripAt,
		ProbeInFlight:       atomic.LoadUint32(&s.probeInFlight) == 1,
	}
}

func (m *Manager) ResetChannel(channelID int) (int, []string) {
	prefix := fmt.Sprintf("%d:", channelID)
	keys := make([]string, 0)
	for i := 0; i < shardCount; i++ {
		sh := &m.shards[i]
		sh.mu.Lock()
		for key := range sh.m {
			if strings.HasPrefix(key, prefix) {
				delete(sh.m, key)
				keys = append(keys, key)
			}
		}
		sh.mu.Unlock()
	}
	return len(keys), keys
}

func (m *Manager) ResetKey(key string) bool {
	sh := &m.shards[hashKey(key)%shardCount]
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if _, ok := sh.m[key]; !ok {
		return false
	}
	delete(sh.m, key)
	return true
}

func (m *Manager) getState(key string) *breakerState {
	sh := &m.shards[hashKey(key)%shardCount]
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.m[key]
}

func (m *Manager) getOrCreateState(key string) *breakerState {
	sh := &m.shards[hashKey(key)%shardCount]
	sh.mu.RLock()
	state := sh.m[key]
	sh.mu.RUnlock()
	if state != nil {
		return state
	}

	sh.mu.Lock()
	defer sh.mu.Unlock()
	if state = sh.m[key]; state != nil {
		return state
	}
	state = &breakerState{
		state: model.CircuitBreakerStateClosed,
	}
	sh.m[key] = state
	return state
}

func (m *Manager) recordResultFromState(s *breakerState, decision Decision) RecordResult {
	return RecordResult{
		Decision:      decision,
		StateAfter:    s.state,
		TripCount:     s.tripCount,
		OpenUntil:     s.openUntil,
		ProbeInFlight: atomic.LoadUint32(&s.probeInFlight) == 1,
	}
}

func applyDecay(s *breakerState, now time.Time, decayWindow time.Duration) {
	if s.tripCount == 0 || s.lastTripAt.IsZero() || decayWindow <= 0 {
		return
	}
	steps := int(now.Sub(s.lastTripAt) / decayWindow)
	if steps <= 0 {
		return
	}
	s.tripCount = max(0, s.tripCount-steps)
}

func cooldownForTrip(tripCount int, cfg model.CircuitBreakerConfig) time.Duration {
	n := max(1, tripCount)
	base := float64(cfg.BaseCooldownMS)
	maxCD := float64(cfg.MaxCooldownMS)
	backoff := math.Pow(cfg.BackoffFactor, float64(n-1))
	cooldown := base * backoff
	if cooldown > maxCD {
		cooldown = maxCD
	}
	jitter := cfg.JitterMin + rand.Float64()*(cfg.JitterMax-cfg.JitterMin)
	if jitter < 0 {
		jitter = 0
	}
	cooldown *= jitter
	if cooldown < 1 {
		cooldown = 1
	}
	return time.Duration(cooldown) * time.Millisecond
}

func hashKey(key string) int {
	var h uint32 = 2166136261
	const prime32 = 16777619
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= prime32
	}
	return int(h)
}

func truncateReason(reason string) string {
	const maxLen = 512
	reason = strings.TrimSpace(reason)
	if len(reason) <= maxLen {
		return reason
	}
	return reason[:maxLen]
}
