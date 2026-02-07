package breaker

import (
	"testing"
	"time"

	"github.com/bestruirui/octopus/internal/model"
)

func testConfig() model.CircuitBreakerConfig {
	return model.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
		BaseCooldownMS:   1000,
		MaxCooldownMS:    10000,
		BackoffFactor:    2,
		JitterMin:        1,
		JitterMax:        1,
		DecayWindowMS:    1000,
	}
}

func TestProbeSingleFlight(t *testing.T) {
	m := NewManager()
	cfg := testConfig()
	key := BuildKey(1, "glm-4.7")
	now := time.Now()

	// 先连续失败触发 OPEN
	a1 := m.Acquire(key, now, cfg)
	m.RecordFailure(key, "upstream 500", a1, cfg, now)
	a2 := m.Acquire(key, now, cfg)
	m.RecordFailure(key, "upstream 500", a2, cfg, now)

	// OPEN 窗口内应拒绝
	rejected := m.Acquire(key, now.Add(100*time.Millisecond), cfg)
	if rejected.Allowed {
		t.Fatalf("expected acquire rejected while OPEN")
	}

	// 过了 openUntil 后，只允许一个 probe
	snap := m.Snapshot(key)
	probe1 := m.Acquire(key, snap.OpenUntil.Add(10*time.Millisecond), cfg)
	probe2 := m.Acquire(key, snap.OpenUntil.Add(10*time.Millisecond), cfg)

	if !probe1.Allowed && !probe2.Allowed {
		t.Fatalf("expected one probe to be allowed")
	}
	if probe1.Allowed && probe2.Allowed {
		t.Fatalf("expected only one probe to be allowed")
	}
}

func TestDecayTripCount(t *testing.T) {
	m := NewManager()
	cfg := testConfig()
	key := BuildKey(2, "gpt-4o")
	now := time.Now()

	// trip 1
	a1 := m.Acquire(key, now, cfg)
	m.RecordFailure(key, "x", a1, cfg, now)
	a2 := m.Acquire(key, now, cfg)
	m.RecordFailure(key, "x", a2, cfg, now)
	s1 := m.Snapshot(key)
	if s1.TripCount != 1 {
		t.Fatalf("expected tripCount=1, got %d", s1.TripCount)
	}

	// 长时间稳定后再次故障，先衰减再计算
	later := now.Add(5 * time.Second)
	a3 := m.Acquire(key, later, cfg)
	r3 := m.RecordFailure(key, "x", a3, cfg, later)
	if r3.TripCount != 1 {
		t.Fatalf("expected decay to reduce tripCount before re-trip, got %d", r3.TripCount)
	}
}
