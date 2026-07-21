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
// P0 探针：
//   - sys_enter_connect / sys_exit_connect：全局 connect_hits + per-host 建连成败/时延
//   - ObserveChannel 将渠道 baseURL 解析为 IPv4 host key，readLoop 回填
//     ChannelRTTMS / ChannelRetransRate（后者用 fail rate 作为 soft proxy）
//
// 无样本时返回 0 → ChannelScore 零影响，与纯 Go 行为一致。

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall" tcpConn ./bpf/tcp_conn.c -- -I/usr/include

// 与 bpf/tcp_conn.c 中 struct host_key / host_stat 布局一致。
type bpfHostKey struct {
	Daddr uint32
	Dport uint16
	Pad   uint16
}

type bpfHostStat struct {
	Connects     uint64
	Fails        uint64
	LatencySumNs uint64
	LatencyCount uint64
}

type EBPFObserver struct {
	mu      sync.Mutex
	started bool
	links   []link.Link
	coll    *ebpf.Collection

	// channelID -> host keys
	channelKeys map[int][]HostKey
	// hostKey -> channelIDs
	keyChannels map[HostKey]map[int]struct{}

	rttStore sync.Map // channelID -> float64 ms
	failRate sync.Map // channelID -> float64 0..1  (exposed as ChannelRetransRate)

	hits     uint64
	lastRead time.Time
}

func NewEBPFObserver() *EBPFObserver {
	return &EBPFObserver{
		channelKeys: make(map[int][]HostKey),
		keyChannels: make(map[HostKey]map[int]struct{}),
	}
}

func (e *EBPFObserver) Name() string { return "ebpf" }

func (e *EBPFObserver) Available() bool {
	_ = rlimit.RemoveMemlock()
	_, _ = os.Stat("/sys/kernel/btf/vmlinux")
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

	enterProg := coll.Programs["handle_connect_enter"]
	if enterProg == nil {
		coll.Close()
		return fmt.Errorf("ebpf: program handle_connect_enter not found")
	}
	exitProg := coll.Programs["handle_connect_exit"]
	if exitProg == nil {
		coll.Close()
		return fmt.Errorf("ebpf: program handle_connect_exit not found")
	}

	lEnter, err := link.Tracepoint("syscalls", "sys_enter_connect", enterProg, nil)
	if err != nil {
		coll.Close()
		return fmt.Errorf("ebpf: attach sys_enter_connect: %w", err)
	}
	lExit, err := link.Tracepoint("syscalls", "sys_exit_connect", exitProg, nil)
	if err != nil {
		_ = lEnter.Close()
		coll.Close()
		return fmt.Errorf("ebpf: attach sys_exit_connect: %w", err)
	}
	e.links = append(e.links, lEnter, lExit)

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
		coll := e.coll
		e.mu.Unlock()
		if !active || coll == nil {
			return
		}

		if m := coll.Maps["connect_hits"]; m != nil {
			var key uint32
			var val uint64
			if err := m.Lookup(&key, &val); err == nil {
				e.mu.Lock()
				e.hits = val
				e.lastRead = time.Now()
				e.mu.Unlock()
			}
		}

		if m := coll.Maps["host_stats"]; m != nil {
			e.consumeHostStats(m)
		}

		<-ticker.C
	}
}

func (e *EBPFObserver) consumeHostStats(m *ebpf.Map) {
	type agg struct {
		connects uint64
		fails    uint64
		latSum   uint64
		latCnt   uint64
	}

	e.mu.Lock()
	keyChannels := make(map[HostKey]map[int]struct{}, len(e.keyChannels))
	for k, v := range e.keyChannels {
		cp := make(map[int]struct{}, len(v))
		for id := range v {
			cp[id] = struct{}{}
		}
		keyChannels[k] = cp
	}
	e.mu.Unlock()

	channelAgg := map[int]*agg{}

	var key bpfHostKey
	var val bpfHostStat
	it := m.Iterate()
	for it.Next(&key, &val) {
		// BPF map stores daddr/dport in network byte order (big-endian) as it comes
		// from sockaddr_in. cilium/ebpf on x86 reads the raw bytes as little-endian
		// uint32/uint16, so we must ntohl/ntohs to get the canonical network-order
		// value that matches what Go's binary.BigEndian.Uint32 produces.
		hk := HostKey{Daddr: ntohl(key.Daddr), Dport: ntohs(key.Dport)}
		chans, ok := keyChannels[hk]
		if !ok {
			continue
		}
		for id := range chans {
			a := channelAgg[id]
			if a == nil {
				a = &agg{}
				channelAgg[id] = a
			}
			a.connects += val.Connects
			a.fails += val.Fails
			a.latSum += val.LatencySumNs
			a.latCnt += val.LatencyCount
		}
	}
	_ = it.Err()

	for id, a := range channelAgg {
		if a.connects > 0 {
			rate := float64(a.fails) / float64(a.connects)
			if rate < 0 {
				rate = 0
			}
			if rate > 1 {
				rate = 1
			}
			e.failRate.Store(id, rate)
		}
		if a.latCnt > 0 {
			ms := float64(a.latSum) / float64(a.latCnt) / 1e6
			e.rttStore.Store(id, ms)
		}
	}
}

func ntohs(v uint16) uint16 {
	return (v>>8)&0xff | (v&0xff)<<8
}

// ntohl converts a uint32 value from network byte order (big-endian) to host byte order.
// On little-endian x86, this is a byte swap. The BPF map stores daddr in network byte order,
// but cilium/ebpf reads map keys as native-endian, so we need to swap.
func ntohl(v uint32) uint32 {
	return (v >> 24) | ((v >> 8) & 0xff00) | ((v & 0xff00) << 8) | (v << 24)
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

// ChannelRetransRate returns connect fail rate [0,1] as a soft proxy for path
// quality. True TCP retransmit ratio can replace this later without interface change.
func (e *EBPFObserver) ChannelRetransRate(channelID int) float64 {
	v, ok := e.failRate.Load(channelID)
	if !ok {
		return 0
	}
	return v.(float64)
}

func (e *EBPFObserver) ObserveChannel(channelID int, upstreamHost string) {
	if channelID <= 0 {
		return
	}
	keys := ResolveHostKeys(upstreamHost)

	e.mu.Lock()
	defer e.mu.Unlock()

	if old, ok := e.channelKeys[channelID]; ok {
		for _, k := range old {
			if set, ok := e.keyChannels[k]; ok {
				delete(set, channelID)
				if len(set) == 0 {
					delete(e.keyChannels, k)
				}
			}
		}
	}
	e.channelKeys[channelID] = keys
	for _, k := range keys {
		set := e.keyChannels[k]
		if set == nil {
			set = make(map[int]struct{})
			e.keyChannels[k] = set
		}
		set[channelID] = struct{}{}
	}
}

func (e *EBPFObserver) Active() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.started
}

func (e *EBPFObserver) ConnectHits() uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.hits
}
