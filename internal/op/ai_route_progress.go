package op

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/lingyuins/octopus/internal/model"
)

type aiRouteProgressReporter func(model.GenerateAIRouteProgress)

type aiRouteProgressTracker struct {
	mu                  sync.Mutex
	report              aiRouteProgressReporter
	progress            model.GenerateAIRouteProgress
	channelIndexByID    map[int]int
	channelActiveCounts map[int]int
	runningBatches      map[int]model.GenerateAIRouteRunningBatch
	runningBatchOrder   map[int]int64
	nextBatchOrder      int64
	totalModels         int
	completedModels     int
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
			MessageKey:      "group.aiRoute.progress.steps.collecting_models",
		},
		channelIndexByID:    make(map[int]int),
		channelActiveCounts: make(map[int]int),
		runningBatches:      make(map[int]model.GenerateAIRouteRunningBatch),
		runningBatchOrder:   make(map[int]int64),
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

	t.mu.Lock()
	t.channelIndexByID = channelIndexByID
	t.channelActiveCounts = make(map[int]int, len(channelIndexByID))
	t.totalModels = totalModels
	t.completedModels = 0
	t.progress.Channels = channels
	t.progress.Summary = t.buildSummaryLocked()
	t.progress.ProgressPercent = 10
	t.progress.Message = fmt.Sprintf("已收集 %d 个渠道，共 %d 个模型", len(channels), totalModels)
	t.progress.MessageKey = "group.aiRoute.progress.runtime.collectedModels"
	t.progress.MessageArgs = map[string]any{
		"channels": len(channels),
		"models":   totalModels,
	}
	t.mu.Unlock()

	t.emit()
}

func (t *aiRouteProgressTracker) SetTargetGroup(groupID int) {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.progress.GroupID = groupID
	t.mu.Unlock()
}

func (t *aiRouteProgressTracker) SetBuckets(buckets []aiRoutePromptBucket) {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.progress.CurrentStep = model.AIRouteTaskStepBuildingBatches
	t.progress.TotalBatches = len(buckets)
	t.progress.CompletedBatches = 0
	t.progress.CurrentBatch = nil
	t.progress.RunningBatches = nil
	t.progress.ProgressPercent = 15
	t.runningBatches = make(map[int]model.GenerateAIRouteRunningBatch, len(buckets))
	t.runningBatchOrder = make(map[int]int64, len(buckets))
	t.nextBatchOrder = 0
	if len(buckets) <= 1 {
		t.progress.Message = "已完成批次规划，准备开始 AI 分析"
		t.progress.MessageKey = "group.aiRoute.progress.runtime.plannedSingleBatch"
		t.progress.MessageArgs = nil
	} else {
		t.progress.Message = fmt.Sprintf("已完成批次规划，共拆分为 %d 批", len(buckets))
		t.progress.MessageKey = "group.aiRoute.progress.runtime.plannedBatches"
		t.progress.MessageArgs = map[string]any{"batches": len(buckets)}
	}
	t.mu.Unlock()

	t.emit()
}

func (t *aiRouteProgressTracker) StartBatch(index int, bucket aiRoutePromptBucket, serviceName string, attempt int) {
	if t == nil {
		return
	}

	t.mu.Lock()
	batch := t.buildRunningBatchLocked(index, bucket, serviceName, attempt, model.AIRouteBatchStatusRunning,
		fmt.Sprintf("正在使用 %s 分析第 %d/%d 批", fallbackAIRouteServiceName(serviceName), index, t.progress.TotalBatches))
	_, existed := t.runningBatches[index]
	t.upsertRunningBatchLocked(batch)
	if !existed {
		for _, channelID := range batch.ChannelIDs {
			t.incrementChannelActiveLocked(channelID, bucket.PromptEndpointType)
		}
	}

	t.progress.CurrentStep = model.AIRouteTaskStepAnalyzingBatches
	t.progress.CurrentBatch = t.currentBatchFromRunningLocked()
	t.progress.ProgressPercent = t.analysisProgressLocked()
	t.progress.Summary = t.buildSummaryLocked()
	t.progress.Message = t.buildAnalysisMessageLocked(batch)
	t.progress.MessageKey, t.progress.MessageArgs = t.buildAnalysisMessageMetaLocked(batch)
	t.mu.Unlock()

	t.emit()
}

func (t *aiRouteProgressTracker) MarkBatchRetry(index int, bucket aiRoutePromptBucket, serviceName string, attempt int, message string) {
	if t == nil {
		return
	}

	t.mu.Lock()
	batch := t.buildRunningBatchLocked(index, bucket, serviceName, attempt, model.AIRouteBatchStatusRetrying, strings.TrimSpace(message))
	t.upsertRunningBatchLocked(batch)
	t.progress.CurrentStep = model.AIRouteTaskStepAnalyzingBatches
	t.progress.CurrentBatch = t.currentBatchFromRunningLocked()
	t.progress.ProgressPercent = t.analysisProgressLocked()
	t.progress.Summary = t.buildSummaryLocked()
	if strings.TrimSpace(message) == "" {
		t.progress.Message = fmt.Sprintf("第 %d/%d 批分析失败，正在切换其他服务重试", index, t.progress.TotalBatches)
		t.progress.MessageKey = "group.aiRoute.progress.runtime.batchRetrying"
		t.progress.MessageArgs = map[string]any{"index": index, "total": t.progress.TotalBatches}
	} else {
		t.progress.Message = fmt.Sprintf("第 %d/%d 批分析失败，正在切换其他服务重试：%s", index, t.progress.TotalBatches, strings.TrimSpace(message))
		t.progress.MessageKey = "group.aiRoute.progress.runtime.batchRetryingWithReason"
		t.progress.MessageArgs = map[string]any{"index": index, "total": t.progress.TotalBatches, "reason": strings.TrimSpace(message)}
	}
	t.mu.Unlock()

	t.emit()
}

func (t *aiRouteProgressTracker) MarkBatchAIResponseReceived(index int, bucket aiRoutePromptBucket, serviceName string, attempt int) {
	if t == nil {
		return
	}

	t.mu.Lock()
	batch := t.buildRunningBatchLocked(index, bucket, serviceName, attempt, model.AIRouteBatchStatusParsing, "AI 已返回结果，正在解析和校验")
	t.upsertRunningBatchLocked(batch)
	t.progress.CurrentStep = model.AIRouteTaskStepParsingResponse
	t.progress.CurrentBatch = t.currentBatchFromRunningLocked()
	t.progress.ProgressPercent = t.analysisProgressLocked()
	t.progress.Summary = t.buildSummaryLocked()
	t.progress.Message = fmt.Sprintf("AI 已返回第 %d/%d 批结果，正在解析和校验", index, t.progress.TotalBatches)
	t.progress.MessageKey = "group.aiRoute.progress.runtime.batchParsing"
	t.progress.MessageArgs = map[string]any{"index": index, "total": t.progress.TotalBatches}
	t.mu.Unlock()

	t.emit()
}

func (t *aiRouteProgressTracker) CompleteBatch(index int, bucket aiRoutePromptBucket, serviceName string, attempt int) {
	if t == nil {
		return
	}

	t.mu.Lock()
	batch, existed := t.runningBatches[index]
	if !existed {
		batch = t.buildRunningBatchLocked(index, bucket, serviceName, attempt, model.AIRouteBatchStatusRunning, "")
	}
	for _, channelID := range batch.ChannelIDs {
		t.decrementChannelActiveLocked(channelID)
	}
	delete(t.runningBatches, index)
	delete(t.runningBatchOrder, index)
	t.syncRunningBatchesLocked()

	for _, input := range bucket.ModelInputs {
		channel := t.channelByIDLocked(input.ChannelID)
		if channel == nil {
			continue
		}
		if channel.ProcessedModels < channel.TotalModels {
			channel.ProcessedModels++
			t.completedModels++
		}
		t.refreshChannelStateLocked(channel)
	}

	t.progress.CompletedBatches++
	t.progress.ProgressPercent = t.analysisProgressLocked()
	t.progress.Summary = t.buildSummaryLocked()

	completedBatch := t.currentBatchFromRunningBatch(batch, "completed", "当前批次已完成")
	if activeBatch := t.currentBatchFromRunningLocked(); activeBatch != nil {
		t.progress.CurrentStep = model.AIRouteTaskStepAnalyzingBatches
		t.progress.CurrentBatch = activeBatch
		t.progress.Message = fmt.Sprintf("第 %d/%d 批已完成，当前还有 %d 个活跃批次", index, t.progress.TotalBatches, len(t.runningBatches))
		t.progress.MessageKey = "group.aiRoute.progress.runtime.batchCompletedWithActive"
		t.progress.MessageArgs = map[string]any{"index": index, "total": t.progress.TotalBatches, "active": len(t.runningBatches)}
	} else if t.progress.CompletedBatches < t.progress.TotalBatches {
		t.progress.CurrentStep = model.AIRouteTaskStepAnalyzingBatches
		t.progress.CurrentBatch = completedBatch
		t.progress.Message = fmt.Sprintf("第 %d/%d 批已完成，等待启动下一批", index, t.progress.TotalBatches)
		t.progress.MessageKey = "group.aiRoute.progress.runtime.batchCompletedWaitingNext"
		t.progress.MessageArgs = map[string]any{"index": index, "total": t.progress.TotalBatches}
	} else {
		t.progress.CurrentStep = model.AIRouteTaskStepParsingResponse
		t.progress.CurrentBatch = completedBatch
		t.progress.Message = "全部批次分析完成，准备校验路由结果"
		t.progress.MessageKey = "group.aiRoute.progress.runtime.allBatchesCompleted"
		t.progress.MessageArgs = nil
	}
	t.mu.Unlock()

	t.emit()
}

func (t *aiRouteProgressTracker) FailBatch(index int, bucket aiRoutePromptBucket, serviceName string, attempt int, message string) {
	if t == nil {
		return
	}

	t.mu.Lock()
	batch, existed := t.runningBatches[index]
	if !existed {
		batch = t.buildRunningBatchLocked(index, bucket, serviceName, attempt, model.AIRouteBatchStatusFailed, message)
	}
	for _, channelID := range batch.ChannelIDs {
		t.decrementChannelActiveLocked(channelID)
		channel := t.channelByIDLocked(channelID)
		if channel == nil {
			continue
		}
		channel.Status = model.AIRouteChannelStatusFailed
		channel.Message = strings.TrimSpace(message)
		channel.MessageKey = "group.aiRoute.progress.runtime.channelFailedWithReason"
		channel.MessageArgs = map[string]any{"reason": strings.TrimSpace(message)}
	}
	delete(t.runningBatches, index)
	delete(t.runningBatchOrder, index)
	t.syncRunningBatchesLocked()
	t.progress.CurrentStep = model.AIRouteTaskStepAnalyzingBatches
	t.progress.CurrentBatch = t.currentBatchFromRunningBatch(batch, string(model.AIRouteBatchStatusFailed), message)
	t.progress.ProgressPercent = t.analysisProgressLocked()
	t.progress.Summary = t.buildSummaryLocked()
	if strings.TrimSpace(message) == "" {
		t.progress.Message = fmt.Sprintf("第 %d/%d 批分析失败", index, t.progress.TotalBatches)
		t.progress.MessageKey = "group.aiRoute.progress.runtime.batchFailedIndexed"
		t.progress.MessageArgs = map[string]any{"index": index, "total": t.progress.TotalBatches}
	} else {
		t.progress.Message = fmt.Sprintf("第 %d/%d 批分析失败：%s", index, t.progress.TotalBatches, strings.TrimSpace(message))
		t.progress.MessageKey = "group.aiRoute.progress.runtime.batchFailedIndexedWithReason"
		t.progress.MessageArgs = map[string]any{"index": index, "total": t.progress.TotalBatches, "reason": strings.TrimSpace(message)}
	}
	t.mu.Unlock()

	t.emit()
}

func (t *aiRouteProgressTracker) SetValidatingRoutes(routeCount int) {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.progress.CurrentStep = model.AIRouteTaskStepValidatingRoutes
	t.progress.CurrentBatch = nil
	t.runningBatches = make(map[int]model.GenerateAIRouteRunningBatch)
	t.runningBatchOrder = make(map[int]int64)
	t.progress.RunningBatches = nil
	t.progress.ProgressPercent = 88
	if routeCount > 0 {
		t.progress.Message = fmt.Sprintf("正在校验 AI 返回的 %d 条候选路由", routeCount)
		t.progress.MessageKey = "group.aiRoute.progress.runtime.validatingRoutes"
		t.progress.MessageArgs = map[string]any{"routes": routeCount}
	} else {
		t.progress.Message = "正在校验 AI 返回的候选路由"
		t.progress.MessageKey = "group.aiRoute.progress.runtime.validatingRoutesGeneric"
		t.progress.MessageArgs = nil
	}
	t.mu.Unlock()

	t.emit()
}

func (t *aiRouteProgressTracker) SetWritingGroup(groupName string) {
	if t == nil {
		return
	}

	groupName = strings.TrimSpace(groupName)

	t.mu.Lock()
	t.progress.CurrentStep = model.AIRouteTaskStepWritingGroups
	t.progress.CurrentBatch = nil
	t.progress.RunningBatches = nil
	t.progress.ProgressPercent = 94
	if groupName == "" {
		t.progress.Message = "正在写入当前分组"
		t.progress.MessageKey = "group.aiRoute.progress.runtime.writingCurrentGroup"
		t.progress.MessageArgs = nil
	} else {
		t.progress.Message = fmt.Sprintf("正在写入分组 %q", groupName)
		t.progress.MessageKey = "group.aiRoute.progress.runtime.writingNamedGroup"
		t.progress.MessageArgs = map[string]any{"group": groupName}
	}
	t.mu.Unlock()

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

	t.mu.Lock()
	t.progress.CurrentStep = model.AIRouteTaskStepWritingGroups
	t.progress.CurrentBatch = nil
	t.progress.RunningBatches = nil
	t.progress.ProgressPercent = 90 + int(float64(index)/float64(total)*8)
	if t.progress.ProgressPercent > 98 {
		t.progress.ProgressPercent = 98
	}

	if requestedModel == "" {
		t.progress.Message = fmt.Sprintf("正在写入第 %d/%d 条路由", index, total)
		t.progress.MessageKey = "group.aiRoute.progress.runtime.writingRoute"
		t.progress.MessageArgs = map[string]any{"index": index, "total": total}
	} else {
		t.progress.Message = fmt.Sprintf("正在写入路由 %q（%d/%d）", requestedModel, index, total)
		t.progress.MessageKey = "group.aiRoute.progress.runtime.writingNamedRoute"
		t.progress.MessageArgs = map[string]any{"route": requestedModel, "index": index, "total": total}
	}
	t.mu.Unlock()

	t.emit()
}

func (t *aiRouteProgressTracker) SetFinalizing(message string) {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.progress.CurrentStep = model.AIRouteTaskStepFinalizing
	t.progress.CurrentBatch = nil
	t.progress.RunningBatches = nil
	t.progress.ProgressPercent = 99
	if strings.TrimSpace(message) == "" {
		t.progress.Message = "正在收尾"
		t.progress.MessageKey = "group.aiRoute.progress.runtime.finalizing"
		t.progress.MessageArgs = nil
	} else {
		t.progress.Message = strings.TrimSpace(message)
		t.progress.MessageKey = "group.aiRoute.progress.runtime.finalizingCustom"
		t.progress.MessageArgs = map[string]any{"message": strings.TrimSpace(message)}
	}
	t.mu.Unlock()

	t.emit()
}

func (t *aiRouteProgressTracker) analysisProgressLocked() int {
	if t == nil || t.totalModels <= 0 {
		return 20
	}

	progressModels := float64(t.completedModels)
	for _, batch := range t.runningBatches {
		progressModels += float64(batch.ModelCount) * airouteBatchFraction(batch.Status)
	}

	percent := 20 + int(progressModels/float64(t.totalModels)*60)
	if percent < 20 {
		return 20
	}
	if percent > 80 {
		return 80
	}
	return percent
}

func (t *aiRouteProgressTracker) buildSummaryLocked() *model.GenerateAIRouteProgressSummary {
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

func (t *aiRouteProgressTracker) buildRunningBatchLocked(
	index int,
	bucket aiRoutePromptBucket,
	serviceName string,
	attempt int,
	status model.AIRouteBatchStatus,
	message string,
) model.GenerateAIRouteRunningBatch {
	channelIDs := make([]int, 0)
	channelNames := make([]string, 0)
	seenChannels := make(map[int]struct{})

	for _, input := range bucket.ModelInputs {
		if _, ok := seenChannels[input.ChannelID]; ok {
			continue
		}
		seenChannels[input.ChannelID] = struct{}{}
		channelIDs = append(channelIDs, input.ChannelID)
		channelNames = append(channelNames, t.channelNameByIDLocked(input.ChannelID))
	}

	return model.GenerateAIRouteRunningBatch{
		Index:        index,
		Total:        t.progress.TotalBatches,
		EndpointType: bucket.PromptEndpointType,
		ModelCount:   len(bucket.ModelInputs),
		ChannelIDs:   channelIDs,
		ChannelNames: channelNames,
		ServiceName:  strings.TrimSpace(serviceName),
		Attempt:      attempt,
		Status:       status,
		Message:      strings.TrimSpace(message),
	}
}

func (t *aiRouteProgressTracker) buildAnalysisMessageLocked(batch model.GenerateAIRouteRunningBatch) string {
	activeCount := len(t.runningBatches)
	if activeCount <= 1 {
		return fmt.Sprintf(
			"正在等待 AI 返回第 %d/%d 批结果（%s，涉及 %d 个渠道 / %d 个模型）",
			batch.Index,
			batch.Total,
			airoutePromptEndpointLabel(batch.EndpointType),
			len(batch.ChannelIDs),
			batch.ModelCount,
		)
	}

	return fmt.Sprintf(
		"正在并发分析 %d 个批次，最近启动第 %d/%d 批（%s，涉及 %d 个渠道 / %d 个模型）",
		activeCount,
		batch.Index,
		batch.Total,
		airoutePromptEndpointLabel(batch.EndpointType),
		len(batch.ChannelIDs),
		batch.ModelCount,
	)
}

func (t *aiRouteProgressTracker) upsertRunningBatchLocked(batch model.GenerateAIRouteRunningBatch) {
	t.runningBatches[batch.Index] = batch
	t.nextBatchOrder++
	t.runningBatchOrder[batch.Index] = t.nextBatchOrder
	t.syncRunningBatchesLocked()
}

func (t *aiRouteProgressTracker) syncRunningBatchesLocked() {
	if len(t.runningBatches) == 0 {
		t.progress.RunningBatches = nil
		return
	}

	indexes := make([]int, 0, len(t.runningBatches))
	for index := range t.runningBatches {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	batches := make([]model.GenerateAIRouteRunningBatch, 0, len(indexes))
	for _, index := range indexes {
		batches = append(batches, t.runningBatches[index])
	}
	t.progress.RunningBatches = cloneAIRouteRunningBatchList(batches)
}

func (t *aiRouteProgressTracker) currentBatchFromRunningLocked() *model.GenerateAIRouteCurrentBatch {
	if len(t.runningBatches) == 0 {
		return nil
	}

	latestIndex := 0
	latestOrder := int64(-1)
	for index, order := range t.runningBatchOrder {
		if order > latestOrder {
			latestIndex = index
			latestOrder = order
		}
	}
	batch, ok := t.runningBatches[latestIndex]
	if !ok {
		return nil
	}
	return t.currentBatchFromRunningBatch(batch, string(batch.Status), batch.Message)
}

func (t *aiRouteProgressTracker) currentBatchFromRunningBatch(
	batch model.GenerateAIRouteRunningBatch,
	status string,
	message string,
) *model.GenerateAIRouteCurrentBatch {
	return &model.GenerateAIRouteCurrentBatch{
		Index:        batch.Index,
		Total:        batch.Total,
		EndpointType: batch.EndpointType,
		ModelCount:   batch.ModelCount,
		ChannelIDs:   append([]int(nil), batch.ChannelIDs...),
		ChannelNames: append([]string(nil), batch.ChannelNames...),
		ServiceName:  batch.ServiceName,
		Attempt:      batch.Attempt,
		Status:       strings.TrimSpace(status),
		Message:      strings.TrimSpace(message),
	}
}

func (t *aiRouteProgressTracker) incrementChannelActiveLocked(channelID int, endpointType string) {
	if channelID <= 0 {
		return
	}

	t.channelActiveCounts[channelID]++
	channel := t.channelByIDLocked(channelID)
	if channel == nil {
		return
	}
	channel.Status = model.AIRouteChannelStatusRunning
	channel.Message = fmt.Sprintf("正在分析 %s 模型", airoutePromptEndpointLabel(endpointType))
	channel.MessageKey = "group.aiRoute.progress.runtime.channelAnalyzingEndpoint"
	channel.MessageArgs = map[string]any{"endpoint": airoutePromptEndpointLabel(endpointType)}
}

func (t *aiRouteProgressTracker) decrementChannelActiveLocked(channelID int) {
	if channelID <= 0 {
		return
	}

	if current := t.channelActiveCounts[channelID]; current > 0 {
		current--
		if current == 0 {
			delete(t.channelActiveCounts, channelID)
		} else {
			t.channelActiveCounts[channelID] = current
		}
	}

	channel := t.channelByIDLocked(channelID)
	if channel == nil {
		return
	}
	t.refreshChannelStateLocked(channel)
}

func (t *aiRouteProgressTracker) refreshChannelStateLocked(channel *model.GenerateAIRouteChannelProgress) {
	if channel == nil {
		return
	}

	if t.channelActiveCounts[channel.ChannelID] > 0 {
		channel.Status = model.AIRouteChannelStatusRunning
		if strings.TrimSpace(channel.Message) == "" {
			channel.Message = "正在分析"
			channel.MessageKey = "group.aiRoute.progress.runtime.channelAnalyzing"
			channel.MessageArgs = nil
		}
		return
	}

	if channel.ProcessedModels >= channel.TotalModels && channel.TotalModels > 0 {
		channel.Status = model.AIRouteChannelStatusCompleted
		channel.Message = "已完成"
		channel.MessageKey = "group.aiRoute.progress.runtime.channelCompleted"
		channel.MessageArgs = nil
		return
	}

	if channel.Status == model.AIRouteChannelStatusFailed {
		return
	}

	channel.Status = model.AIRouteChannelStatusPending
	channel.Message = ""
	channel.MessageKey = ""
	channel.MessageArgs = nil
}

func (t *aiRouteProgressTracker) buildAnalysisMessageMetaLocked(batch model.GenerateAIRouteRunningBatch) (string, map[string]any) {
	activeCount := len(t.runningBatches)
	args := map[string]any{
		"index":    batch.Index,
		"total":    batch.Total,
		"endpoint": airoutePromptEndpointLabel(batch.EndpointType),
		"channels": len(batch.ChannelIDs),
		"models":   batch.ModelCount,
	}
	if activeCount <= 1 {
		return "group.aiRoute.progress.runtime.waitingBatchResult", args
	}
	args["active"] = activeCount
	return "group.aiRoute.progress.runtime.parallelAnalyzing", args
}

func (t *aiRouteProgressTracker) channelByIDLocked(channelID int) *model.GenerateAIRouteChannelProgress {
	index, ok := t.channelIndexByID[channelID]
	if !ok || index < 0 || index >= len(t.progress.Channels) {
		return nil
	}
	return &t.progress.Channels[index]
}

func (t *aiRouteProgressTracker) channelNameByIDLocked(channelID int) string {
	channel := t.channelByIDLocked(channelID)
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

	t.mu.Lock()
	t.progress.Summary = t.buildSummaryLocked()
	snapshot := cloneAIRouteProgressSnapshot(t.progress)
	t.mu.Unlock()

	t.report(snapshot)
}

func airouteBatchFraction(status model.AIRouteBatchStatus) float64 {
	switch status {
	case model.AIRouteBatchStatusParsing:
		return 0.72
	case model.AIRouteBatchStatusRetrying:
		return 0.08
	default:
		return 0.15
	}
}

func fallbackAIRouteServiceName(serviceName string) string {
	if name := strings.TrimSpace(serviceName); name != "" {
		return name
	}
	return "default"
}

func cloneAIRouteProgressSnapshot(progress model.GenerateAIRouteProgress) model.GenerateAIRouteProgress {
	cloned := progress
	cloned.Summary = cloneAIRouteProgressSummary(progress.Summary)
	cloned.CurrentBatch = cloneAIRouteCurrentBatch(progress.CurrentBatch)
	cloned.RunningBatches = cloneAIRouteRunningBatchList(progress.RunningBatches)
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

func cloneAIRouteResult(result *model.GenerateAIRouteResult) *model.GenerateAIRouteResult {
	if result == nil {
		return nil
	}

	cloned := *result
	return &cloned
}
