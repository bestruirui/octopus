// Per-destination connect quality probe (P0).
// Classic headers only (no vmlinux.h) for Arch / container clang builds.
//
// Signals:
//   - connect_hits: global sys_enter_connect counter (badge heartbeat)
//   - host_stats: per IPv4 daddr:dport connects/fails + connect latency
//
// IPv6 ignored in P0 (can be layered later without changing Go interface).

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

#ifndef AF_INET
#define AF_INET 2
#endif

struct host_key {
	__u32 daddr; /* network byte order */
	__u16 dport; /* network byte order */
	__u16 pad;
};

struct host_stat {
	__u64 connects;
	__u64 fails;
	__u64 latency_sum_ns;
	__u64 latency_count;
};

struct inflight_val {
	struct host_key key;
	__u64 ts_ns;
};

/* Minimal sockaddr_in (no libc headers inside BPF). */
struct so_in {
	__u16 family;
	__u16 port;
	__u32 addr;
	__u8 zero[8];
};

/* Tracepoint arg layouts for syscalls/sys_enter_connect & sys_exit_connect. */
struct enter_connect_args {
	unsigned long long unused;
	long syscall_nr;
	unsigned long fd;
	unsigned long uservaddr;
	unsigned long addrlen;
};

struct exit_connect_args {
	unsigned long long unused;
	long syscall_nr;
	long ret;
};

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, __u64);
} connect_hits SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 1024);
	__type(key, struct host_key);
	__type(value, struct host_stat);
} host_stats SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 4096);
	__type(key, __u64); /* pid_tgid */
	__type(value, struct inflight_val);
} inflight SEC(".maps");

static __always_inline void bump_connect_hits(void)
{
	__u32 key = 0;
	__u64 *val = bpf_map_lookup_elem(&connect_hits, &key);
	if (val) {
		__sync_fetch_and_add(val, 1);
	} else {
		__u64 one = 1;
		bpf_map_update_elem(&connect_hits, &key, &one, BPF_ANY);
	}
}

static __always_inline int read_ipv4_dest(unsigned long uservaddr, unsigned long addrlen, struct host_key *out)
{
	struct so_in sa = {};

	if (addrlen < 8)
		return -1;
	if (bpf_probe_read_user(&sa, sizeof(sa), (void *)uservaddr))
		return -1;
	if (sa.family != AF_INET)
		return -1;
	out->daddr = sa.addr;
	out->dport = sa.port;
	out->pad = 0;
	return 0;
}

SEC("tracepoint/syscalls/sys_enter_connect")
int handle_connect_enter(struct enter_connect_args *ctx)
{
	struct host_key hk = {};
	struct inflight_val iv = {};
	__u64 pid_tgid;

	bump_connect_hits();

	if (read_ipv4_dest(ctx->uservaddr, ctx->addrlen, &hk))
		return 0;

	pid_tgid = bpf_get_current_pid_tgid();
	iv.key = hk;
	iv.ts_ns = bpf_ktime_get_ns();
	bpf_map_update_elem(&inflight, &pid_tgid, &iv, BPF_ANY);
	return 0;
}

SEC("tracepoint/syscalls/sys_exit_connect")
int handle_connect_exit(struct exit_connect_args *ctx)
{
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	struct inflight_val *iv;
	struct host_stat *st;
	struct host_stat init = {};
	__u64 now;
	__u64 dt;

	iv = bpf_map_lookup_elem(&inflight, &pid_tgid);
	if (!iv)
		return 0;

	st = bpf_map_lookup_elem(&host_stats, &iv->key);
	if (!st) {
		init.connects = 0;
		init.fails = 0;
		init.latency_sum_ns = 0;
		init.latency_count = 0;
		bpf_map_update_elem(&host_stats, &iv->key, &init, BPF_NOEXIST);
		st = bpf_map_lookup_elem(&host_stats, &iv->key);
		if (!st) {
			bpf_map_delete_elem(&inflight, &pid_tgid);
			return 0;
		}
	}

	__sync_fetch_and_add(&st->connects, 1);
	if (ctx->ret < 0)
		__sync_fetch_and_add(&st->fails, 1);

	now = bpf_ktime_get_ns();
	if (now > iv->ts_ns) {
		dt = now - iv->ts_ns;
		/* ignore absurd outliers (> 30s) */
		if (dt < 30000000000ULL) {
			__sync_fetch_and_add(&st->latency_sum_ns, dt);
			__sync_fetch_and_add(&st->latency_count, 1);
		}
	}

	bpf_map_delete_elem(&inflight, &pid_tgid);
	return 0;
}
