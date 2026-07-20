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
	if interval <= 0 {
		log.Debugf("task %s not registered: interval is 0", name)
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
// 当 interval 为 0 时，删除任务
func Update(name string, interval time.Duration) {
	tasksMu.Lock()
	entry, exists := tasks[name]
	if !exists {
		tasksMu.Unlock()
		log.Warnf("task %s not found", name)
		return
	}

	if interval <= 0 {
		delete(tasks, name)
		tasksMu.Unlock()
		close(entry.stopCh)
		log.Infof("task %s removed: interval is 0", name)
		return
	}
	tasksMu.Unlock()

	select {
	case entry.updateCh <- interval:
		log.Infof("task %s interval updated to %v", name, interval)
	default:
		log.Warnf("task %s update channel full, skipping", name)
	}
}

// UpdateOrRegister 热更新间隔；若任务不存在则重新注册并启动
// 用于探活开关从关→开、或进程启动时 disabled 后来又启用的场景
func UpdateOrRegister(name string, interval time.Duration, runOnStart bool, fn func()) {
	if interval <= 0 {
		Update(name, 0)
		return
	}

	tasksMu.RLock()
	_, exists := tasks[name]
	tasksMu.RUnlock()

	if exists {
		Update(name, interval)
		return
	}

	// 重新注册并立即启动
	Register(name, interval, runOnStart, fn)
	tasksMu.RLock()
	entry, ok := tasks[name]
	tasksMu.RUnlock()
	if ok {
		go runTask(entry)
		log.Infof("task %s re-registered with interval %v", name, interval)
	}
}

// RUN 启动所有注册的任务
func RUN() {
	tasksMu.RLock()
	for _, entry := range tasks {
		go runTask(entry)
	}
	tasksMu.RUnlock()

	// 阻塞主协程
	select {}
}

func runTask(entry *taskEntry) {
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
