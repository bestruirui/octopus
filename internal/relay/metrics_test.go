package relay

import "testing"

func TestSuccessfulCallCost(t *testing.T) {
	if got := successfulCallCost(true, 0.25); got != 0.25 {
		t.Fatalf("successful request cost = %v, want 0.25", got)
	}
	if got := successfulCallCost(false, 0.25); got != 0 {
		t.Fatalf("failed request cost = %v, want 0", got)
	}
}