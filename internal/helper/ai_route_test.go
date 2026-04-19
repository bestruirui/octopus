package helper

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
)

func snapshotAIRouteProgressEntries() map[string]aiRouteProgressEntry {
	snapshot := make(map[string]aiRouteProgressEntry)
	aiRouteProgress.Range(func(key, value any) bool {
		id, ok := key.(string)
		if !ok {
			return true
		}
		entry, ok := value.(aiRouteProgressEntry)
		if !ok {
			return true
		}
		snapshot[id] = entry
		return true
	})
	return snapshot
}

func restoreAIRouteProgressEntries(snapshot map[string]aiRouteProgressEntry) {
	aiRouteProgress = sync.Map{}
	for id, entry := range snapshot {
		aiRouteProgress.Store(id, entry)
	}
}

func TestGetGenerateAIRouteProgressExpiresDoneEntries(t *testing.T) {
	originalNow := aiRouteProgressNow
	originalProgress := snapshotAIRouteProgressEntries()
	defer func() {
		aiRouteProgressNow = originalNow
		restoreAIRouteProgressEntries(originalProgress)
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

func TestFinalizeAIRouteProgressPreservesResultOnPartialFailure(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	originalNow := aiRouteProgressNow
	defer func() {
		aiRouteProgressNow = originalNow
	}()
	aiRouteProgressNow = func() time.Time { return now }

	progress := &model.GenerateAIRouteProgress{
		Status:          model.AIRouteTaskStatusRunning,
		CurrentStep:     model.AIRouteTaskStepWritingGroups,
		ProgressPercent: 96,
	}
	result := &model.GenerateAIRouteResult{
		Scope:      model.AIRouteScopeTable,
		GroupCount: 2,
		RouteCount: 4,
		ItemCount:  9,
	}

	finalizeAIRouteProgress(progress, result, &op.AIRoutePartialFailureError{
		Message: "AI 路由部分失败，但已保留成功写入的 2 个分组",
		Cause:   errors.New("第 2/3 批 AI 分析失败"),
	}, nil)

	if progress.Status != model.AIRouteTaskStatusFailed {
		t.Fatalf("finalizeAIRouteProgress() status = %q, want failed", progress.Status)
	}
	if !progress.ResultReady {
		t.Fatal("finalizeAIRouteProgress() result_ready = false, want true")
	}
	if progress.Result == nil || progress.Result.GroupCount != 2 {
		t.Fatalf("finalizeAIRouteProgress() result = %+v, want preserved result", progress.Result)
	}
	if progress.Message != "AI 路由部分失败，但已保留成功写入的 2 个分组" {
		t.Fatalf("finalizeAIRouteProgress() message = %q, want preserved partial failure message", progress.Message)
	}
}
