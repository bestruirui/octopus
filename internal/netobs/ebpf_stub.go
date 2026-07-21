//go:build !ebpf

package netobs

import "fmt"

// EBPFObserver stub：未用 -tags ebpf 编译时的占位实现。
// auto 模式下 Available() 返回 false → 自动回落到 GoObserver。

type EBPFObserver struct{}

func NewEBPFObserver() *EBPFObserver {
	return &EBPFObserver{}
}

func (e *EBPFObserver) Name() string { return "ebpf" }

func (e *EBPFObserver) Available() bool { return false }

func (e *EBPFObserver) Start() error {
	return fmt.Errorf("eBPF observer not compiled: rebuild with -tags ebpf")
}

func (e *EBPFObserver) Stop() error { return nil }

func (e *EBPFObserver) ChannelRTTMS(_ int) float64       { return 0 }
func (e *EBPFObserver) ChannelRetransRate(_ int) float64 { return 0 }
func (e *EBPFObserver) ChannelFailRate(_ int) float64    { return 0 }
func (e *EBPFObserver) ChannelHasSample(_ int) bool      { return false }
func (e *EBPFObserver) ObserveChannel(_ int, _ string)   {}
func (e *EBPFObserver) Active() bool                     { return false }
func (e *EBPFObserver) ConnectHits() uint64              { return 0 }
