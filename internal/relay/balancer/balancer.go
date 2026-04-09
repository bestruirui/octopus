package balancer

import (
	"context"
	"math/rand"
	"sort"
	"sync/atomic"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
)

var roundRobinCounter uint64

// Balancer 根据负载均衡模式选择通道
type Balancer interface {
	// Candidates 返回按策略排序的候选列表
	// 调用方在遍历候选列表时自行检查熔断状态
	Candidates(items []model.GroupItem) []model.GroupItem
}

// GetBalancer 根据模式返回对应的负载均衡器
func GetBalancer(mode model.GroupMode) Balancer {
	switch mode {
	case model.GroupModeRoundRobin:
		return &RoundRobin{}
	case model.GroupModeRandom:
		return &Random{}
	case model.GroupModeFailover:
		return &Failover{}
	case model.GroupModeWeighted:
		return &Weighted{}
	case model.GroupModeScored:
		return &Scored{}
	default:
		return &RoundRobin{}
	}
}

// RoundRobin 轮询：从上次位置开始轮转排列
type RoundRobin struct{}

func (b *RoundRobin) Candidates(items []model.GroupItem) []model.GroupItem {
	n := len(items)
	if n == 0 {
		return nil
	}
	idx := int(atomic.AddUint64(&roundRobinCounter, 1) % uint64(n))
	result := make([]model.GroupItem, n)
	for i := 0; i < n; i++ {
		result[i] = items[(idx+i)%n]
	}
	return result
}

// Random 随机：随机打乱所有 items
type Random struct{}

func (b *Random) Candidates(items []model.GroupItem) []model.GroupItem {
	n := len(items)
	if n == 0 {
		return nil
	}
	result := make([]model.GroupItem, n)
	copy(result, items)
	rand.Shuffle(n, func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}

// Failover 故障转移：按优先级排序
type Failover struct{}

func (b *Failover) Candidates(items []model.GroupItem) []model.GroupItem {
	if len(items) == 0 {
		return nil
	}
	return sortByPriority(items)
}

// Weighted 加权分配：按权重概率排序
type Weighted struct{}

func (b *Weighted) Candidates(items []model.GroupItem) []model.GroupItem {
	n := len(items)
	if n == 0 {
		return nil
	}

	// 构建加权随机排序
	type weightedItem struct {
		item   model.GroupItem
		score  float64
	}

	totalWeight := 0
	for _, item := range items {
		w := item.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}

	scored := make([]weightedItem, n)
	for i, item := range items {
		w := item.Weight
		if w <= 0 {
			w = 1
		}
		// 给每个 item 一个加权随机分数：weight/totalWeight 作为概率基础，加上随机扰动
		scored[i] = weightedItem{
			item:  item,
			score: rand.Float64() * float64(w) / float64(totalWeight),
		}
	}

	// 按分数降序排列（分数越高优先级越高）
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]model.GroupItem, n)
	for i := range scored {
		result[i] = scored[i].item
	}
	return result
}

// Scored 评分优先：先按优先级兜底，再按渠道/模型健康度综合评分排序。
type Scored struct{}

func (b *Scored) Candidates(items []model.GroupItem) []model.GroupItem {
	if len(items) == 0 {
		return nil
	}
	scored := sortByPriority(items)
	sort.SliceStable(scored, func(i, j int) bool {
		left := scoreGroupItem(scored[i])
		right := scoreGroupItem(scored[j])
		if left == right {
			if scored[i].Priority == scored[j].Priority {
				return scored[i].ChannelID < scored[j].ChannelID
			}
			return scored[i].Priority < scored[j].Priority
		}
		return left > right
	})
	return scored
}

func scoreGroupItem(item model.GroupItem) float64 {
	channel, err := op.ChannelGet(item.ChannelID, context.Background())
	if err != nil {
		return 0
	}
	weights := op.GetHealthScoreWeights()
	totalKeys := len(channel.Keys)
	enabledKeys := 0
	for _, key := range channel.Keys {
		if key.Enabled && key.ChannelKey != "" {
			enabledKeys++
		}
	}
	baseDelay := 0
	if url := channel.GetBaseUrl(); url != "" {
		for _, bu := range channel.BaseUrls {
			if bu.URL == url {
				baseDelay = bu.Delay
				break
			}
		}
	}
	modelStats := op.StatsModelGet(op.StatsModelKey(item.ChannelID, item.ModelName))
	stats := modelStats.StatsMetrics
	if modelStats.Name == "" {
		stats = op.StatsChannelGet(item.ChannelID).StatsMetrics
	}
	score := op.ComputeHealthScore(stats, baseDelay, enabledKeys, totalKeys)
	if item.Priority > 0 {
		score += weights.PriorityBoost / float64(item.Priority)
	}
	if item.Weight > 0 {
		score += float64(item.Weight) * weights.WeightBoost
	}
	return score
}

func sortByPriority(items []model.GroupItem) []model.GroupItem {
	sorted := make([]model.GroupItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return sorted
}
