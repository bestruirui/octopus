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
	MessageKey       string                           `json:"message_key" gorm:"type:text"`
	MessageArgs      map[string]any                   `json:"message_args,omitempty" gorm:"serializer:json"`
	ErrorReason      string                           `json:"error_reason" gorm:"type:text"`
	ErrorReasonKey   string                           `json:"error_reason_key" gorm:"type:text"`
	ErrorReasonArgs  map[string]any                   `json:"error_reason_args,omitempty" gorm:"serializer:json"`
	StartedAt        *time.Time                       `json:"started_at,omitempty"`
	UpdatedAt        *time.Time                       `json:"updated_at,omitempty"`
	HeartbeatAt      *time.Time                       `json:"heartbeat_at,omitempty;index"`
	FinishedAt       *time.Time                       `json:"finished_at,omitempty"`
	EventSequence    int64                            `json:"event_sequence" gorm:"not null;default:0"`
	Summary          *GenerateAIRouteProgressSummary  `json:"summary,omitempty" gorm:"serializer:json"`
	CurrentBatch     *GenerateAIRouteCurrentBatch     `json:"current_batch,omitempty" gorm:"serializer:json"`
	RunningBatches   []GenerateAIRouteRunningBatch    `json:"running_batches,omitempty" gorm:"serializer:json"`
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
		MessageKey:       progress.MessageKey,
		MessageArgs:      cloneAIRouteTaskArgs(progress.MessageArgs),
		ErrorReason:      progress.ErrorReason,
		ErrorReasonKey:   progress.ErrorReasonKey,
		ErrorReasonArgs:  cloneAIRouteTaskArgs(progress.ErrorReasonArgs),
		StartedAt:        cloneAIRouteTaskTime(progress.StartedAt),
		UpdatedAt:        cloneAIRouteTaskTime(progress.UpdatedAt),
		HeartbeatAt:      cloneAIRouteTaskTime(progress.HeartbeatAt),
		FinishedAt:       cloneAIRouteTaskTime(progress.FinishedAt),
		EventSequence:    progress.EventSequence,
		Summary:          cloneAIRouteTaskSummary(progress.Summary),
		CurrentBatch:     cloneAIRouteTaskCurrentBatch(progress.CurrentBatch),
		RunningBatches:   cloneAIRouteTaskRunningBatches(progress.RunningBatches),
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
		MessageKey:       task.MessageKey,
		MessageArgs:      cloneAIRouteTaskArgs(task.MessageArgs),
		ErrorReason:      task.ErrorReason,
		ErrorReasonKey:   task.ErrorReasonKey,
		ErrorReasonArgs:  cloneAIRouteTaskArgs(task.ErrorReasonArgs),
		StartedAt:        cloneAIRouteTaskTime(task.StartedAt),
		UpdatedAt:        cloneAIRouteTaskTime(task.UpdatedAt),
		HeartbeatAt:      cloneAIRouteTaskTime(task.HeartbeatAt),
		FinishedAt:       cloneAIRouteTaskTime(task.FinishedAt),
		EventSequence:    task.EventSequence,
		Summary:          cloneAIRouteTaskSummary(task.Summary),
		CurrentBatch:     cloneAIRouteTaskCurrentBatch(task.CurrentBatch),
		RunningBatches:   cloneAIRouteTaskRunningBatches(task.RunningBatches),
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

func cloneAIRouteTaskArgs(args map[string]any) map[string]any {
	if len(args) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(args))
	for key, value := range args {
		cloned[key] = value
	}
	return cloned
}

func cloneAIRouteTaskCurrentBatch(batch *GenerateAIRouteCurrentBatch) *GenerateAIRouteCurrentBatch {
	if batch == nil {
		return nil
	}

	cloned := *batch
	cloned.MessageArgs = cloneAIRouteTaskArgs(batch.MessageArgs)
	if len(batch.ChannelIDs) > 0 {
		cloned.ChannelIDs = append([]int(nil), batch.ChannelIDs...)
	}
	if len(batch.ChannelNames) > 0 {
		cloned.ChannelNames = append([]string(nil), batch.ChannelNames...)
	}
	return &cloned
}

func cloneAIRouteTaskRunningBatches(batches []GenerateAIRouteRunningBatch) []GenerateAIRouteRunningBatch {
	if len(batches) == 0 {
		return nil
	}

	cloned := make([]GenerateAIRouteRunningBatch, len(batches))
	for i := range batches {
		cloned[i] = batches[i]
		cloned[i].MessageArgs = cloneAIRouteTaskArgs(batches[i].MessageArgs)
		if len(batches[i].ChannelIDs) > 0 {
			cloned[i].ChannelIDs = append([]int(nil), batches[i].ChannelIDs...)
		}
		if len(batches[i].ChannelNames) > 0 {
			cloned[i].ChannelNames = append([]string(nil), batches[i].ChannelNames...)
		}
	}
	return cloned
}

func cloneAIRouteTaskChannels(channels []GenerateAIRouteChannelProgress) []GenerateAIRouteChannelProgress {
	if len(channels) == 0 {
		return nil
	}

	cloned := make([]GenerateAIRouteChannelProgress, len(channels))
	for i := range channels {
		cloned[i] = channels[i]
		cloned[i].MessageArgs = cloneAIRouteTaskArgs(channels[i].MessageArgs)
	}
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
