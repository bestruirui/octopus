package op

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"gorm.io/gorm"
)

const DefaultAIRouteTaskInterruptedMessage = "AI 路由任务因服务重启或中断而终止，请重新发起"

func AIRouteTaskCreate(ctx context.Context, progress model.GenerateAIRouteProgress) error {
	if db.GetDB() == nil {
		return fmt.Errorf("db is nil")
	}

	task := model.NewAIRouteTask(progress)
	return db.GetDB().WithContext(ctx).Create(&task).Error
}

func AIRouteTaskSaveProgress(ctx context.Context, progress model.GenerateAIRouteProgress) error {
	if db.GetDB() == nil {
		return fmt.Errorf("db is nil")
	}

	task := model.NewAIRouteTask(progress)
	return db.GetDB().WithContext(ctx).Save(&task).Error
}

func AIRouteTaskGet(ctx context.Context, id string) (*model.GenerateAIRouteProgress, error) {
	if strings.TrimSpace(id) == "" {
		return nil, nil
	}
	if db.GetDB() == nil {
		return nil, nil
	}

	var task model.AIRouteTask
	if err := db.GetDB().WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	progress := task.ToProgress()
	return &progress, nil
}

func AIRouteTaskFindActive(ctx context.Context, scope model.AIRouteScope, groupID int) (*model.GenerateAIRouteProgress, error) {
	if db.GetDB() == nil {
		return nil, nil
	}

	var task model.AIRouteTask
	query := db.GetDB().WithContext(ctx).
		Where("scope = ? AND group_id = ? AND done = ?", scope, groupID, false).
		Order("heartbeat_at DESC").
		Order("updated_at DESC")

	if err := query.First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	progress := task.ToProgress()
	return &progress, nil
}

func AIRouteTaskMarkActiveInterrupted(ctx context.Context, message string) (int64, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		message = DefaultAIRouteTaskInterruptedMessage
	}
	if db.GetDB() == nil {
		return 0, nil
	}

	tasks := make([]model.AIRouteTask, 0)
	if err := db.GetDB().WithContext(ctx).Where("done = ?", false).Find(&tasks).Error; err != nil {
		return 0, err
	}

	if len(tasks) == 0 {
		return 0, nil
	}

	updated := int64(0)
	for _, task := range tasks {
		progress := task.ToProgress()
		interruptAIRouteTaskProgress(&progress, message)
		if err := AIRouteTaskSaveProgress(ctx, progress); err != nil {
			return updated, err
		}
		updated++
	}

	return updated, nil
}

func interruptAIRouteTaskProgress(progress *model.GenerateAIRouteProgress, message string) {
	if progress == nil {
		return
	}

	now := time.Now()
	progress.Status = model.AIRouteTaskStatusFailed
	progress.CurrentStep = model.AIRouteTaskStepFailed
	progress.Done = true
	progress.ResultReady = false
	progress.Result = nil
	progress.Message = message
	progress.ErrorReason = message
	progress.UpdatedAt = cloneAIRouteTaskTimePtr(&now)
	progress.HeartbeatAt = cloneAIRouteTaskTimePtr(&now)
	progress.FinishedAt = cloneAIRouteTaskTimePtr(&now)
	progress.EventSequence++
	if progress.ProgressPercent > 99 {
		progress.ProgressPercent = 99
	}
	if progress.CurrentBatch != nil {
		progress.CurrentBatch.Status = "failed"
		if strings.TrimSpace(progress.CurrentBatch.Message) == "" {
			progress.CurrentBatch.Message = message
		}
	}
	for i := range progress.RunningBatches {
		progress.RunningBatches[i].Status = model.AIRouteBatchStatusFailed
		if strings.TrimSpace(progress.RunningBatches[i].Message) == "" {
			progress.RunningBatches[i].Message = message
		}
	}

	for i := range progress.Channels {
		if progress.Channels[i].Status != model.AIRouteChannelStatusRunning {
			continue
		}
		progress.Channels[i].Status = model.AIRouteChannelStatusFailed
		if strings.TrimSpace(progress.Channels[i].Message) == "" {
			progress.Channels[i].Message = message
		}
	}

	if progress.Summary == nil {
		return
	}

	progress.Summary.CompletedChannels = 0
	progress.Summary.RunningChannels = 0
	progress.Summary.PendingChannels = 0
	progress.Summary.FailedChannels = 0
	for _, channel := range progress.Channels {
		switch channel.Status {
		case model.AIRouteChannelStatusCompleted:
			progress.Summary.CompletedChannels++
		case model.AIRouteChannelStatusFailed:
			progress.Summary.FailedChannels++
		case model.AIRouteChannelStatusRunning:
			progress.Summary.RunningChannels++
		default:
			progress.Summary.PendingChannels++
		}
	}
}

func cloneAIRouteTaskTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}
