package helper

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
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
