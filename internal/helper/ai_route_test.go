package helper

import (
	"sync"
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/model"
)

func TestGetGenerateAIRouteProgressExpiresDoneEntries(t *testing.T) {
	originalNow := aiRouteProgressNow
	originalProgress := aiRouteProgress
	defer func() {
		aiRouteProgressNow = originalNow
		aiRouteProgress = originalProgress
	}()

	aiRouteProgress = sync.Map{}
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	aiRouteProgressNow = func() time.Time { return now }

	storeAIRouteProgress(&model.GenerateAIRouteProgress{
		ID:   "done-progress",
		Done: true,
	})

	if _, ok := GetGenerateAIRouteProgress("done-progress"); !ok {
		t.Fatal("GetGenerateAIRouteProgress() ok = false, want true before expiry")
	}

	now = now.Add(aiRouteProgressDoneTTL + time.Second)
	if _, ok := GetGenerateAIRouteProgress("done-progress"); ok {
		t.Fatal("GetGenerateAIRouteProgress() ok = true, want false after expiry")
	}
}
