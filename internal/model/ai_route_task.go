package model

import "time"

type AIRouteTask struct {
	ID               string                           `json:"id" gorm:"primaryKey;size:64"`
	Scope            AIRouteScope                     `json:"scope" gorm:"size:16;not null;index:idx_ai_route_task_scope_group_done,priority:1"`
	GroupID          int                              `json:"group_id" gorm:"not null;default:0;index:idx_ai_route_task_scope_group_done,priority:2"`
	Status           AIRouteTaskStatus                `json:"status" gorm:"size:32;not null;index"`
	CurrentStep      AIRouteTaskStep                  `json:"current_step" gorm:"size:32;not null"`
	ProgressPercent  int                              `json:"progress_percent" gorm:"not null;default:0"`
	TotalBatches     int                              `json:"total_batches" gorm:"not null;default:0"`
	CompletedBatches int                              `json:"completed_batches" gorm:"not null;default:0"`
	Done             bool                             `json:"done" gorm:"not null;default:false;index:idx_ai_route_task_scope_group_done,priority:3"`
	ResultReady      bool                             `json:"result_ready" gorm:"not null;default:false"`
	Message          string                           `json:"message" gorm:"type:text"`
	ErrorReason      string                           `json:"error_reason" gorm:"type:text"`
	StartedAt        *time.Time                       `json:"started_at,omitempty"`
	UpdatedAt        *time.Time                       `json:"updated_at,omitempty"`
	HeartbeatAt      *time.Time                       `json:"heartbeat_at,omitempty;index"`
	FinishedAt       *time.Time                       `json:"finished_at,omitempty"`
	EventSequence    int64                            `json:"event_sequence" gorm:"not null;default:0"`
	Summary          *GenerateAIRouteProgressSummary  `json:"summary,omitempty" gorm:"serializer:json"`
	CurrentBatch     *GenerateAIRouteCurrentBatch     `json:"current_batch,omitempty" gorm:"serializer:json"`
	Channels         []GenerateAIRouteChannelProgress `json:"channels,omitempty" gorm:"serializer:json"`
	Result           *GenerateAIRouteResult           `json:"result,omitempty" gorm:"serializer:json"`
}

func (AIRouteTask) TableName() string { return "ai_route_tasks" }

func NewAIRouteTask(progress GenerateAIRouteProgress) AIRouteTask {
	return AIRouteTask{
		ID:               progress.ID,
		Scope:            progress.Scope,
		GroupID:          progress.GroupID,
		Status:           progress.Status,
		CurrentStep:      progress.CurrentStep,
		ProgressPercent:  progress.ProgressPercent,
		TotalBatches:     progress.TotalBatches,
		CompletedBatches: progress.CompletedBatches,
		Done:             progress.Done,
		ResultReady:      progress.ResultReady,
		Message:          progress.Message,
		ErrorReason:      progress.ErrorReason,
		StartedAt:        cloneAIRouteTaskTime(progress.StartedAt),
		UpdatedAt:        cloneAIRouteTaskTime(progress.UpdatedAt),
		HeartbeatAt:      cloneAIRouteTaskTime(progress.HeartbeatAt),
		FinishedAt:       cloneAIRouteTaskTime(progress.FinishedAt),
		EventSequence:    progress.EventSequence,
		Summary:          cloneAIRouteTaskSummary(progress.Summary),
		CurrentBatch:     cloneAIRouteTaskCurrentBatch(progress.CurrentBatch),
		Channels:         cloneAIRouteTaskChannels(progress.Channels),
		Result:           cloneAIRouteTaskResult(progress.Result),
	}
}

func (task *AIRouteTask) ToProgress() GenerateAIRouteProgress {
	if task == nil {
		return GenerateAIRouteProgress{}
	}

	return GenerateAIRouteProgress{
		ID:               task.ID,
		Scope:            task.Scope,
		GroupID:          task.GroupID,
		Status:           task.Status,
		CurrentStep:      task.CurrentStep,
		ProgressPercent:  task.ProgressPercent,
		TotalBatches:     task.TotalBatches,
		CompletedBatches: task.CompletedBatches,
		Done:             task.Done,
		ResultReady:      task.ResultReady,
		Message:          task.Message,
		ErrorReason:      task.ErrorReason,
		StartedAt:        cloneAIRouteTaskTime(task.StartedAt),
		UpdatedAt:        cloneAIRouteTaskTime(task.UpdatedAt),
		HeartbeatAt:      cloneAIRouteTaskTime(task.HeartbeatAt),
		FinishedAt:       cloneAIRouteTaskTime(task.FinishedAt),
		EventSequence:    task.EventSequence,
		Summary:          cloneAIRouteTaskSummary(task.Summary),
		CurrentBatch:     cloneAIRouteTaskCurrentBatch(task.CurrentBatch),
		Channels:         cloneAIRouteTaskChannels(task.Channels),
		Result:           cloneAIRouteTaskResult(task.Result),
	}
}

func cloneAIRouteTaskSummary(summary *GenerateAIRouteProgressSummary) *GenerateAIRouteProgressSummary {
	if summary == nil {
		return nil
	}

	cloned := *summary
	return &cloned
}

func cloneAIRouteTaskCurrentBatch(batch *GenerateAIRouteCurrentBatch) *GenerateAIRouteCurrentBatch {
	if batch == nil {
		return nil
	}

	cloned := *batch
	if len(batch.ChannelIDs) > 0 {
		cloned.ChannelIDs = append([]int(nil), batch.ChannelIDs...)
	}
	if len(batch.ChannelNames) > 0 {
		cloned.ChannelNames = append([]string(nil), batch.ChannelNames...)
	}
	return &cloned
}

func cloneAIRouteTaskChannels(channels []GenerateAIRouteChannelProgress) []GenerateAIRouteChannelProgress {
	if len(channels) == 0 {
		return nil
	}

	cloned := make([]GenerateAIRouteChannelProgress, len(channels))
	copy(cloned, channels)
	return cloned
}

func cloneAIRouteTaskResult(result *GenerateAIRouteResult) *GenerateAIRouteResult {
	if result == nil {
		return nil
	}

	cloned := *result
	return &cloned
}

func cloneAIRouteTaskTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}
