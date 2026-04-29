package semantic_cache

import (
	"bytes"
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

func TestStore_CopiesResponseJSON(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	ApplyRuntimeConfig(RuntimeConfig{Enabled: true, MaxEntries: 16, Threshold: 0.95, TTL: time.Hour})

	embedding := []float64{1, 0}
	responseJSON := []byte(`{"id":"resp-a"}`)
	Store("k1:chat:gpt-4.1", "req-a", responseJSON, embedding)

	copy(responseJSON, []byte(`{"id":"mutate"}`))

	got, ok := Lookup("k1:chat:gpt-4.1", embedding)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !bytes.Equal(got, []byte(`{"id":"resp-a"}`)) {
		t.Fatalf("Lookup() = %s, want original response", string(got))
	}
}

func TestLookup_ReturnsResponseCopy(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	ApplyRuntimeConfig(RuntimeConfig{Enabled: true, MaxEntries: 16, Threshold: 0.95, TTL: time.Hour})

	embedding := []float64{1, 0}
	Store("k1:chat:gpt-4.1", "req-a", []byte(`{"id":"resp-a"}`), embedding)

	got, ok := Lookup("k1:chat:gpt-4.1", embedding)
	if !ok {
		t.Fatal("expected first cache hit")
	}
	copy(got, []byte(`{"id":"mutate"}`))

	gotAgain, ok := Lookup("k1:chat:gpt-4.1", embedding)
	if !ok {
		t.Fatal("expected second cache hit")
	}
	if !bytes.Equal(gotAgain, []byte(`{"id":"resp-a"}`)) {
		t.Fatalf("Lookup() after caller mutation = %s, want original response", string(gotAgain))
	}
}
