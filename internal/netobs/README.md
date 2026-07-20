# Network Observation (netobs)

## Architecture

```
NetworkObserver (interface)
├── GoObserver      // always available, pure Go (L7 probe + EWMA)
└── EBPFObserver    // optional, //go:build ebpf
```

Selection at runtime via setting `net_obs_mode`:

| mode | behavior |
|------|----------|
| `auto` (default) | try eBPF → fallback Go |
| `go` | pure Go only |
| `ebpf` | force eBPF, fallback Go on failure |

## Default build (what ships today)

```bash
CGO_ENABLED=0 go build -o octopus .
```

- Uses `ebpf_stub.go` (`//go:build !ebpf`)
- `EBPFObserver.Available() == false`
- `auto` → **Go observer**
- Zero extra deps, works in unprivileged Docker

## Enable real eBPF

Requirements:
1. Linux host (not pure Docker netns without CAP_BPF)
2. `CAP_BPF` / `CAP_SYS_ADMIN` (or privileged)
3. `/sys/fs/bpf` mounted
4. clang + kernel headers for bpf2go
5. `go get github.com/cilium/ebpf`

```bash
# 1. generate BPF Go bindings
go generate -tags ebpf ./internal/netobs

# 2. build with tag
go build -tags ebpf -o octopus .

# 3. run with capability
sudo setcap cap_bpf,cap_perfmon,cap_net_admin=+ep ./octopus
# or: docker run --cap-add=BPF --cap-add=PERFMON --cap-add=NET_ADMIN ...
```

## Why interface + runtime?

- Keeps default binary simple (no cilium dep, no CGO)
- Same call sites for Go/eBPF (health score can later blend RTT)
- Matches your preferred "interface + runtime.GOOS/capability" pattern
