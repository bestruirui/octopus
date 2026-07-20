package netobs

import "sync"

// GoObserver 纯 Go 应用层观测后端。
// 不依赖内核，RTT/重传率为 0（由 balancer EWMA 覆盖延迟数据）。
// 这是 auto 模式下的 fallback 和 go 模式下的唯一后端。

type GoObserver struct {
	mu       sync.Mutex
	started  bool
	channels map[int]string // channelID -> upstreamHost（记录但不使用）
}

func NewGoObserver() *GoObserver {
	return &GoObserver{
		channels: make(map[int]string),
	}
}

func (g *GoObserver) Name() string { return "go" }

func (g *GoObserver) Available() bool { return true }

func (g *GoObserver) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.started = true
	return nil
}

func (g *GoObserver) Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.started = false
	return nil
}

func (g *GoObserver) ChannelRTTMS(channelID int) float64 {
	// Go 版无内核 RTT 数据；延迟由 balancer EWMA 被动测量覆盖
	return 0
}

func (g *GoObserver) ChannelRetransRate(channelID int) float64 {
	// Go 版无 TCP 重传统计
	return 0
}

func (g *GoObserver) ObserveChannel(channelID int, upstreamHost string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.channels[channelID] = upstreamHost
}

func (g *GoObserver) Active() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.started
}

func (g *GoObserver) ConnectHits() uint64 { return 0 }
