package task

import (
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/utils/log"
)

type taskEntry struct {
	name       string
	interval   time.Duration
	fn         func()
	runOnStart bool
	ticker     *time.Ticker
	stopCh     chan struct{}
	updateCh   chan time.Duration
}

var (
	tasks   = make(map[string]*taskEntry)
	tasksMu sync.RWMutex
)

// Register 注册一个定时任务
// runOnStart: 是否在启动时立即执行一次
func Register(name string, interval time.Duration, runOnStart bool, fn func()) {
	// 如果间隔为0或负数，跳过注册（禁用该任务）
	if interval <= 0 {
		log.Infof("task %s disabled (interval: %v)", name, interval)
		return
	}

	tasksMu.Lock()
	defer tasksMu.Unlock()

	if _, exists := tasks[name]; exists {
		log.Warnf("task %s already registered, skipping", name)
		return
	}

	tasks[name] = &taskEntry{
		name:       name,
		interval:   interval,
		fn:         fn,
		runOnStart: runOnStart,
		stopCh:     make(chan struct{}),
		updateCh:   make(chan time.Duration),
	}
	log.Debugf("task %s registered with interval %v, runOnStart: %v", name, interval, runOnStart)
}

// Update 更新任务的执行间隔
func Update(name string, interval time.Duration) {
	tasksMu.RLock()
	entry, exists := tasks[name]
	tasksMu.RUnlock()

	if !exists {
		log.Warnf("task %s not found", name)
		return
	}

	select {
	case entry.updateCh <- interval:
		log.Infof("task %s interval updated to %v", name, interval)
	default:
		log.Warnf("task %s update channel full, skipping", name)
	}
}

// RUN 启动所有注册的任务
func RUN() {
	tasksMu.RLock()
	defer tasksMu.RUnlock()

	for _, entry := range tasks {
		go runTask(entry)
	}

	// 阻塞主协程
	select {}
}

func runTask(entry *taskEntry) {
	// 防御性检查：间隔必须大于0
	if entry.interval <= 0 {
		log.Warnf("task %s has non-positive interval, skipping", entry.name)
		return
	}

	// 根据配置决定是否在启动时立即执行
	if entry.runOnStart {
		go entry.fn()
	}

	entry.ticker = time.NewTicker(entry.interval)
	defer entry.ticker.Stop()

	for {
		select {
		case <-entry.ticker.C:
			go entry.fn()
		case newInterval := <-entry.updateCh:
			entry.ticker.Stop()
			entry.interval = newInterval
			entry.ticker = time.NewTicker(newInterval)
		case <-entry.stopCh:
			return
		}
	}
}
