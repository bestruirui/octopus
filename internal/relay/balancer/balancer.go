package balancer

import (
	"math/rand"
	"sort"
	"sync/atomic"

	"github.com/bestruirui/octopus/internal/model"
)

var roundRobinCounter uint64

// Balancer selects channel based on load balancing mode
type Balancer interface {
	Select(items []model.GroupItem) *model.GroupItem
	Next(items []model.GroupItem, current *model.GroupItem) (*model.GroupItem, bool)
}

// GetBalancer returns balancer by mode
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

// RoundRobin balancer
type RoundRobin struct{}

func (b *RoundRobin) Select(items []model.GroupItem) *model.GroupItem {
	if len(items) == 0 {
		return nil
	}
	idx := atomic.AddUint64(&roundRobinCounter, 1) % uint64(len(items))
	return &items[idx]
}

func (b *RoundRobin) Next(items []model.GroupItem, current *model.GroupItem) (*model.GroupItem, bool) {
	if len(items) == 0 {
		return nil, false
	}
	return b.Select(items), true
}

// Random balancer
type Random struct{}

func (b *Random) Select(items []model.GroupItem) *model.GroupItem {
	if len(items) == 0 {
		return nil
	}
	return &items[rand.Intn(len(items))]
}

func (b *Random) Next(items []model.GroupItem, current *model.GroupItem) (*model.GroupItem, bool) {
	if len(items) == 0 {
		return nil, false
	}
	return b.Select(items), true
}

// Failover balancer - tries by priority, falls back on failure
type Failover struct{}

func (b *Failover) Select(items []model.GroupItem) *model.GroupItem {
	if len(items) == 0 {
		return nil
	}
	sorted := sortByPriority(items)
	return &sorted[0]
}

func (b *Failover) Next(items []model.GroupItem, current *model.GroupItem) (*model.GroupItem, bool) {
	if len(items) == 0 {
		return nil, false
	}
	next := b.Select(items)
	if next == nil {
		return nil, false
	}
	return next, true
}

// Weighted balancer
type Weighted struct{}

func (b *Weighted) Select(items []model.GroupItem) *model.GroupItem {
	if len(items) == 0 {
		return nil
	}
	totalWeight := 0
	for _, item := range items {
		totalWeight += item.Weight
	}
	if totalWeight == 0 {
		return &items[0]
	}
	r := rand.Intn(totalWeight)
	for i := range items {
		r -= items[i].Weight
		if r < 0 {
			return &items[i]
		}
	}
	return &items[0]
}

func (b *Weighted) Next(items []model.GroupItem, current *model.GroupItem) (*model.GroupItem, bool) {
	if len(items) == 0 {
		return nil, false
	}
	return b.Select(items), true
}

func sortByPriority(items []model.GroupItem) []model.GroupItem {
	sorted := make([]model.GroupItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return sorted
}
