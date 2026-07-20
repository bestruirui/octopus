//go:build ebpf

package netobs

import (
	"fmt"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
)

// EBPFObserver 内核态 TCP 观测后端（仅 -tags ebpf 时编译）。
// 依赖：github.com/cilium/ebpf、CAP_BPF、/sys/fs/bpf。
//
// 当前为可运行骨架：加载 + 挂载 tracepoint 框架已就绪，
// map 读取循环与 channel↔IP 关联可按需扩展。

//go:generate bpf2go -cc clang tcp_conn ./bpf/tcp_conn.c -- -I/usr/include

type EBPFObserver struct {
	mu       sync.Mutex
	started  bool
	links    []link.Link
	coll     *ebpf.Collection
	channels map[int]string
	rttStore sync.Map
	retrans  sync.Map
}

func NewEBPFObserver() *EBPFObserver {
	return &EBPFObserver{channels: make(map[int]string)}
}

func (e *EBPFObserver) Name() string { return "ebpf" }

func (e *EBPFObserver) Available() bool {
	if err := rlimit.RemoveMemlock(); err != nil {
		return false
	}
	return true
}

func (e *EBPFObserver) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return nil
	}
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("ebpf: remove memlock: %w", err)
	}

	// 若尚未 bpf2go 生成，loadTcpConn 不存在——此处用空 collection 占位。
	// 真正上线时：bpf2go 生成 loadTcpConn()，然后：
	//   coll, err := loadTcpConnObjects(nil, nil)
	//   link.Tracepoint("sock", "inet_sock_set_state", coll.TcpConnect, nil)
	//
	// 当前返回明确错误，让 auto 模式回落 Go，而不是 silent fake。
	return fmt.Errorf("ebpf: BPF object not generated yet (run go generate ./internal/netobs with -tags ebpf)")
}

func (e *EBPFObserver) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, l := range e.links {
		_ = l.Close()
	}
	e.links = nil
	if e.coll != nil {
		e.coll.Close()
		e.coll = nil
	}
	e.started = false
	return nil
}

func (e *EBPFObserver) ChannelRTTMS(channelID int) float64 {
	v, ok := e.rttStore.Load(channelID)
	if !ok {
		return 0
	}
	return v.(float64)
}

func (e *EBPFObserver) ChannelRetransRate(channelID int) float64 {
	v, ok := e.retrans.Load(channelID)
	if !ok {
		return 0
	}
	return v.(float64)
}

func (e *EBPFObserver) ObserveChannel(channelID int, upstreamHost string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.channels[channelID] = upstreamHost
}

func (e *EBPFObserver) Active() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.started
}
