package helper

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/utils/log"
)

const aiRouteTaskTimeout = 30 * time.Minute
const (
	aiRouteProgressPendingTTL      = aiRouteTaskTimeout + 5*time.Minute
	aiRouteProgressDoneTTL         = 10 * time.Minute
	aiRouteProgressHeartbeatPeriod = 2 * time.Second
	aiRouteProgressPersistenceTTL  = 5 * time.Second
)

var aiRouteProgress sync.Map
var aiRouteProgressNow = time.Now

type aiRouteProgressEntry struct {
	progress  model.GenerateAIRouteProgress
	expiresAt time.Time
}

func StartGenerateAIRoute(req model.GenerateAIRouteRequest) (*model.GenerateAIRouteProgress, error) {
	existingProgress, err := findActiveAIRouteProgress(req)
	if err != nil {
		return nil, err
	}
	if existingProgress != nil {
		storeAIRouteProgress(existingProgress)
		cloned := cloneAIRouteProgress(existingProgress)
		return &cloned, nil
	}

	id := uuid.NewString()
	now := aiRouteProgressNow()
	progress := &model.GenerateAIRouteProgress{
		ID:              id,
		Scope:           req.Scope,
		GroupID:         req.GroupID,
		Status:          model.AIRouteTaskStatusQueued,
		CurrentStep:     model.AIRouteTaskStepQueued,
		ProgressPercent: 0,
		Message:         "AI 路由任务已创建，等待开始",
		StartedAt:       cloneTimePtr(&now),
		UpdatedAt:       cloneTimePtr(&now),
		HeartbeatAt:     cloneTimePtr(&now),
		EventSequence:   1,
	}
	storeAIRouteProgress(progress)
	if err := persistAIRouteProgress(progress, true); err != nil {
		aiRouteProgress.Delete(id)
		return nil, err
	}
	publishGenerateAIRouteProgress(progress)

	var mu sync.Mutex
	stopHeartbeat := make(chan struct{})
	heartbeatDone := make(chan struct{})

	go func() {
		ticker := time.NewTicker(aiRouteProgressHeartbeatPeriod)
		defer func() {
			ticker.Stop()
			close(heartbeatDone)
		}()

		for {
			select {
			case <-stopHeartbeat:
				return
			case <-ticker.C:
				mu.Lock()
				if progress.Done {
					mu.Unlock()
					return
				}
				markAIRouteProgressHeartbeat(progress)
				storeAIRouteProgress(progress)
				if err := persistAIRouteProgress(progress, false); err != nil {
					log.Warnf("ai route heartbeat progress persistence failed: task_id=%s err=%v", progress.ID, err)
				} else {
					publishGenerateAIRouteProgress(progress)
				}
				mu.Unlock()
			}
		}
	}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), aiRouteTaskTimeout)
		defer cancel()

		report := func(snapshot model.GenerateAIRouteProgress) {
			mu.Lock()
			defer mu.Unlock()

			mergeAIRouteProgressSnapshot(progress, snapshot)
			markAIRouteProgressUpdated(progress)
			storeAIRouteProgress(progress)
			if err := persistAIRouteProgress(progress, false); err != nil {
				log.Warnf("ai route progress persistence failed: task_id=%s err=%v", progress.ID, err)
			} else {
				publishGenerateAIRouteProgress(progress)
			}
		}

		result, err := op.GenerateAIRoute(ctx, req, report)
		close(stopHeartbeat)
		<-heartbeatDone

		mu.Lock()
		defer mu.Unlock()

		finalizeAIRouteProgress(progress, result, err, ctx.Err())
		storeAIRouteProgress(progress)
		if persistErr := persistAIRouteProgress(progress, false); persistErr != nil {
			log.Warnf("ai route final progress persistence failed: task_id=%s err=%v", progress.ID, persistErr)
		} else {
			publishGenerateAIRouteProgress(progress)
		}
	}()

	cloned := cloneAIRouteProgress(progress)
	return &cloned, nil
}

func GetGenerateAIRouteProgress(id string) (*model.GenerateAIRouteProgress, bool) {
	if id == "" {
		return nil, false
	}

	value, ok := aiRouteProgress.Load(id)
	if ok {
		entry, ok := value.(aiRouteProgressEntry)
		if !ok {
			aiRouteProgress.Delete(id)
		} else {
			now := aiRouteProgressNow()
			if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
				aiRouteProgress.Delete(id)
			} else {
				cloned := cloneAIRouteProgress(&entry.progress)
				return &cloned, true
			}
		}
	}

	progress, err := loadAIRouteProgress(id)
	if err != nil {
		log.Warnf("ai route progress load failed: task_id=%s err=%v", id, err)
		return nil, false
	}
	if progress == nil {
		return nil, false
	}

	storeAIRouteProgress(progress)
	cloned := cloneAIRouteProgress(progress)
	return &cloned, true
}

func storeAIRouteProgress(progress *model.GenerateAIRouteProgress) {
	if progress == nil || progress.ID == "" {
		return
	}

	now := aiRouteProgressNow()
	cleanupExpiredAIRouteProgress(now)

	ttl := aiRouteProgressPendingTTL
	if progress.Done {
		ttl = aiRouteProgressDoneTTL
	}

	aiRouteProgress.Store(progress.ID, aiRouteProgressEntry{
		progress:  cloneAIRouteProgress(progress),
		expiresAt: now.Add(ttl),
	})
}

func cleanupExpiredAIRouteProgress(now time.Time) {
	aiRouteProgress.Range(func(key, value any) bool {
		id, ok := key.(string)
		if !ok {
			return true
		}

		entry, ok := value.(aiRouteProgressEntry)
		if !ok {
			aiRouteProgress.Delete(id)
			return true
		}
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			aiRouteProgress.Delete(id)
		}
		return true
	})
}

func cloneAIRouteProgress(progress *model.GenerateAIRouteProgress) model.GenerateAIRouteProgress {
	if progress == nil {
		return model.GenerateAIRouteProgress{}
	}

	cloned := *progress
	cloned.StartedAt = cloneTimePtr(progress.StartedAt)
	cloned.UpdatedAt = cloneTimePtr(progress.UpdatedAt)
	cloned.HeartbeatAt = cloneTimePtr(progress.HeartbeatAt)
	cloned.FinishedAt = cloneTimePtr(progress.FinishedAt)
	cloned.Summary = cloneAIRouteProgressSummary(progress.Summary)
	cloned.CurrentBatch = cloneAIRouteCurrentBatch(progress.CurrentBatch)
	cloned.RunningBatches = cloneAIRouteRunningBatchList(progress.RunningBatches)
	cloned.Channels = cloneAIRouteChannelProgressList(progress.Channels)
	cloned.Result = cloneAIRouteResult(progress.Result)
	return cloned
}

func cloneAIRouteResult(result *model.GenerateAIRouteResult) *model.GenerateAIRouteResult {
	if result == nil {
		return nil
	}

	cloned := *result
	return &cloned
}

func cloneAIRouteProgressSummary(summary *model.GenerateAIRouteProgressSummary) *model.GenerateAIRouteProgressSummary {
	if summary == nil {
		return nil
	}

	cloned := *summary
	return &cloned
}

func cloneAIRouteCurrentBatch(batch *model.GenerateAIRouteCurrentBatch) *model.GenerateAIRouteCurrentBatch {
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

func cloneAIRouteRunningBatchList(batches []model.GenerateAIRouteRunningBatch) []model.GenerateAIRouteRunningBatch {
	if len(batches) == 0 {
		return nil
	}

	cloned := make([]model.GenerateAIRouteRunningBatch, len(batches))
	for i := range batches {
		cloned[i] = batches[i]
		if len(batches[i].ChannelIDs) > 0 {
			cloned[i].ChannelIDs = append([]int(nil), batches[i].ChannelIDs...)
		}
		if len(batches[i].ChannelNames) > 0 {
			cloned[i].ChannelNames = append([]string(nil), batches[i].ChannelNames...)
		}
	}
	return cloned
}

func cloneAIRouteChannelProgressList(channels []model.GenerateAIRouteChannelProgress) []model.GenerateAIRouteChannelProgress {
	if len(channels) == 0 {
		return nil
	}

	cloned := make([]model.GenerateAIRouteChannelProgress, len(channels))
	copy(cloned, channels)
	return cloned
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func mergeAIRouteProgressSnapshot(dst *model.GenerateAIRouteProgress, snapshot model.GenerateAIRouteProgress) {
	if dst == nil {
		return
	}

	if snapshot.Scope != "" {
		dst.Scope = snapshot.Scope
	}
	dst.GroupID = snapshot.GroupID
	if snapshot.Status != "" {
		dst.Status = snapshot.Status
	}
	if snapshot.CurrentStep != "" {
		dst.CurrentStep = snapshot.CurrentStep
	}
	dst.ProgressPercent = snapshot.ProgressPercent
	dst.TotalBatches = snapshot.TotalBatches
	dst.CompletedBatches = snapshot.CompletedBatches
	dst.Done = snapshot.Done
	dst.ResultReady = snapshot.ResultReady
	dst.Message = snapshot.Message
	dst.ErrorReason = snapshot.ErrorReason
	dst.Summary = cloneAIRouteProgressSummary(snapshot.Summary)
	dst.CurrentBatch = cloneAIRouteCurrentBatch(snapshot.CurrentBatch)
	dst.RunningBatches = cloneAIRouteRunningBatchList(snapshot.RunningBatches)
	dst.Channels = cloneAIRouteChannelProgressList(snapshot.Channels)
	dst.Result = cloneAIRouteResult(snapshot.Result)
}

func markAIRouteProgressUpdated(progress *model.GenerateAIRouteProgress) {
	if progress == nil {
		return
	}

	now := aiRouteProgressNow()
	progress.UpdatedAt = cloneTimePtr(&now)
	progress.HeartbeatAt = cloneTimePtr(&now)
	progress.EventSequence++
}

func markAIRouteProgressHeartbeat(progress *model.GenerateAIRouteProgress) {
	if progress == nil {
		return
	}

	now := aiRouteProgressNow()
	progress.HeartbeatAt = cloneTimePtr(&now)
	progress.EventSequence++
}

func finalizeAIRouteProgress(
	progress *model.GenerateAIRouteProgress,
	result *model.GenerateAIRouteResult,
	runErr error,
	ctxErr error,
) {
	if progress == nil {
		return
	}

	now := aiRouteProgressNow()
	progress.Done = true
	progress.Result = cloneAIRouteResult(result)
	progress.FinishedAt = cloneTimePtr(&now)
	progress.UpdatedAt = cloneTimePtr(&now)
	progress.HeartbeatAt = cloneTimePtr(&now)
	progress.EventSequence++
	progress.ErrorReason = ""
	progress.ResultReady = false

	if result != nil {
		progress.Scope = result.Scope
		progress.GroupID = result.GroupID
	}

	if runErr != nil {
		progress.ProgressPercent = minInt(progress.ProgressPercent, 99)
		var partialErr *op.AIRoutePartialFailureError
		if errors.As(runErr, &partialErr) {
			progress.Status = model.AIRouteTaskStatusFailed
			progress.CurrentStep = model.AIRouteTaskStepFailed
			progress.Message = partialErr.Error()
		} else if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(ctxErr, context.DeadlineExceeded) {
			progress.Status = model.AIRouteTaskStatusTimeout
			progress.CurrentStep = model.AIRouteTaskStepTimeout
			progress.Message = "AI 路由任务超时，请稍后重试"
		} else {
			progress.Status = model.AIRouteTaskStatusFailed
			progress.CurrentStep = model.AIRouteTaskStepFailed
			progress.Message = runErr.Error()
		}
		progress.ErrorReason = progress.Message
		progress.ResultReady = progress.Result != nil
		markRunningAIRouteChannelsFailed(progress, progress.Message)
		markRunningAIRouteBatchesFailed(progress, progress.Message)
		return
	}

	progress.Status = model.AIRouteTaskStatusCompleted
	progress.CurrentStep = model.AIRouteTaskStepCompleted
	progress.ProgressPercent = 100
	progress.CurrentBatch = nil
	progress.RunningBatches = nil
	progress.Message = "AI 路由生成完成"
	progress.ResultReady = result != nil
}

func markRunningAIRouteChannelsFailed(progress *model.GenerateAIRouteProgress, message string) {
	if progress == nil {
		return
	}

	for i := range progress.Channels {
		if progress.Channels[i].Status != model.AIRouteChannelStatusRunning {
			continue
		}
		progress.Channels[i].Status = model.AIRouteChannelStatusFailed
		if progress.Channels[i].Message == "" {
			progress.Channels[i].Message = message
		}
	}

	if progress.Summary == nil {
		return
	}

	progress.Summary.RunningChannels = 0
	progress.Summary.FailedChannels = 0
	progress.Summary.PendingChannels = 0
	progress.Summary.CompletedChannels = 0

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

func markRunningAIRouteBatchesFailed(progress *model.GenerateAIRouteProgress, message string) {
	if progress == nil {
		return
	}

	if progress.CurrentBatch != nil {
		progress.CurrentBatch.Status = "failed"
		if strings.TrimSpace(progress.CurrentBatch.Message) == "" {
			progress.CurrentBatch.Message = message
		}
	}

	if len(progress.RunningBatches) == 0 {
		return
	}

	for i := range progress.RunningBatches {
		progress.RunningBatches[i].Status = model.AIRouteBatchStatusFailed
		if strings.TrimSpace(progress.RunningBatches[i].Message) == "" {
			progress.RunningBatches[i].Message = message
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func findActiveAIRouteProgress(req model.GenerateAIRouteRequest) (*model.GenerateAIRouteProgress, error) {
	ctx, cancel := context.WithTimeout(context.Background(), aiRouteProgressPersistenceTTL)
	defer cancel()

	return op.AIRouteTaskFindActive(ctx, req.Scope, req.GroupID)
}

func loadAIRouteProgress(id string) (*model.GenerateAIRouteProgress, error) {
	ctx, cancel := context.WithTimeout(context.Background(), aiRouteProgressPersistenceTTL)
	defer cancel()

	return op.AIRouteTaskGet(ctx, id)
}

func persistAIRouteProgress(progress *model.GenerateAIRouteProgress, create bool) error {
	if progress == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), aiRouteProgressPersistenceTTL)
	defer cancel()

	snapshot := cloneAIRouteProgress(progress)
	if create {
		return op.AIRouteTaskCreate(ctx, snapshot)
	}
	return op.AIRouteTaskSaveProgress(ctx, snapshot)
}
