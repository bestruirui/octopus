// Minimal eBPF C skeleton for TCP connect / state observation.
// Compile via bpf2go (go generate) when building with -tags ebpf.
//
// This is intentionally small — attach to sock:inet_sock_set_state
// to observe TCP handshake RTT proxies and retransmit counters later.

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

// Placeholder map: channel_id -> rtt_us (filled by userspace correlation later)
struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 1024);
	__type(key, __u32);
	__type(value, __u64);
} rtt_map SEC(".maps");

SEC("tp/sock/inet_sock_set_state")
int tcp_connect(struct trace_event_raw_inet_sock_set_state *ctx)
{
	// Skeleton only: real implementation would filter new connections,
	// sample RTT from tcp_sock, and write into rtt_map.
	// Userspace reads the map and maps IP → channel_id.
	return 0;
}
