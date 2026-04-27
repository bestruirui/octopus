package task

import (
	"context"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/utils/diff"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/bestruirui/octopus/internal/utils/xstrings"
)

var lastSyncModelsTime = time.Now()

// SyncModelsTask 同步模型任务
func SyncModelsTask() {
	log.Debugf("sync models task started")
	startTime := time.Now()
	defer func() {
		log.Debugf("sync models task finished, sync time: %s", time.Since(startTime))
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	channels, err := op.ChannelList(ctx)
	if err != nil {
		log.Errorf("failed to list channels: %v", err)
		return
	}
	totalNewModels := make([]string, 0, 128)
	seenTotalNewModels := make(map[string]struct{}, 128)
	for _, channel := range channels {
		if !channel.AutoSync {
			continue
		}
		fetchModels, err := helper.FetchModels(ctx, channel)
		if err != nil {
			log.Warnf("failed to fetch models for channel %s: %v", channel.Name, err)
			continue
		}
		oldModels := xstrings.SplitTrimCompact(",", channel.Model)
		newModels := xstrings.TrimCompact(fetchModels)

		// 合并 custom_model 中的模型，避免覆盖用户手动添加的模型
		customModels := xstrings.SplitTrimCompact(",", channel.CustomModel)
		customModelSet := make(map[string]struct{}, len(customModels))
		for _, cm := range customModels {
			if cm != "" {
				customModelSet[cm] = struct{}{}
			}
		}

		// mergedModels 只包含 fetch 来的模型，排除已在 custom_model 中的（避免重复写入 Model 字段）
		newModelSet := make(map[string]struct{}, len(newModels))
		for _, m := range newModels {
			if m != "" && m != " " {
				if _, isCustom := customModelSet[m]; !isCustom {
					newModelSet[m] = struct{}{}
				}
			}
		}

		mergedModels := make([]string, 0, len(newModelSet))
		for m := range newModelSet {
			mergedModels = append(mergedModels, m)
		}

		// totalNewModels 统计时同时包含 fetch 模型和 custom_model
		allModels := make([]string, 0, len(mergedModels)+len(customModels))
		allModels = append(allModels, mergedModels...)
		allModels = append(allModels, customModels...)
		for _, m := range allModels {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			m = strings.ToLower(m)
			if _, ok := seenTotalNewModels[m]; ok {
				continue
			}
			seenTotalNewModels[m] = struct{}{}
			totalNewModels = append(totalNewModels, m)
		}
		deletedModels, addedModels := diff.Diff(oldModels, mergedModels)
		if len(deletedModels) > 0 || len(addedModels) > 0 {
			fetchModelStr := strings.Join(mergedModels, ",")
			if _, err := op.ChannelUpdate(&model.ChannelUpdateRequest{
				ID:    channel.ID,
				Model: &fetchModelStr,
			}, ctx); err != nil {
				log.Errorf("failed to update channel %s: %v", channel.Name, err)
				continue
			}
		}
		// 批量删除消失的模型对应的 GroupItem（但不删除 custom_model 中的模型关联）
		if len(deletedModels) > 0 {
			// 过滤掉 custom_model 中的模型，避免误删用户手动配置的分组关联
			actualDeletedModels := make([]string, 0, len(deletedModels))
			for _, m := range deletedModels {
				if _, isCustom := customModelSet[m]; !isCustom {
					actualDeletedModels = append(actualDeletedModels, m)
				}
			}

			if len(actualDeletedModels) > 0 {
				log.Infof("deleted channel %s models: %v", channel.Name, actualDeletedModels)
				keys := make([]model.GroupIDAndLLMName, len(actualDeletedModels))
				for i, m := range actualDeletedModels {
					keys[i] = model.GroupIDAndLLMName{ChannelID: channel.ID, ModelName: m}
				}
				if err := op.GroupItemBatchDelByChannelAndModels(keys, ctx); err != nil {
					log.Errorf("failed to batch delete group items for channel %s: %v", channel.Name, err)
				}
			}
		}

		// 自动分组
		if len(mergedModels) > 0 {
			helper.ChannelAutoGroup(&channel, ctx)
		}
	}
	llmPrice, err := op.LLMList(ctx)
	if err != nil {
		log.Errorf("failed to list models price: %v", err)
		return
	}
	llmPriceNames := make([]string, 0, len(llmPrice))
	for _, price := range llmPrice {
		llmPriceNames = append(llmPriceNames, price.Name)
	}

	deletedNorm, addedNorm := diff.Diff(llmPriceNames, totalNewModels)
	if len(deletedNorm) > 0 {
		if err := helper.LLMPriceDeleteFromDBWithNoPrice(deletedNorm, ctx); err != nil {
			log.Errorf("failed to batch delete models price: %v", err)
		}
	}
	if len(addedNorm) > 0 {
		if err := helper.LLMPriceAddToDB(addedNorm, ctx); err != nil {
			log.Errorf("failed to add models price: %v", err)
		}
	}
	lastSyncModelsTime = time.Now()
}

func GetLastSyncModelsTime() time.Time {
	return lastSyncModelsTime
}
