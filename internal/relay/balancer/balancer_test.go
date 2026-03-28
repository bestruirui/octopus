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

func TestWeightedCandidates_Reduces1hImpactWhenSamplesLow(t *testing.T) {
	t.Cleanup(resetSmartStatsForTest)
	base := time.Unix(120*60, 0).UTC()
	smartNowFunc = func() time.Time { return base }
	t.Cleanup(func() { smartNowFunc = time.Now })

	// channel 1: higher manual weight, mediocre 24h, tiny 1h sample with perfect success
	for i := 0; i < 60; i++ {
		recordSmartOutcome(1, "m", true)
	}
	for i := 0; i < 60; i++ {
		recordSmartOutcome(1, "m", false)
	}
	for i := 0; i < 5; i++ {
		recordSmartOutcome(1, "m", true)
	}

	// channel 2: lower manual weight, good 24h, tiny 1h sample with complete failure
	for i := 0; i < 120; i++ {
		recordSmartOutcome(2, "m", true)
	}
	for i := 0; i < 5; i++ {
		recordSmartOutcome(2, "m", false)
	}

	items := []model.GroupItem{
		{ChannelID: 1, ModelName: "m", Weight: 70, Priority: 10},
		{ChannelID: 2, ModelName: "m", Weight: 30, Priority: 10},
	}
	got := (&Weighted{}).Candidates(items)
	if got[0].ChannelID != 2 {
		t.Fatalf("expected channel 2 first due to stronger long-window health with low 1h samples, got %d", got[0].ChannelID)
	}
}

func TestWeightedCandidates_PenalizesOneHourComponentWhenRecentFailuresExist(t *testing.T) {
	t.Cleanup(resetSmartStatsForTest)
	base := time.Unix(240*60, 0).UTC()
	smartNowFunc = func() time.Time { return base }
	t.Cleanup(func() { smartNowFunc = time.Now })

	// Same manual weight and nearly same 24h health.
	// Channel 1 has one recent 1h failure, so its 1h component should be divided by 3.
	for i := 0; i < 30; i++ {
		recordSmartOutcome(1, "m", true)
	}
	recordSmartOutcome(1, "m", false)

	for i := 0; i < 30; i++ {
		recordSmartOutcome(2, "m", true)
	}

	items := []model.GroupItem{
		{ChannelID: 1, ModelName: "m", Weight: 50, Priority: 10},
		{ChannelID: 2, ModelName: "m", Weight: 50, Priority: 10},
	}

	got := (&Weighted{}).Candidates(items)
	if got[0].ChannelID != 2 {
		t.Fatalf("expected channel 2 first because channel 1 has 1h failure penalty, got %d", got[0].ChannelID)
	}
}

func TestWeightedCandidates_StableTopOrderWithoutDiversifyRotation(t *testing.T) {
	t.Cleanup(resetSmartStatsForTest)
	base := time.Unix(120*60, 0).UTC()
	smartNowFunc = func() time.Time { return base }
	t.Cleanup(func() { smartNowFunc = time.Now })

	for _, ch := range []int{1, 2, 3} {
		for i := 0; i < 50; i++ {
			recordSmartOutcome(ch, "m", true)
		}
	}

	items := []model.GroupItem{
		{ChannelID: 1, ModelName: "m", Weight: 100, Priority: 10},
		{ChannelID: 2, ModelName: "m", Weight: 100, Priority: 10},
		{ChannelID: 3, ModelName: "m", Weight: 100, Priority: 10},
	}

	w := &Weighted{}
	for i := 0; i < 3; i++ {
		got := w.Candidates(items)
		if got[0].ChannelID != 1 {
			t.Fatalf("expected stable deterministic top candidate channel 1, got %d", got[0].ChannelID)
		}
	}
}
