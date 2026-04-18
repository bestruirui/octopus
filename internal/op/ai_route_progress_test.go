package op

import (
	"testing"

	"github.com/lingyuins/octopus/internal/model"
)

func TestAIRouteProgressTrackerTracksRunningBatchesAndRetries(t *testing.T) {
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
		{ChannelID: 3, ChannelName: "gamma", Provider: "openai", Model: "text-embedding-3-large"},
	}
	bucketOne := aiRoutePromptBucket{
		PromptEndpointType: model.EndpointTypeChat,
		GroupEndpointType:  model.EndpointTypeAll,
		ModelInputs: []aiRoutePromptModelInput{
			{ChannelID: 1, Model: "gpt-4o"},
			{ChannelID: 2, Model: "claude-sonnet-4.5"},
		},
	}
	bucketTwo := aiRoutePromptBucket{
		PromptEndpointType: model.EndpointTypeEmbeddings,
		GroupEndpointType:  model.EndpointTypeEmbeddings,
		ModelInputs: []aiRoutePromptModelInput{
			{ChannelID: 3, Model: "text-embedding-3-large"},
		},
	}

	tracker.SetModelInputs(inputs)
	tracker.SetBuckets([]aiRoutePromptBucket{bucketOne, bucketTwo})
	tracker.StartBatch(1, bucketOne, "svc-a", 1)
	tracker.StartBatch(2, bucketTwo, "svc-b", 1)
	tracker.MarkBatchRetry(1, bucketOne, "svc-a", 1, "429")
	tracker.StartBatch(1, bucketOne, "svc-c", 2)
	runningSnapshot := snapshots[len(snapshots)-1]

	if runningSnapshot.CurrentBatch == nil {
		t.Fatal("running snapshot current batch = nil, want non-nil")
	}
	if runningSnapshot.CurrentBatch.ServiceName != "svc-c" || runningSnapshot.CurrentBatch.Attempt != 2 {
		t.Fatalf("running snapshot current batch = %+v, want svc-c attempt 2", runningSnapshot.CurrentBatch)
	}
	if len(runningSnapshot.RunningBatches) != 2 {
		t.Fatalf("running snapshot running batches len = %d, want 2", len(runningSnapshot.RunningBatches))
	}
	if runningSnapshot.Summary == nil || runningSnapshot.Summary.RunningChannels != 3 {
		t.Fatalf("running snapshot summary = %+v, want running_channels=3", runningSnapshot.Summary)
	}

	tracker.MarkBatchAIResponseReceived(2, bucketTwo, "svc-b", 1)
	parsingSnapshot := snapshots[len(snapshots)-1]
	if len(parsingSnapshot.RunningBatches) < 2 || parsingSnapshot.RunningBatches[1].Status != model.AIRouteBatchStatusParsing {
		t.Fatalf("parsing snapshot running batches = %+v, want batch 2 parsing", parsingSnapshot.RunningBatches)
	}

	tracker.CompleteBatch(2, bucketTwo, "svc-b", 1)
	tracker.CompleteBatch(1, bucketOne, "svc-c", 2)
	completedSnapshot := snapshots[len(snapshots)-1]

	if completedSnapshot.CompletedBatches != 2 || completedSnapshot.TotalBatches != 2 {
		t.Fatalf("completed snapshot batches = %d/%d, want 2/2", completedSnapshot.CompletedBatches, completedSnapshot.TotalBatches)
	}
	if completedSnapshot.CurrentBatch == nil || completedSnapshot.CurrentBatch.Status != "completed" {
		t.Fatalf("completed snapshot current batch = %+v, want completed batch", completedSnapshot.CurrentBatch)
	}
	if len(completedSnapshot.RunningBatches) != 0 {
		t.Fatalf("completed snapshot running batches len = %d, want 0", len(completedSnapshot.RunningBatches))
	}
	if completedSnapshot.Summary == nil || completedSnapshot.Summary.CompletedModels != 3 {
		t.Fatalf("completed snapshot summary = %+v, want completed_models=3", completedSnapshot.Summary)
	}
	for i, channel := range completedSnapshot.Channels {
		if channel.Status != model.AIRouteChannelStatusCompleted {
			t.Fatalf("completed snapshot channel[%d] status = %q, want completed", i, channel.Status)
		}
	}

	if runningSnapshot.RunningBatches[0].ServiceName != "svc-a" && runningSnapshot.RunningBatches[0].ServiceName != "svc-c" {
		t.Fatalf("running snapshot mutated unexpectedly: %+v", runningSnapshot.RunningBatches)
	}
}
