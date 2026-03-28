package balancer

import (
	"math/rand"
	"sort"
	"sync/atomic"

	"github.com/bestruirui/octopus/internal/model"
)

var roundRobinCounter uint64
var weightedDiversifyCounter uint64

const (
	// smartDiversifyMaxCandidates 限制接近最优分数的轮转范围，避免扩大到长尾候选。
	smartDiversifyMaxCandidates = 3
	// smartDiversifyScoreThreshold 定义“接近最优”的分数差阈值。
	smartDiversifyScoreThreshold = 0.05
)

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

// Weighted 加权分配：按综合评分择优
type Weighted struct{}

func (b *Weighted) Candidates(items []model.GroupItem) []model.GroupItem {
	n := len(items)
	if n == 0 {
		return nil
	}

	// 构建智能择优排序：
	// score = 手动权重(30%) + 近1h成功率(40%) + 近24h成功率(30%)
	// 同分时按权重、优先级稳定排序，避免抖动。
	type scoredItem struct {
		item   model.GroupItem
		score  float64
	}

	totalWeight := 0.0
	for _, item := range items {
		w := item.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += float64(w)
	}

	scored := make([]scoredItem, n)
	for i, item := range items {
		w := item.Weight
		if w <= 0 {
			w = 1
		}
		manualWeight := float64(w) / totalWeight
		success1h, total1h, success24h := getSmartSuccessRates(item.ChannelID, item.ModelName)
		effective1hWeight, effective24hWeight := smartDynamicWeights(total1h)
		score := smartWeightManual*manualWeight + effective1hWeight*success1h + effective24hWeight*success24h
		scored[i] = scoredItem{item: item, score: score}
	}

	// 按分数降序排列（分数越高优先级越高）
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].item.Weight != scored[j].item.Weight {
			return scored[i].item.Weight > scored[j].item.Weight
		}
		if scored[i].item.Priority != scored[j].item.Priority {
			return scored[i].item.Priority < scored[j].item.Priority
		}
		if scored[i].item.ChannelID != scored[j].item.ChannelID {
			return scored[i].item.ChannelID < scored[j].item.ChannelID
		}
		return scored[i].item.ModelName < scored[j].item.ModelName
	})

	// 轻量多站点分流：在“接近最优分数”的头部候选中做轮转，避免单点过热。
	if n > 1 {
		limit := 1
		topScore := scored[0].score
		for limit < n && limit < smartDiversifyMaxCandidates {
			if topScore-scored[limit].score > smartDiversifyScoreThreshold {
				break
			}
			limit++
		}
		if limit > 1 {
			offset := int(atomic.AddUint64(&weightedDiversifyCounter, 1) % uint64(limit))
			if offset > 0 {
				head := append([]scoredItem(nil), scored[:limit]...)
				for i := 0; i < limit; i++ {
					scored[i] = head[(i+offset)%limit]
				}
			}
		}
	}

	result := make([]model.GroupItem, n)
	for i := range scored {
		result[i] = scored[i].item
	}
	return result
}

func sortByPriority(items []model.GroupItem) []model.GroupItem {
	sorted := make([]model.GroupItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return sorted
}
