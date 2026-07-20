package balancer

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bestruirui/octopus/internal/model"
)

// 被动延迟 EWMA：真实请求成功/失败后更新，用于健康分选路。
// 不替代分组模式（轮询/随机/故障迁移/加权），只在 Candidates 之后 soft re-rank。

type latencySample struct {
	ewmaMS  float64
	success uint64
	fail    uint64
	mu      sync.Mutex
}

var (
	latencyStore   sync.Map // channelID(int) -> *latencySample
	scoreRoutingOn atomic.Bool
)

func init() {
	scoreRoutingOn.Store(true)
}

// SetHealthScoreRouting 热更新开关
func SetHealthScoreRouting(on bool) {
	scoreRoutingOn.Store(on)
}

// HealthScoreRoutingEnabled 是否启用健康分选路
func HealthScoreRoutingEnabled() bool {
	return scoreRoutingOn.Load()
}

// RecordLatency 记录一次真实请求延迟（ms）与成败
func RecordLatency(channelID int, latencyMS int64, success bool) {
	if channelID <= 0 {
		return
	}
	if latencyMS < 0 {
		latencyMS = 0
	}
	v, _ := latencyStore.LoadOrStore(channelID, &latencySample{})
	s := v.(*latencySample)
	s.mu.Lock()
	defer s.mu.Unlock()

	const alpha = 0.3
	if success {
		s.success++
		if s.ewmaMS <= 0 {
			s.ewmaMS = float64(latencyMS)
		} else {
			s.ewmaMS = alpha*float64(latencyMS) + (1-alpha)*s.ewmaMS
		}
	} else {
		s.fail++
		// 失败时把 EWMA 往上拉，降低后续优先度
		penalized := float64(latencyMS) * 1.5
		if penalized < 500 {
			penalized = 500
		}
		if s.ewmaMS <= 0 {
			s.ewmaMS = penalized
		} else {
			s.ewmaMS = alpha*penalized + (1-alpha)*s.ewmaMS
		}
	}
	_ = time.Now() // keep import if needed later
}

// ChannelScore 0~100，越高越好。综合：探活状态 + 被动 EWMA + 成功率
func ChannelScore(channelID int) float64 {
	h := GetChannelHealth(channelID)
	base := 80.0
	switch h.Status {
	case HealthHealthy:
		base = 95
	case HealthDegraded:
		base = 55
	case HealthUnhealthy:
		return 0
	case HealthUnknown:
		base = 70
	}

	if v, ok := latencyStore.Load(channelID); ok {
		s := v.(*latencySample)
		s.mu.Lock()
		lat := s.ewmaMS
		succ, fail := s.success, s.fail
		s.mu.Unlock()

		if lat > 0 {
			// 100ms → 0，5000ms → -40
			penalty := math.Min(40, lat/125.0)
			base -= penalty
		}
		total := succ + fail
		if total >= 5 {
			rate := float64(succ) / float64(total)
			// 100% → +10，50% → -15
			base += (rate - 0.8) * 50
		}
	}
	if base < 0 {
		return 0
	}
	if base > 100 {
		return 100
	}
	return base
}

// ReorderByHealthScore soft re-rank 候选列表：分数高的靠前，同分保相对顺序。
// Failover 模式仍尊重 Priority 主序：仅在 Priority 相同的段内重排。
func ReorderByHealthScore(items []model.GroupItem, mode model.GroupMode) []model.GroupItem {
	if !scoreRoutingOn.Load() || len(items) <= 1 {
		return items
	}

	type scored struct {
		item  model.GroupItem
		score float64
		idx   int
	}
	arr := make([]scored, len(items))
	for i, it := range items {
		arr[i] = scored{item: it, score: ChannelScore(it.ChannelID), idx: i}
	}

	if mode == model.GroupModeFailover {
		// Failover：Priority 升序主排序，同分内按健康分降序
		sort.SliceStable(arr, func(i, j int) bool {
			if arr[i].item.Priority != arr[j].item.Priority {
				return arr[i].item.Priority < arr[j].item.Priority
			}
			if arr[i].score != arr[j].score {
				return arr[i].score > arr[j].score
			}
			return arr[i].idx < arr[j].idx
		})
	} else {
		// 其它模式：健康分优先，同分保 Candidates 原序
		sort.SliceStable(arr, func(i, j int) bool {
			if arr[i].score != arr[j].score {
				return arr[i].score > arr[j].score
			}
			return arr[i].idx < arr[j].idx
		})
	}

	out := make([]model.GroupItem, len(arr))
	for i := range arr {
		out[i] = arr[i].item
	}
	return out
}
