package op

import (
	"testing"

	"github.com/lingyuins/octopus/internal/model"
)

func TestAIRouteProgressTrackerEmitsIndependentSnapshots(t *testing.T) {
	snapshots := make([]model.GenerateAIRouteProgress, 0)
	tracker := newAIRouteProgressTracker(
		model.GenerateAIRouteRequest{Scope: model.AIRouteScopeTable},
		func(progress model.GenerateAIRouteProgress) {
			snapshots = append(snapshots, progress)
		},
	)

	inputs := []model.AIRouteModelInput{
		{ChannelID: 1, ChannelName: "alpha", Provider: "openai", Model: "gpt-4o"},
		{ChannelID: 2, ChannelName: "beta", Provider: "anthropic", Model: "claude-sonnet-4.5"},
	}
	bucket := aiRoutePromptBucket{
		PromptEndpointType: model.EndpointTypeChat,
		GroupEndpointType:  model.EndpointTypeAll,
		ModelInputs: []aiRoutePromptModelInput{
			{ChannelID: 1, Model: "gpt-4o"},
			{ChannelID: 2, Model: "claude-sonnet-4.5"},
		},
	}

	tracker.SetModelInputs(inputs)
	tracker.SetBuckets([]aiRoutePromptBucket{bucket})
	tracker.StartBatch(1, bucket)
	runningSnapshot := snapshots[len(snapshots)-1]

	tracker.MarkBatchAIResponseReceived(1, bucket)
	tracker.CompleteBatch(1, bucket)

	completedSnapshot := snapshots[len(snapshots)-1]

	if runningSnapshot.CurrentStep != model.AIRouteTaskStepAnalyzingBatches {
		t.Fatalf("running snapshot step = %q, want %q", runningSnapshot.CurrentStep, model.AIRouteTaskStepAnalyzingBatches)
	}
	if runningSnapshot.CurrentBatch == nil || runningSnapshot.CurrentBatch.Index != 1 || runningSnapshot.CurrentBatch.Total != 1 {
		t.Fatalf("running snapshot current batch = %+v, want index=1 total=1", runningSnapshot.CurrentBatch)
	}
	if len(runningSnapshot.Channels) != 2 {
		t.Fatalf("running snapshot channel len = %d, want 2", len(runningSnapshot.Channels))
	}
	if runningSnapshot.Channels[0].Status != model.AIRouteChannelStatusRunning {
		t.Fatalf("running snapshot channel[0] status = %q, want %q", runningSnapshot.Channels[0].Status, model.AIRouteChannelStatusRunning)
	}
	if runningSnapshot.Channels[0].ProcessedModels != 0 {
		t.Fatalf("running snapshot channel[0].processed_models = %d, want 0", runningSnapshot.Channels[0].ProcessedModels)
	}

	if completedSnapshot.CompletedBatches != 1 || completedSnapshot.TotalBatches != 1 {
		t.Fatalf("completed snapshot batches = %d/%d, want 1/1", completedSnapshot.CompletedBatches, completedSnapshot.TotalBatches)
	}
	if completedSnapshot.ProgressPercent != 80 {
		t.Fatalf("completed snapshot progress = %d, want 80", completedSnapshot.ProgressPercent)
	}
	if completedSnapshot.Summary == nil || completedSnapshot.Summary.CompletedModels != 2 {
		t.Fatalf("completed snapshot summary = %+v, want completed_models=2", completedSnapshot.Summary)
	}
	if completedSnapshot.Channels[0].Status != model.AIRouteChannelStatusCompleted {
		t.Fatalf("completed snapshot channel[0] status = %q, want %q", completedSnapshot.Channels[0].Status, model.AIRouteChannelStatusCompleted)
	}
	if completedSnapshot.Channels[0].ProcessedModels != 1 {
		t.Fatalf("completed snapshot channel[0].processed_models = %d, want 1", completedSnapshot.Channels[0].ProcessedModels)
	}

	if runningSnapshot.Channels[0].Status != model.AIRouteChannelStatusRunning {
		t.Fatalf("running snapshot channel[0] mutated to %q, want independent running snapshot", runningSnapshot.Channels[0].Status)
	}
	if runningSnapshot.Channels[0].ProcessedModels != 0 {
		t.Fatalf("running snapshot channel[0].processed_models mutated to %d, want 0", runningSnapshot.Channels[0].ProcessedModels)
	}
}
