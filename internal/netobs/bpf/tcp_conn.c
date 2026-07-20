// Minimal TCP-related observation skeleton.
// Uses classic headers (no vmlinux.h) so clang + libbpf headers work on Arch.
// Attaches to sys_enter_connect as a light proxy for outbound connection activity.
// Real RTT/retransmit maps can be layered later without changing the Go interface.

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

// Global connect counter — proves the probe is live.
struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, __u64);
} connect_hits SEC(".maps");

// Optional: per-hash host activity (userspace correlates channel later)
struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 1024);
	__type(key, __u32);
	__type(value, __u64);
} host_hits SEC(".maps");

SEC("tracepoint/syscalls/sys_enter_connect")
int handle_connect(void *ctx)
{
	__u32 key = 0;
	__u64 *val = bpf_map_lookup_elem(&connect_hits, &key);
	if (val) {
		__sync_fetch_and_add(val, 1);
	} else {
		__u64 one = 1;
		bpf_map_update_elem(&connect_hits, &key, &one, BPF_ANY);
	}
	return 0;
}
