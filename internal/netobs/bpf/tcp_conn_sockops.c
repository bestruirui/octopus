// SRTT + retrans + async-fail via sockops.
// P1 probe for Hermes Agent octopus.
// Uses linux/bpf.h (enums/flags from modern kernels).
//
// Statistics:
//   - srtt_us: smoothed RTT (from BPF_SOCK_OPS_RTT_CB)
//   - retrans: retransmission count (from BPF_SOCK_OPS_RETRANS_CB)
//   - async_fails: async connect failures (via BPF_SOCK_OPS_STATE_CB:
//     SYN_SENT → CLOSE without ESTABLISHED)

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

#ifndef AF_INET
#define AF_INET 2
#endif

/* Same host_key as tcp_conn.c (daddr/dport in network byte order, LE on x86). */
struct host_key {
	__u32 daddr;
	__u16 dport;
	__u16 pad;
};

/* Per-destination connection statistics. */
struct conn_stats {
	__u64 srtt_us;      /* smoothed RTT in microseconds (>> 3) */
	__u64 count;         /* RTT sample count */
	__u64 retrans;       /* retransmission count */
	__u64 async_fails;   /* async connect failures (SYN_SENT→CLOSE) */
};

/* Map: dest host_key → conn_stats */
struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 1024);
	__type(key, struct host_key);
	__type(value, struct conn_stats);
} srtt_map SEC(".maps");

/* Per-socket inflight tracking for async connect failures.
 * Key: socket cookie (u64, from bpf_get_socket_cookie).
 * Value: destination host_key + timestamp of SYN_SENT. */
struct async_inflight {
	struct host_key key;
	__u64 ts_ns;
};

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 4096);
	__type(key, __u64);
	__type(value, struct async_inflight);
} inflight_map SEC(".maps");

/* Callback flags to enable: RTT + RETRANS + STATE */
#define CB_FLAGS (BPF_SOCK_OPS_RTT_CB_FLAG | BPF_SOCK_OPS_RETRANS_CB_FLAG | BPF_SOCK_OPS_STATE_CB_FLAG)

/* TCP state values from bpf_tcp_state (linux/bpf.h) */
#ifndef TCP_ESTABLISHED
#define TCP_ESTABLISHED 1
#define TCP_SYN_SENT    2
#define TCP_CLOSE       7
#endif

static __always_inline void enable_callbacks(struct bpf_sock_ops *skops)
{
	if (!skops)
		return;
	/* Enable RTT / RETRANS / STATE callbacks for this socket.
	 * Must be set as early as possible (TCP_CONNECT_CB preferred)
	 * so STATE_CB can see SYN_SENT → CLOSE for async fails. */
	bpf_sock_ops_cb_flags_set(skops, CB_FLAGS);
}

/* ── RTT callback ─────────────────────────────────────────────── */

static __always_inline void on_rtt(struct bpf_sock_ops *skops)
{
	struct host_key k = {};
	struct conn_stats *e;
	struct conn_stats init = {};
	__u32 srtt;

	if (!skops || skops->family != AF_INET)
		return;

	/* skops->srtt_us is (actual_us << 3) */
	srtt = skops->srtt_us >> 3;
	if (srtt == 0)
		return;

	/* Key format: daddr = remote_ip4 (network byte order, LE on x86),
	 * dport = upper 16 bits of remote_port (network byte order port). */
	k.daddr = skops->remote_ip4;
	k.dport = (__u16)(skops->remote_port >> 16);
	k.pad = 0;

	e = bpf_map_lookup_elem(&srtt_map, &k);
	if (!e) {
		init.srtt_us = srtt;
		init.count = 1;
		init.retrans = 0;
		init.async_fails = 0;
		bpf_map_update_elem(&srtt_map, &k, &init, BPF_ANY);
		return;
	}
	/* EWMA: 0.25*new + 0.75*old */
	e->srtt_us = (srtt >> 2) + ((e->srtt_us * 3) >> 2);
	e->count += 1;
}

/* ── Retransmission callback ───────────────────────────────────── */

static __always_inline void on_retrans(struct bpf_sock_ops *skops)
{
	struct host_key k = {};
	struct conn_stats *e;
	struct conn_stats init = {};

	if (!skops || skops->family != AF_INET)
		return;

	k.daddr = skops->remote_ip4;
	k.dport = (__u16)(skops->remote_port >> 16);
	k.pad = 0;

	e = bpf_map_lookup_elem(&srtt_map, &k);
	if (!e) {
		init.srtt_us = 0;
		init.count = 0;
		init.retrans = 1;
		init.async_fails = 0;
		bpf_map_update_elem(&srtt_map, &k, &init, BPF_ANY);
		return;
	}
	__sync_fetch_and_add(&e->retrans, 1);
}

/* ── State change callback (async connect failure detection) ───── */

static __always_inline void on_state_change(struct bpf_sock_ops *skops)
{
	__u32 old_state, new_state;
	struct host_key k = {};
	struct conn_stats *e;
	struct conn_stats init = {};
	struct async_inflight *iv;
	struct async_inflight new_iv = {};
	__u64 cookie;

	if (!skops || skops->family != AF_INET)
		return;

	old_state = skops->args[0];
	new_state = skops->args[1];

	k.daddr = skops->remote_ip4;
	k.dport = (__u16)(skops->remote_port >> 16);
	k.pad = 0;

	if (new_state == TCP_SYN_SENT) {
		/* Async connect started — record inflight. */
		cookie = bpf_get_socket_cookie(skops);
		if (cookie == 0)
			return;
		new_iv.key = k;
		new_iv.ts_ns = bpf_ktime_get_ns();
		bpf_map_update_elem(&inflight_map, &cookie, &new_iv, BPF_ANY);
		return;
	}

	if (new_state == TCP_ESTABLISHED) {
		/* Async connect succeeded — clean up inflight. */
		cookie = bpf_get_socket_cookie(skops);
		if (cookie)
			bpf_map_delete_elem(&inflight_map, &cookie);
		return;
	}

	if (new_state == TCP_CLOSE && old_state != TCP_ESTABLISHED) {
		/* Socket closing without having been established.
		 * Check if we were tracking it. */
		cookie = bpf_get_socket_cookie(skops);
		if (cookie == 0)
			return;
		iv = bpf_map_lookup_elem(&inflight_map, &cookie);
		if (!iv)
			return;
		/* This was an async connect that failed. */
		bpf_map_delete_elem(&inflight_map, &cookie);

		e = bpf_map_lookup_elem(&srtt_map, &k);
		if (!e) {
			init.srtt_us = 0;
			init.count = 0;
			init.retrans = 0;
			init.async_fails = 1;
			bpf_map_update_elem(&srtt_map, &k, &init, BPF_ANY);
			return;
		}
		__sync_fetch_and_add(&e->async_fails, 1);
	}
}

/* ── Main sockops handler ──────────────────────────────────────── */

SEC("sockops")
int handle_sockops(struct bpf_sock_ops *skops)
{
	switch (skops->op) {
	/* Earliest point for active connect — enable flags so STATE_CB
	 * can observe SYN_SENT → ESTABLISHED/CLOSE transitions. */
	case BPF_SOCK_OPS_TCP_CONNECT_CB:
		enable_callbacks(skops);
		break;
	case BPF_SOCK_OPS_ACTIVE_ESTABLISHED_CB:
	case BPF_SOCK_OPS_PASSIVE_ESTABLISHED_CB:
		/* Re-assert flags (harmless if already set). */
		enable_callbacks(skops);
		break;
	case BPF_SOCK_OPS_RTT_CB:
		on_rtt(skops);
		break;
	case BPF_SOCK_OPS_RETRANS_CB:
		on_retrans(skops);
		break;
	case BPF_SOCK_OPS_STATE_CB:
		on_state_change(skops);
		break;
	}
	return 0;
}