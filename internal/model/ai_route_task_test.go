package model

import (
	"testing"
	"time"
)

func TestAIRouteTaskRoundTripPreservesProgressSnapshot(t *testing.T) {
	startedAt := time.Unix(1710000000, 0).UTC()
	updatedAt := startedAt.Add(5 * time.Second)
	heartbeatAt := updatedAt.Add(3 * time.Second)
	finishedAt := heartbeatAt.Add(7 * time.Second)

	original := GenerateAIRouteProgress{
		ID:               "task-1",
		Scope:            AIRouteScopeTable,
		GroupID:          42,
		Status:           AIRouteTaskStatusCompleted,
		CurrentStep:      AIRouteTaskStepCompleted,
		ProgressPercent:  100,
		TotalBatches:     3,
		CompletedBatches: 3,
		Done:             true,
		ResultReady:      true,
		Message:          "AI 路由生成完成",
		ErrorReason:      "",
		StartedAt:        &startedAt,
		UpdatedAt:        &updatedAt,
		HeartbeatAt:      &heartbeatAt,
		FinishedAt:       &finishedAt,
		EventSequence:    9,
		Summary: &GenerateAIRouteProgressSummary{
			TotalChannels:     2,
			CompletedChannels: 2,
			TotalModels:       3,
			CompletedModels:   3,
		},
		CurrentBatch: &GenerateAIRouteCurrentBatch{
			Index:        3,
			Total:        3,
			EndpointType: EndpointTypeChat,
			ModelCount:   2,
			ChannelIDs:   []int{1, 2},
			ChannelNames: []string{"alpha", "beta"},
			ServiceName:  "svc-a",
			Attempt:      2,
			Status:       "completed",
			Message:      "当前批次已完成",
		},
		RunningBatches: []GenerateAIRouteRunningBatch{
			{
				Index:        2,
				Total:        3,
				EndpointType: EndpointTypeEmbeddings,
				ModelCount:   1,
				ChannelIDs:   []int{2},
				ChannelNames: []string{"beta"},
				ServiceName:  "svc-b",
				Attempt:      1,
				Status:       AIRouteBatchStatusParsing,
				Message:      "解析中",
			},
		},
		Channels: []GenerateAIRouteChannelProgress{
			{
				ChannelID:       1,
				ChannelName:     "alpha",
				Status:          AIRouteChannelStatusCompleted,
				TotalModels:     2,
				ProcessedModels: 2,
			},
			{
				ChannelID:       2,
				ChannelName:     "beta",
				Status:          AIRouteChannelStatusCompleted,
				TotalModels:     1,
				ProcessedModels: 1,
			},
		},
		Result: &GenerateAIRouteResult{
			Scope:      AIRouteScopeTable,
			GroupCount: 2,
			RouteCount: 3,
			ItemCount:  5,
		},
	}

	task := NewAIRouteTask(original)
	got := task.ToProgress()

	if got.ID != original.ID || got.Scope != original.Scope || got.GroupID != original.GroupID {
		t.Fatalf("task progress basic fields changed: got %+v want %+v", got, original)
	}
	if !got.Done || !got.ResultReady || got.EventSequence != original.EventSequence {
		t.Fatalf("task progress state fields changed: got %+v", got)
	}
	if got.Summary == nil || got.Summary.TotalModels != original.Summary.TotalModels {
		t.Fatalf("summary not preserved: got %+v", got.Summary)
	}
	if got.CurrentBatch == nil || len(got.CurrentBatch.ChannelIDs) != 2 || got.CurrentBatch.ChannelNames[1] != "beta" {
		t.Fatalf("current batch not preserved: got %+v", got.CurrentBatch)
	}
	if len(got.RunningBatches) != 1 || got.RunningBatches[0].ServiceName != "svc-b" || got.RunningBatches[0].Status != AIRouteBatchStatusParsing {
		t.Fatalf("running batches not preserved: got %+v", got.RunningBatches)
	}
	if len(got.Channels) != 2 || got.Channels[0].ChannelName != "alpha" || got.Channels[1].ProcessedModels != 1 {
		t.Fatalf("channels not preserved: got %+v", got.Channels)
	}
	if got.Result == nil || got.Result.RouteCount != original.Result.RouteCount || got.Result.ItemCount != original.Result.ItemCount {
		t.Fatalf("result not preserved: got %+v", got.Result)
	}
}
