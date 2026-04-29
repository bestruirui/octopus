package semantic_cache

import (
	"testing"
	"time"
)

func TestLookup_IsolatedByNamespace(t *testing.T) {
	Reset()
	ApplyRuntimeConfig(RuntimeConfig{Enabled: true, MaxEntries: 16, Threshold: 0.95, TTL: time.Hour})

	embedding := []float64{1, 0}
	Store("k1:chat:gpt-4.1", "req-a", []byte(`{"id":"resp-a"}`), embedding)

	if _, ok := Lookup("k2:chat:gpt-4.1", embedding); ok {
		t.Fatal("expected namespace miss")
	}
}
