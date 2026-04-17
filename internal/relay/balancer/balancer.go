package balancer

import (
	"math/rand"
	"sort"
	"sync/atomic"

	"github.com/lingyuins/octopus/internal/model"
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
	case model.GroupModeAuto:
		return &Auto{}
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

// Weighted 加权分配：按权重从高到低排序
type Weighted struct{}

func (b *Weighted) Candidates(items []model.GroupItem) []model.GroupItem {
	if len(items) == 0 {
		return nil
	}
	return sortByWeight(items)
}

// Auto 自动策略：探索优先，基于成功率动态选择
// - 探索阶段：优先选择未达到最小样本数的渠道
// - 利用阶段：选择成功率最高的渠道
// - 相同成功率时：按权重/优先级兜底
type Auto struct{}

type autoScoredItem struct {
	item         model.GroupItem
	score        float64
	totalSamples int
	explored     bool
}

func (b *Auto) Candidates(items []model.GroupItem) []model.GroupItem {
	if len(items) == 0 {
		return nil
	}

	minSamples := getMinSamples()
	timeWindow := getTimeWindow()

	// Calculate scores for each item
	scored := make([]autoScoredItem, len(items))
	for i, item := range items {
		stats := getOrCreateStats(item.ChannelID, item.ModelName)
		successRate, totalSamples := stats.GetStats(timeWindow)

		scored[i].item = item
		scored[i].totalSamples = totalSamples
		scored[i].explored = totalSamples >= minSamples

		if !scored[i].explored {
			// Exploration phase: items with fewer samples are tried first.
			scored[i].score = 0
		} else {
			// Exploitation phase: use success rate
			scored[i].score = successRate
		}
	}

	// Sort: unexplored first, then by success rate descending
	sort.SliceStable(scored, func(i, j int) bool {
		// Exploration priority: unexplored channels come first
		if scored[i].explored != scored[j].explored {
			return !scored[i].explored
		}

		// Exploration phase: prefer lower-sampled candidates first so other
		// candidates are not starved by weight/priority.
		if !scored[i].explored {
			if scored[i].totalSamples != scored[j].totalSamples {
				return scored[i].totalSamples < scored[j].totalSamples
			}
			return compareByWeightPriority(scored[i].item, scored[j].item)
		}

		// Both explored: sort by success rate descending
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}

		// Same success rate: prefer the candidate backed by more in-window
		// samples before falling back to weight/priority.
		if scored[i].totalSamples != scored[j].totalSamples {
			return scored[i].totalSamples > scored[j].totalSamples
		}

		// Same success rate: fall back to weight/priority
		return compareByWeightPriority(scored[i].item, scored[j].item)
	})

	// Extract sorted items
	result := make([]model.GroupItem, len(items))
	for i, s := range scored {
		result[i] = s.item
	}
	return result
}

// compareByWeightPriority compares two items by weight (descending), then priority (ascending).
// Returns true if a should come before b.
func compareByWeightPriority(a, b model.GroupItem) bool {
	// Compare weight (higher weight first)
	leftWeight := a.Weight
	if leftWeight <= 0 {
		leftWeight = 1
	}
	rightWeight := b.Weight
	if rightWeight <= 0 {
		rightWeight = 1
	}
	if leftWeight != rightWeight {
		return leftWeight > rightWeight
	}

	// Compare priority (lower priority value first)
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}

	// Tie-breaker: channel ID, then model name
	if a.ChannelID != b.ChannelID {
		return a.ChannelID < b.ChannelID
	}
	return a.ModelName < b.ModelName
}

func sortByPriority(items []model.GroupItem) []model.GroupItem {
	sorted := make([]model.GroupItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return sorted
}

func sortByWeight(items []model.GroupItem) []model.GroupItem {
	sorted := make([]model.GroupItem, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(i, j int) bool {
		leftWeight := sorted[i].Weight
		if leftWeight <= 0 {
			leftWeight = 1
		}
		rightWeight := sorted[j].Weight
		if rightWeight <= 0 {
			rightWeight = 1
		}
		if leftWeight != rightWeight {
			return leftWeight > rightWeight
		}
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}
		if sorted[i].ChannelID != sorted[j].ChannelID {
			return sorted[i].ChannelID < sorted[j].ChannelID
		}
		return sorted[i].ModelName < sorted[j].ModelName
	})
	return sorted
}
