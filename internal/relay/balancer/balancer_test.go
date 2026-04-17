package balancer

import (
	"fmt"
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/model"
)

func clearAutoStatsForTest() {
	globalAutoStats.Range(func(key, _ any) bool {
		globalAutoStats.Delete(key)
		return true
	})
}

func recordOutcome(channelID int, modelName string, success bool, count int) {
	for i := 0; i < count; i++ {
		stats := getOrCreateStats(channelID, modelName)
		stats.Record(success)
	}
}

func TestAutoCandidatesPreferLowerSampleCountDuringExploration(t *testing.T) {
	clearAutoStatsForTest()

	modelName := fmt.Sprintf("auto-explore-%d", time.Now().UnixNano())
	items := []model.GroupItem{
		{ChannelID: 1, ModelName: modelName, Weight: 100, Priority: 1},
		{ChannelID: 2, ModelName: modelName, Weight: 1, Priority: 2},
	}

	recordOutcome(1, modelName, true, 1)

	got := (&Auto{}).Candidates(items)
	if len(got) != 2 {
		t.Fatalf("Candidates() len = %d, want 2", len(got))
	}
	if got[0].ChannelID != 2 {
		t.Fatalf("Candidates()[0].ChannelID = %d, want 2", got[0].ChannelID)
	}
}

func TestAutoCandidatesUseWeightPriorityWhenAllSamplesAreZero(t *testing.T) {
	clearAutoStatsForTest()

	modelName := fmt.Sprintf("auto-zero-%d", time.Now().UnixNano())
	items := []model.GroupItem{
		{ChannelID: 1, ModelName: modelName, Weight: 1, Priority: 2},
		{ChannelID: 2, ModelName: modelName, Weight: 10, Priority: 1},
	}

	got := (&Auto{}).Candidates(items)
	if len(got) != 2 {
		t.Fatalf("Candidates() len = %d, want 2", len(got))
	}
	if got[0].ChannelID != 2 {
		t.Fatalf("Candidates()[0].ChannelID = %d, want 2", got[0].ChannelID)
	}
}

func TestAutoCandidatesPreferHigherSuccessRateAfterMinSamples(t *testing.T) {
	clearAutoStatsForTest()

	modelName := fmt.Sprintf("auto-exploit-%d", time.Now().UnixNano())
	items := []model.GroupItem{
		{ChannelID: 1, ModelName: modelName, Weight: 1, Priority: 1},
		{ChannelID: 2, ModelName: modelName, Weight: 100, Priority: 2},
	}

	recordOutcome(1, modelName, true, 10)
	recordOutcome(2, modelName, true, 6)
	recordOutcome(2, modelName, false, 4)

	got := (&Auto{}).Candidates(items)
	if len(got) != 2 {
		t.Fatalf("Candidates() len = %d, want 2", len(got))
	}
	if got[0].ChannelID != 1 {
		t.Fatalf("Candidates()[0].ChannelID = %d, want 1", got[0].ChannelID)
	}
}

func TestAutoCandidatesUseWeightPriorityAsTieBreaker(t *testing.T) {
	clearAutoStatsForTest()

	modelName := fmt.Sprintf("auto-tie-%d", time.Now().UnixNano())
	items := []model.GroupItem{
		{ChannelID: 1, ModelName: modelName, Weight: 1, Priority: 2},
		{ChannelID: 2, ModelName: modelName, Weight: 10, Priority: 1},
	}

	recordOutcome(1, modelName, true, 10)
	recordOutcome(2, modelName, true, 10)

	got := (&Auto{}).Candidates(items)
	if len(got) != 2 {
		t.Fatalf("Candidates() len = %d, want 2", len(got))
	}
	if got[0].ChannelID != 2 {
		t.Fatalf("Candidates()[0].ChannelID = %d, want 2", got[0].ChannelID)
	}
}
