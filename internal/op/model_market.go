package op

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/lingyuins/octopus/internal/model"
)

type modelMarketStatsAggregate struct {
	waitTime       int64
	requestSuccess int64
	requestFailed  int64
}

func ModelMarketGet(ctx context.Context, lastUpdateTime time.Time) (model.ModelMarketResponse, error) {
	models, err := LLMList(ctx)
	if err != nil {
		return model.ModelMarketResponse{}, err
	}

	modelChannels, err := ChannelLLMList(ctx)
	if err != nil {
		return model.ModelMarketResponse{}, err
	}

	items, summary := buildModelMarket(models, modelChannels, channelCache.GetAll(), StatsModelList(), lastUpdateTime)
	return model.ModelMarketResponse{
		Summary: summary,
		Items:   items,
	}, nil
}

func buildModelMarket(
	models []model.LLMInfo,
	modelChannels []model.LLMChannel,
	channelsByID map[int]model.Channel,
	stats []model.StatsModel,
	lastUpdateTime time.Time,
) ([]model.ModelMarketItem, model.ModelMarketSummary) {
	statsByModelName := make(map[string]modelMarketStatsAggregate)
	for _, item := range stats {
		name := strings.ToLower(strings.TrimSpace(item.Name))
		if name == "" {
			continue
		}
		aggregate := statsByModelName[name]
		aggregate.waitTime += item.WaitTime
		aggregate.requestSuccess += item.RequestSuccess
		aggregate.requestFailed += item.RequestFailed
		statsByModelName[name] = aggregate
	}

	channelsByModelName := make(map[string][]model.ModelMarketChannel)
	seenChannelsByModel := make(map[string]map[int]struct{})
	for _, item := range modelChannels {
		name := strings.ToLower(strings.TrimSpace(item.Name))
		if name == "" {
			continue
		}
		if seenChannelsByModel[name] == nil {
			seenChannelsByModel[name] = make(map[int]struct{})
		}
		if _, exists := seenChannelsByModel[name][item.ChannelID]; exists {
			continue
		}
		seenChannelsByModel[name][item.ChannelID] = struct{}{}

		channel := channelsByID[item.ChannelID]
		enabledKeyCount := 0
		for _, key := range channel.Keys {
			if key.Enabled {
				enabledKeyCount++
			}
		}

		channelsByModelName[name] = append(channelsByModelName[name], model.ModelMarketChannel{
			ChannelID:       item.ChannelID,
			ChannelName:     item.ChannelName,
			Enabled:         item.Enabled,
			EnabledKeyCount: enabledKeyCount,
		})
	}

	items := make([]model.ModelMarketItem, 0, len(models))
	totalWaitTime := int64(0)
	totalRequests := int64(0)
	uniqueChannels := make(map[int]struct{})
	coverageCount := 0

	for _, llm := range models {
		name := strings.ToLower(strings.TrimSpace(llm.Name))
		modelItemChannels := channelsByModelName[name]
		if modelItemChannels == nil {
			modelItemChannels = make([]model.ModelMarketChannel, 0)
		}
		slices.SortFunc(modelItemChannels, func(a, b model.ModelMarketChannel) int {
			return strings.Compare(a.ChannelName, b.ChannelName)
		})

		enabledKeyCount := 0
		for _, channel := range modelItemChannels {
			enabledKeyCount += channel.EnabledKeyCount
			uniqueChannels[channel.ChannelID] = struct{}{}
		}

		statsAggregate := statsByModelName[name]
		requestCount := statsAggregate.requestSuccess + statsAggregate.requestFailed
		averageLatency := int64(0)
		successRate := 0.0
		if requestCount > 0 {
			averageLatency = statsAggregate.waitTime / requestCount
			successRate = float64(statsAggregate.requestSuccess) / float64(requestCount)
		}

		totalWaitTime += statsAggregate.waitTime
		totalRequests += requestCount
		coverageCount += len(modelItemChannels)

		items = append(items, model.ModelMarketItem{
			Name:             llm.Name,
			Input:            llm.Input,
			Output:           llm.Output,
			CacheRead:        llm.CacheRead,
			CacheWrite:       llm.CacheWrite,
			ChannelCount:     len(modelItemChannels),
			EnabledKeyCount:  enabledKeyCount,
			AverageLatencyMS: averageLatency,
			SuccessRate:      successRate,
			RequestSuccess:   statsAggregate.requestSuccess,
			RequestFailed:    statsAggregate.requestFailed,
			Channels:         modelItemChannels,
		})
	}

	slices.SortFunc(items, func(a, b model.ModelMarketItem) int {
		return strings.Compare(a.Name, b.Name)
	})

	summaryAverageLatency := int64(0)
	if totalRequests > 0 {
		summaryAverageLatency = totalWaitTime / totalRequests
	}

	return items, model.ModelMarketSummary{
		ModelCount:         len(items),
		CoverageCount:      coverageCount,
		UniqueChannelCount: len(uniqueChannels),
		AverageLatencyMS:   summaryAverageLatency,
		LastUpdateTime:     lastUpdateTime,
	}
}
