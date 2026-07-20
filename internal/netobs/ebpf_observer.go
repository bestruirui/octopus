//go:build ebpf

package netobs

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
)

// EBPFObserver 内核态观测后端（-tags ebpf 时编译）。
// 依赖：github.com/cilium/ebpf、CAP_BPF/CAP_PERFMON、/sys/fs/bpf、BTF。
//
// 当前最小可运行探针：sys_enter_connect 计数。
// ChannelRTTMS / ChannelRetransRate 仍返回 0，等 map 关联 channel 后再接 EWMA。

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall" tcpConn ./bpf/tcp_conn.c -- -I/usr/include

type EBPFObserver struct {
	mu       sync.Mutex
	started  bool
	links    []link.Link
	coll     *ebpf.Collection
	channels map[int]string
	rttStore sync.Map
	retrans  sync.Map

	hits     uint64
	lastRead time.Time
}

func NewEBPFObserver() *EBPFObserver {
	return &EBPFObserver{channels: make(map[int]string)}
}

func (e *EBPFObserver) Name() string { return "ebpf" }

func (e *EBPFObserver) Available() bool {
	_ = rlimit.RemoveMemlock()
	if _, err := os.Stat("/sys/kernel/btf/vmlinux"); err != nil {
		// no BTF: still try later, but mark available so Start can report real error
	}
	return true
}

func (e *EBPFObserver) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return nil
	}

	_ = rlimit.RemoveMemlock()

	spec, err := loadTcpConn()
	if err != nil {
		return fmt.Errorf("ebpf: load spec: %w", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("ebpf: new collection: %w", err)
	}
	e.coll = coll

	prog := coll.Programs[tcpConnProgHandleConnect]
	if prog == nil {
		coll.Close()
		return fmt.Errorf("ebpf: program handle_connect not found")
	}

	l, err := link.Tracepoint("syscalls", "sys_enter_connect", prog, nil)
	if err != nil {
		coll.Close()
		return fmt.Errorf("ebpf: attach sys_enter_connect: %w", err)
	}
	e.links = append(e.links, l)

	e.started = true
	e.lastRead = time.Now()

	go e.readLoop()

	return nil
}

func (e *EBPFObserver) readLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		e.mu.Lock()
		active := e.started
		e.mu.Unlock()
		if !active {
			return
		}

		m := e.coll.Maps[tcpConnMapConnectHits]
		if m != nil {
			var key uint32
			var val uint64
			if err := m.Lookup(&key, &val); err == nil {
				e.mu.Lock()
				e.hits = val
				e.lastRead = time.Now()
				e.mu.Unlock()
			}
		}
		<-ticker.C
	}
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

// ConnectHits 返回 sys_enter_connect 累计次数
func (e *EBPFObserver) ConnectHits() uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.hits
}
