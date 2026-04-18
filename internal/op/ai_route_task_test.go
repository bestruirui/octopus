package op

import (
	"testing"

	"github.com/lingyuins/octopus/internal/model"
)

func TestInterruptAIRouteTaskProgressMarksRunningChannelsFailed(t *testing.T) {
	progress := model.GenerateAIRouteProgress{
		Status:          model.AIRouteTaskStatusRunning,
		CurrentStep:     model.AIRouteTaskStepAnalyzingBatches,
		ProgressPercent: 67,
		Channels: []model.GenerateAIRouteChannelProgress{
			{
				ChannelID:       1,
				ChannelName:     "alpha",
				Status:          model.AIRouteChannelStatusRunning,
				TotalModels:     2,
				ProcessedModels: 1,
			},
			{
				ChannelID:       2,
				ChannelName:     "beta",
				Status:          model.AIRouteChannelStatusCompleted,
				TotalModels:     1,
				ProcessedModels: 1,
			},
		},
		Summary: &model.GenerateAIRouteProgressSummary{
			TotalChannels:     2,
			CompletedChannels: 1,
			RunningChannels:   1,
			TotalModels:       3,
			CompletedModels:   2,
		},
	}

	interruptAIRouteTaskProgress(&progress, "interrupted")

	if !progress.Done || progress.Status != model.AIRouteTaskStatusFailed || progress.CurrentStep != model.AIRouteTaskStepFailed {
		t.Fatalf("progress terminal state not updated: %+v", progress)
	}
	if progress.Channels[0].Status != model.AIRouteChannelStatusFailed {
		t.Fatalf("running channel status = %q, want failed", progress.Channels[0].Status)
	}
	if progress.Channels[0].Message != "interrupted" {
		t.Fatalf("running channel message = %q, want interrupted", progress.Channels[0].Message)
	}
	if progress.Summary == nil || progress.Summary.FailedChannels != 1 || progress.Summary.CompletedChannels != 1 {
		t.Fatalf("summary not recomputed correctly: %+v", progress.Summary)
	}
}
