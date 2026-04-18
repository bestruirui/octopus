package op

import (
	"fmt"
	"strings"

	"github.com/lingyuins/octopus/internal/model"
)

type aiRouteProgressReporter func(model.GenerateAIRouteProgress)

type aiRouteProgressTracker struct {
	report           aiRouteProgressReporter
	progress         model.GenerateAIRouteProgress
	channelIndexByID map[int]int
	totalModels      int
	completedModels  int
}

func newAIRouteProgressTracker(req model.GenerateAIRouteRequest, report aiRouteProgressReporter) *aiRouteProgressTracker {
	tracker := &aiRouteProgressTracker{
		report: report,
		progress: model.GenerateAIRouteProgress{
			Scope:           req.Scope,
			GroupID:         req.GroupID,
			Status:          model.AIRouteTaskStatusRunning,
			CurrentStep:     model.AIRouteTaskStepCollectingModels,
			ProgressPercent: 3,
			Message:         "正在收集渠道和模型",
		},
		channelIndexByID: make(map[int]int),
	}
	tracker.emit()
	return tracker
}

func (t *aiRouteProgressTracker) SetModelInputs(modelInputs []model.AIRouteModelInput) {
	if t == nil {
		return
	}

	channels := make([]model.GenerateAIRouteChannelProgress, 0)
	channelIndexByID := make(map[int]int)
	totalModels := 0

	for _, input := range modelInputs {
		modelName := strings.TrimSpace(input.Model)
		if input.ChannelID <= 0 || modelName == "" {
			continue
		}

		index, ok := channelIndexByID[input.ChannelID]
		if !ok {
			index = len(channels)
			channelIndexByID[input.ChannelID] = index
			channels = append(channels, model.GenerateAIRouteChannelProgress{
				ChannelID:   input.ChannelID,
				ChannelName: strings.TrimSpace(input.ChannelName),
				Provider:    strings.TrimSpace(input.Provider),
				Status:      model.AIRouteChannelStatusPending,
			})
		}

		channels[index].TotalModels++
		totalModels++
	}

	t.channelIndexByID = channelIndexByID
	t.totalModels = totalModels
	t.completedModels = 0
	t.progress.Channels = channels
	t.progress.Summary = t.buildSummary()
	t.progress.ProgressPercent = 10
	t.progress.Message = fmt.Sprintf("已收集 %d 个渠道，共 %d 个模型", len(channels), totalModels)
	t.emit()
}

func (t *aiRouteProgressTracker) SetTargetGroup(groupID int) {
	if t == nil {
		return
	}

	t.progress.GroupID = groupID
}

func (t *aiRouteProgressTracker) SetBuckets(buckets []aiRoutePromptBucket) {
	if t == nil {
		return
	}

	t.progress.CurrentStep = model.AIRouteTaskStepBuildingBatches
	t.progress.TotalBatches = len(buckets)
	t.progress.CompletedBatches = 0
	t.progress.CurrentBatch = nil
	t.progress.ProgressPercent = 15
	if len(buckets) <= 1 {
		t.progress.Message = "已完成批次规划，准备开始 AI 分析"
	} else {
		t.progress.Message = fmt.Sprintf("已完成批次规划，共拆分为 %d 批", len(buckets))
	}
	t.emit()
}

func (t *aiRouteProgressTracker) StartBatch(index int, bucket aiRoutePromptBucket) {
	if t == nil {
		return
	}

	currentBatch := t.buildCurrentBatch(index, bucket)
	t.progress.CurrentStep = model.AIRouteTaskStepAnalyzingBatches
	t.progress.CurrentBatch = currentBatch
	t.progress.ProgressPercent = t.analysisProgress(currentBatch.ModelCount, 0.1)
	t.progress.Message = fmt.Sprintf(
		"正在等待 AI 返回第 %d/%d 批结果（%s，涉及 %d 个渠道 / %d 个模型）",
		currentBatch.Index,
		currentBatch.Total,
		airoutePromptEndpointLabel(currentBatch.EndpointType),
		len(currentBatch.ChannelIDs),
		currentBatch.ModelCount,
	)

	for _, channelID := range currentBatch.ChannelIDs {
		channel := t.channelByID(channelID)
		if channel == nil || channel.Status == model.AIRouteChannelStatusCompleted {
			continue
		}
		channel.Status = model.AIRouteChannelStatusRunning
		channel.Message = fmt.Sprintf("正在分析 %s 模型", airoutePromptEndpointLabel(bucket.PromptEndpointType))
	}

	t.progress.Summary = t.buildSummary()
	t.emit()
}

func (t *aiRouteProgressTracker) MarkBatchAIResponseReceived(index int, bucket aiRoutePromptBucket) {
	if t == nil {
		return
	}

	currentBatch := t.buildCurrentBatch(index, bucket)
	t.progress.CurrentStep = model.AIRouteTaskStepParsingResponse
	t.progress.CurrentBatch = currentBatch
	t.progress.ProgressPercent = t.analysisProgress(currentBatch.ModelCount, 0.72)
	t.progress.Message = fmt.Sprintf("AI 已返回第 %d/%d 批结果，正在解析和校验", currentBatch.Index, currentBatch.Total)
	t.emit()
}

func (t *aiRouteProgressTracker) CompleteBatch(index int, bucket aiRoutePromptBucket) {
	if t == nil {
		return
	}

	currentBatch := t.buildCurrentBatch(index, bucket)
	for _, input := range bucket.ModelInputs {
		channel := t.channelByID(input.ChannelID)
		if channel == nil {
			continue
		}
		if channel.ProcessedModels < channel.TotalModels {
			channel.ProcessedModels++
			t.completedModels++
		}
		if channel.ProcessedModels >= channel.TotalModels {
			channel.Status = model.AIRouteChannelStatusCompleted
			channel.Message = "已完成"
		} else {
			channel.Status = model.AIRouteChannelStatusPending
			channel.Message = ""
		}
	}

	t.progress.CompletedBatches++
	t.progress.ProgressPercent = t.analysisProgress(0, 0)
	t.progress.Summary = t.buildSummary()
	t.progress.CurrentBatch = currentBatch
	if t.progress.CompletedBatches < t.progress.TotalBatches {
		t.progress.CurrentStep = model.AIRouteTaskStepAnalyzingBatches
		t.progress.Message = fmt.Sprintf("第 %d/%d 批已完成，准备继续下一批", currentBatch.Index, currentBatch.Total)
	} else {
		t.progress.CurrentStep = model.AIRouteTaskStepParsingResponse
		t.progress.Message = "全部批次分析完成，准备校验路由结果"
	}
	t.emit()
}

func (t *aiRouteProgressTracker) SetValidatingRoutes(routeCount int) {
	if t == nil {
		return
	}

	t.progress.CurrentStep = model.AIRouteTaskStepValidatingRoutes
	t.progress.CurrentBatch = nil
	t.progress.ProgressPercent = 88
	if routeCount > 0 {
		t.progress.Message = fmt.Sprintf("正在校验 AI 返回的 %d 条候选路由", routeCount)
	} else {
		t.progress.Message = "正在校验 AI 返回的候选路由"
	}
	t.emit()
}

func (t *aiRouteProgressTracker) SetWritingGroup(groupName string) {
	if t == nil {
		return
	}

	groupName = strings.TrimSpace(groupName)
	t.progress.CurrentStep = model.AIRouteTaskStepWritingGroups
	t.progress.CurrentBatch = nil
	t.progress.ProgressPercent = 94
	if groupName == "" {
		t.progress.Message = "正在写入当前分组"
	} else {
		t.progress.Message = fmt.Sprintf("正在写入分组 %q", groupName)
	}
	t.emit()
}

func (t *aiRouteProgressTracker) SetWritingRoute(index int, total int, requestedModel string) {
	if t == nil {
		return
	}

	if total <= 0 {
		total = 1
	}
	requestedModel = strings.TrimSpace(requestedModel)

	t.progress.CurrentStep = model.AIRouteTaskStepWritingGroups
	t.progress.CurrentBatch = nil
	t.progress.ProgressPercent = 90 + int(float64(index)/float64(total)*8)
	if t.progress.ProgressPercent > 98 {
		t.progress.ProgressPercent = 98
	}

	if requestedModel == "" {
		t.progress.Message = fmt.Sprintf("正在写入第 %d/%d 条路由", index, total)
	} else {
		t.progress.Message = fmt.Sprintf("正在写入路由 %q（%d/%d）", requestedModel, index, total)
	}
	t.emit()
}

func (t *aiRouteProgressTracker) SetFinalizing(message string) {
	if t == nil {
		return
	}

	t.progress.CurrentStep = model.AIRouteTaskStepFinalizing
	t.progress.CurrentBatch = nil
	t.progress.ProgressPercent = 99
	if strings.TrimSpace(message) == "" {
		t.progress.Message = "正在收尾"
	} else {
		t.progress.Message = strings.TrimSpace(message)
	}
	t.emit()
}

func (t *aiRouteProgressTracker) analysisProgress(currentBatchModels int, batchFraction float64) int {
	if t == nil || t.totalModels <= 0 {
		return 20
	}

	progressModels := float64(t.completedModels) + float64(currentBatchModels)*batchFraction
	percent := 20 + int(progressModels/float64(t.totalModels)*60)
	if percent < 20 {
		return 20
	}
	if percent > 80 {
		return 80
	}
	return percent
}

func (t *aiRouteProgressTracker) buildSummary() *model.GenerateAIRouteProgressSummary {
	summary := &model.GenerateAIRouteProgressSummary{
		TotalChannels: len(t.progress.Channels),
		TotalModels:   t.totalModels,
	}

	for _, channel := range t.progress.Channels {
		summary.CompletedModels += channel.ProcessedModels
		switch channel.Status {
		case model.AIRouteChannelStatusCompleted:
			summary.CompletedChannels++
		case model.AIRouteChannelStatusRunning:
			summary.RunningChannels++
		case model.AIRouteChannelStatusFailed:
			summary.FailedChannels++
		default:
			summary.PendingChannels++
		}
	}

	return summary
}

func (t *aiRouteProgressTracker) buildCurrentBatch(index int, bucket aiRoutePromptBucket) *model.GenerateAIRouteCurrentBatch {
	if t == nil {
		return nil
	}

	channelIDs := make([]int, 0)
	channelNames := make([]string, 0)
	seenChannels := make(map[int]struct{})

	for _, input := range bucket.ModelInputs {
		if _, ok := seenChannels[input.ChannelID]; ok {
			continue
		}
		seenChannels[input.ChannelID] = struct{}{}
		channelIDs = append(channelIDs, input.ChannelID)
		channelNames = append(channelNames, t.channelNameByID(input.ChannelID))
	}

	return &model.GenerateAIRouteCurrentBatch{
		Index:        index,
		Total:        t.progress.TotalBatches,
		EndpointType: bucket.PromptEndpointType,
		ModelCount:   len(bucket.ModelInputs),
		ChannelIDs:   channelIDs,
		ChannelNames: channelNames,
	}
}

func (t *aiRouteProgressTracker) channelByID(channelID int) *model.GenerateAIRouteChannelProgress {
	if t == nil {
		return nil
	}

	index, ok := t.channelIndexByID[channelID]
	if !ok || index < 0 || index >= len(t.progress.Channels) {
		return nil
	}
	return &t.progress.Channels[index]
}

func (t *aiRouteProgressTracker) channelNameByID(channelID int) string {
	channel := t.channelByID(channelID)
	if channel == nil {
		return fmt.Sprintf("Channel %d", channelID)
	}
	if name := strings.TrimSpace(channel.ChannelName); name != "" {
		return name
	}
	return fmt.Sprintf("Channel %d", channelID)
}

func (t *aiRouteProgressTracker) emit() {
	if t == nil || t.report == nil {
		return
	}

	t.progress.Summary = t.buildSummary()
	t.report(cloneAIRouteProgressSnapshot(t.progress))
}

func cloneAIRouteProgressSnapshot(progress model.GenerateAIRouteProgress) model.GenerateAIRouteProgress {
	cloned := progress
	cloned.Summary = cloneAIRouteProgressSummary(progress.Summary)
	cloned.CurrentBatch = cloneAIRouteCurrentBatch(progress.CurrentBatch)
	cloned.Channels = cloneAIRouteChannelProgressList(progress.Channels)
	cloned.Result = cloneAIRouteResult(progress.Result)
	return cloned
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

func cloneAIRouteChannelProgressList(channels []model.GenerateAIRouteChannelProgress) []model.GenerateAIRouteChannelProgress {
	if len(channels) == 0 {
		return nil
	}

	cloned := make([]model.GenerateAIRouteChannelProgress, len(channels))
	copy(cloned, channels)
	return cloned
}

func cloneAIRouteResult(result *model.GenerateAIRouteResult) *model.GenerateAIRouteResult {
	if result == nil {
		return nil
	}

	cloned := *result
	return &cloned
}
