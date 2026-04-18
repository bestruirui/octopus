package helper

import (
	"sync"

	"github.com/lingyuins/octopus/internal/model"
)

var aiRouteProgressSubscribers = struct {
	mu     sync.RWMutex
	byTask map[string]map[chan model.GenerateAIRouteProgress]struct{}
}{
	byTask: make(map[string]map[chan model.GenerateAIRouteProgress]struct{}),
}

func SubscribeGenerateAIRouteProgress(id string) (<-chan model.GenerateAIRouteProgress, func()) {
	ch := make(chan model.GenerateAIRouteProgress, 1)
	if id == "" {
		return ch, func() {}
	}

	aiRouteProgressSubscribers.mu.Lock()
	if _, ok := aiRouteProgressSubscribers.byTask[id]; !ok {
		aiRouteProgressSubscribers.byTask[id] = make(map[chan model.GenerateAIRouteProgress]struct{})
	}
	aiRouteProgressSubscribers.byTask[id][ch] = struct{}{}
	aiRouteProgressSubscribers.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			aiRouteProgressSubscribers.mu.Lock()
			subscribers := aiRouteProgressSubscribers.byTask[id]
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(aiRouteProgressSubscribers.byTask, id)
			}
			aiRouteProgressSubscribers.mu.Unlock()
		})
	}

	return ch, unsubscribe
}

func publishGenerateAIRouteProgress(progress *model.GenerateAIRouteProgress) {
	if progress == nil || progress.ID == "" {
		return
	}

	aiRouteProgressSubscribers.mu.RLock()
	subscribers := aiRouteProgressSubscribers.byTask[progress.ID]
	if len(subscribers) == 0 {
		aiRouteProgressSubscribers.mu.RUnlock()
		return
	}

	clone := cloneAIRouteProgress(progress)
	targets := make([]chan model.GenerateAIRouteProgress, 0, len(subscribers))
	for ch := range subscribers {
		targets = append(targets, ch)
	}
	aiRouteProgressSubscribers.mu.RUnlock()

	for _, ch := range targets {
		select {
		case ch <- clone:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- clone:
			default:
			}
		}
	}
}
