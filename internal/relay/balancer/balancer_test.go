package balancer

import (
	"testing"
	"time"

	"github.com/bestruirui/octopus/internal/model"
)

func TestWeightedCandidates_PrioritizesHealthAndWeight(t *testing.T) {
	t.Cleanup(resetSmartStatsForTest)

	base := time.Unix(0, 0).UTC()
	smartNowFunc = func() time.Time { return base }
	t.Cleanup(func() { smartNowFunc = time.Now })

	items := []model.GroupItem{
		{ChannelID: 1, ModelName: "m", Weight: 100, Priority: 10},
		{ChannelID: 2, ModelName: "m", Weight: 10, Priority: 10},
	}

	// channel 1: recent failures dominate
	for i := 0; i < 20; i++ {
		recordSmartOutcome(1, "m", false)
	}
	for i := 0; i < 2; i++ {
		recordSmartOutcome(1, "m", true)
	}

	// channel 2: mostly healthy
	for i := 0; i < 20; i++ {
		recordSmartOutcome(2, "m", true)
	}
	for i := 0; i < 1; i++ {
		recordSmartOutcome(2, "m", false)
	}

	got := (&Weighted{}).Candidates(items)
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0].ChannelID != 2 {
		t.Fatalf("expected healthier channel first, got channel %d", got[0].ChannelID)
	}
}

func TestWeightedCandidates_UsesManualWeightWhenNoStats(t *testing.T) {
	t.Cleanup(resetSmartStatsForTest)

	items := []model.GroupItem{
		{ChannelID: 1, ModelName: "m", Weight: 5, Priority: 10},
		{ChannelID: 2, ModelName: "m", Weight: 50, Priority: 10},
	}

	got := (&Weighted{}).Candidates(items)
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0].ChannelID != 2 {
		t.Fatalf("expected higher manual weight first when no stats, got channel %d", got[0].ChannelID)
	}
}
