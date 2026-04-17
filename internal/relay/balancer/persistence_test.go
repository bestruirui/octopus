package balancer

import (
	"math"
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/model"
)

func clearCircuitBreakerForTest() {
	globalBreaker.Range(func(key, _ any) bool {
		globalBreaker.Delete(key)
		return true
	})
}

func TestRestoreChannelStatsKeepsRecentSamplesWithinWindow(t *testing.T) {
	clearAutoStatsForTest()

	now := time.Now()
	stats := restoreChannelStats([]model.AutoStrategyRecord{
		{Timestamp: now.Add(-10 * time.Minute).UnixMilli(), Success: true},
		{Timestamp: now.Add(-4 * time.Minute).UnixMilli(), Success: true},
		{Timestamp: now.Add(-3 * time.Minute).UnixMilli(), Success: false},
		{Timestamp: now.Add(-2 * time.Minute).UnixMilli(), Success: true},
		{Timestamp: now.Add(-1 * time.Minute).UnixMilli(), Success: true},
	}, now, 5*time.Minute, 3)

	if stats == nil {
		t.Fatalf("restoreChannelStats() = nil, want non-nil")
	}

	successRate, totalSamples := stats.GetStats(5 * time.Minute)
	if totalSamples != 3 {
		t.Fatalf("GetStats() totalSamples = %d, want 3", totalSamples)
	}

	wantRate := 2.0 / 3.0
	if math.Abs(successRate-wantRate) > 1e-9 {
		t.Fatalf("GetStats() successRate = %f, want %f", successRate, wantRate)
	}
}

func TestRestoreCircuitEntryConvertsHalfOpenToOpen(t *testing.T) {
	clearCircuitBreakerForTest()

	entry := restoreCircuitEntry(model.CircuitBreakerState{
		State:               int(StateHalfOpen),
		ConsecutiveFailures: 1,
		LastFailureTime:     time.Now().Add(-time.Minute).UnixMilli(),
		TripCount:           2,
	})

	if entry == nil {
		t.Fatalf("restoreCircuitEntry() = nil, want non-nil")
	}
	if entry.State != StateOpen {
		t.Fatalf("restoreCircuitEntry() state = %v, want %v", entry.State, StateOpen)
	}
	if entry.TripCount != 2 {
		t.Fatalf("restoreCircuitEntry() tripCount = %d, want 2", entry.TripCount)
	}
}
