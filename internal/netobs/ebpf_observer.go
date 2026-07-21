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
	"golang.org/x/sys/unix"
)

// EBPFObserver 内核态观测后端（-tags ebpf 时编译）。
// 依赖：github.com/cilium/ebpf、CAP_BPF/CAP_PERFMON、/sys/fs/bpf、BTF。
//
// P0 探针：
//   - sys_enter_connect / sys_exit_connect：全局 connect_hits + per-host 建连成败/时延
//   - ObserveChannel 将渠道 baseURL 解析为 IPv4 host key，readLoop 回填
//
// P1 探针（sockops）：
//   - BPF_CGROUP_SOCK_OPS：srtt_us（RTT）+ retrans + async_fails
//   - 解决非阻塞 connect 无法测量 RTT / 异步失败的问题
//
// 无样本时返回 0 → ChannelScore 零影响，与纯 Go 行为一致。

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall" tcpConn ./bpf/tcp_conn.c -- -I/usr/include
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall" -target bpf tcpConnSockops ./bpf/tcp_conn_sockops.c -- -I/usr/include

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
	coll    *ebpf.Collection  // P0: tracepoint connect
	sockColl *ebpf.Collection // P1: sockops

	// channelID -> host keys
	channelKeys map[int][]HostKey
	// hostKey -> channelIDs
	keyChannels map[HostKey]map[int]struct{}

	// P0: tracepoint connect 数据
	rttStore      sync.Map // channelID -> float64 ms (blocking connect only)
	syscallFail   sync.Map // channelID -> float64 0..1  (sync connect fail rate)

	// P1: sockops 数据
	srttStore    sync.Map // channelID -> float64 ms (srtt_us/1000)
	retransRate  sync.Map // channelID -> float64 0..1  (retrans / samples)
	asyncFail    sync.Map // channelID -> float64 0..1  (async_fails / samples)
	// Combined fail rate for ChannelFailRate: max(syscall, async) or weighted
	failRate     sync.Map // channelID -> float64 0..1

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

	// ── P0: tracepoint connect ──────────────────────────────────
	spec, err := loadTcpConn()
	if err != nil {
		return fmt.Errorf("ebpf: load spec: %w", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("ebpf: new collection: %w", err)
	}
	e.coll = coll

	// 挂载 P0 tracepoint
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

	// ── P1: sockops ──────────────────────────────────────────────
	// Soft-fail: if sockops attach fails, keep P0 running.
	if err := e.startSockops(); err != nil {
		fmt.Fprintf(os.Stderr, "netobs: sockops attach failed (P0 still active): %v\n", err)
	}

	e.started = true
	e.lastRead = time.Now()
	go e.readLoop()
	return nil
}

// startSockops loads and attaches the sockops program to the container's cgroupv2.
// Must be called with e.mu held.
func (e *EBPFObserver) startSockops() error {
	sockSpec, err := loadTcpConnSockops()
	if err != nil {
		return fmt.Errorf("load sockops spec: %w", err)
	}
	sockColl, err := ebpf.NewCollection(sockSpec)
	if err != nil {
		return fmt.Errorf("new sockops collection: %w", err)
	}

	sockProg := sockColl.Programs["handle_sockops"]
	if sockProg == nil {
		sockColl.Close()
		return fmt.Errorf("program handle_sockops not found")
	}

	cgroupPath := "/sys/fs/cgroup"
	var st unix.Statfs_t
	if err := unix.Statfs(cgroupPath, &st); err != nil {
		sockColl.Close()
		return fmt.Errorf("statfs cgroup: %w", err)
	}
	if st.Type != unix.CGROUP2_SUPER_MAGIC {
		cgroupPath = "/sys/fs/cgroup/unified"
	}

	lSock, err := link.AttachCgroup(link.CgroupOptions{
		Path:    cgroupPath,
		Program: sockProg,
		Attach:  ebpf.AttachCGroupSockOps,
	})
	if err != nil {
		sockColl.Close()
		return fmt.Errorf("attach cgroup sockops: %w", err)
	}

	e.sockColl = sockColl
	e.links = append(e.links, lSock)
	return nil
}

func (e *EBPFObserver) readLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		e.mu.Lock()
		active := e.started
		coll := e.coll
		sockColl := e.sockColl
		e.mu.Unlock()
		if !active {
			return
		}

		// P0: connect_hits
		if coll != nil {
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
			// P0: host_stats
			if m := coll.Maps["host_stats"]; m != nil {
				e.consumeHostStats(m)
			}
		}

		// P1: srtt_map
		if sockColl != nil {
			if m := sockColl.Maps["srtt_map"]; m != nil {
				e.consumeSrttMap(m)
			}
		}

		<-ticker.C
	}
}

func (e *EBPFObserver) consumeSrttMap(m *ebpf.Map) {
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

	// srtt_map value layout (must match bpf/tcp_conn_sockops.c conn_stats):
	//   srtt_us, count, retrans, async_fails  (all u64)
	// key: same as host_stats (daddr LE, dport LE as BE16-as-LE).
	type srttKey struct {
		Daddr uint32
		Dport uint16
		Pad   uint16
	}
	type srttVal struct {
		SrttUs     uint64
		Count      uint64
		Retrans    uint64
		AsyncFails uint64
	}

	type agg struct {
		srttSum    float64
		srttCnt    int
		retrans    uint64
		asyncFails uint64
		samples    uint64 // for rate denom: prefer count, else retrans+async+1
	}
	channelAgg := map[int]*agg{}

	var k srttKey
	var v srttVal
	it := m.Iterate()
	for it.Next(&k, &v) {
		hk := HostKey{Daddr: ntohl(k.Daddr), Dport: ntohs(k.Dport)}
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
			if v.SrttUs > 0 {
				a.srttSum += float64(v.SrttUs) / 1000.0
				a.srttCnt++
			}
			a.retrans += v.Retrans
			a.asyncFails += v.AsyncFails
			if v.Count > a.samples {
				a.samples = v.Count
			}
		}
	}
	_ = it.Err()

	for id, a := range channelAgg {
		if a.srttCnt > 0 {
			e.srttStore.Store(id, a.srttSum/float64(a.srttCnt))
		}
		// Denominators:
		//   retrans rate ≈ retrans / max(rtt_samples, 1)  (relative path quality)
		//   async fail  ≈ async_fails / (rtt_samples + async_fails)
		// rtt samples ≈ successful established connections that got RTT.
		samples := a.samples
		if samples == 0 && (a.retrans > 0 || a.asyncFails > 0) {
			// no RTT yet but had events — use event count as soft denom
			samples = a.retrans + a.asyncFails
		}
		if samples > 0 || a.asyncFails > 0 {
			if samples > 0 {
				rr := float64(a.retrans) / float64(samples)
				if rr > 1 {
					rr = 1
				}
				e.retransRate.Store(id, rr)
			}
			afDenom := samples + a.asyncFails
			if afDenom == 0 {
				afDenom = a.asyncFails
			}
			if afDenom > 0 {
				ar := float64(a.asyncFails) / float64(afDenom)
				if ar > 1 {
					ar = 1
				}
				e.asyncFail.Store(id, ar)
			}
		}

		// Combined fail rate for ChannelFailRate:
		// max(syscallFail, asyncFail) — either is a real path problem.
		var sysF, asyncF float64
		if v, ok := e.syscallFail.Load(id); ok {
			sysF = v.(float64)
		}
		if v, ok := e.asyncFail.Load(id); ok {
			asyncF = v.(float64)
		}
		combined := sysF
		if asyncF > combined {
			combined = asyncF
		}
		// Only store if we have any signal for this channel
		if a.srttCnt > 0 || a.retrans > 0 || a.asyncFails > 0 || sysF > 0 {
			e.failRate.Store(id, combined)
		}
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
			e.syscallFail.Store(id, rate)
			// Refresh combined failRate if async already known
			asyncF := 0.0
			if v, ok := e.asyncFail.Load(id); ok {
				asyncF = v.(float64)
			}
			combined := rate
			if asyncF > combined {
				combined = asyncF
			}
			e.failRate.Store(id, combined)
		}
		if a.latCnt > 0 {
			ms := float64(a.latSum) / float64(a.latCnt) / 1e6
			e.rttStore.Store(id, ms)
		} else if a.connects > 0 && a.latCnt == 0 {
			e.rttStore.Store(id, float64(0))
		}
	}
}

func ntohs(v uint16) uint16 {
	return (v>>8)&0xff | (v&0xff)<<8
}

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
	if e.sockColl != nil {
		e.sockColl.Close()
		e.sockColl = nil
	}
	e.started = false
	return nil
}

func (e *EBPFObserver) ChannelRTTMS(channelID int) float64 {
	// 优先级：P1 sockops srtt > P0 tracepoint latency
	if v, ok := e.srttStore.Load(channelID); ok {
		return v.(float64)
	}
	v, ok := e.rttStore.Load(channelID)
	if !ok {
		return 0
	}
	return v.(float64)
}

func (e *EBPFObserver) ChannelRetransRate(channelID int) float64 {
	// 真·TCP 重传率（RETRANS_CB / samples）
	v, ok := e.retransRate.Load(channelID)
	if !ok {
		return 0
	}
	return v.(float64)
}

func (e *EBPFObserver) ChannelFailRate(channelID int) float64 {
	// 建连失败率：max(syscall sync fail, async SYN_SENT→CLOSE)
	v, ok := e.failRate.Load(channelID)
	if !ok {
		return 0
	}
	return v.(float64)
}

func (e *EBPFObserver) ChannelHasSample(channelID int) bool {
	if _, ok := e.failRate.Load(channelID); ok {
		return true
	}
	if _, ok := e.syscallFail.Load(channelID); ok {
		return true
	}
	if _, ok := e.rttStore.Load(channelID); ok {
		return true
	}
	if _, ok := e.srttStore.Load(channelID); ok {
		return true
	}
	if _, ok := e.retransRate.Load(channelID); ok {
		return true
	}
	return false
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