package netobs

import (
	"encoding/binary"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// HostKey is an IPv4 destination key (addr in network byte order from IPv4
// big-endian bytes; port in host byte order).
type HostKey struct {
	Daddr uint32
	Dport uint16
}

// ParseUpstreamHost extracts host:port from a URL or bare host string.
// Defaults: https→443, http→80, else 443.
func ParseUpstreamHost(raw string) (host string, port uint16, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, false
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", 0, false
	}
	h := u.Hostname()
	if h == "" {
		return "", 0, false
	}
	p := u.Port()
	var portNum uint16
	if p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 || n > 65535 {
			return "", 0, false
		}
		portNum = uint16(n)
	} else {
		switch strings.ToLower(u.Scheme) {
		case "http":
			portNum = 80
		default:
			portNum = 443
		}
	}
	return h, portNum, true
}

// ResolveHostKeys resolves hostname to IPv4 HostKeys (best-effort).
// Empty result is fine — eBPF simply won't correlate until resolution works.
// Pure Go path may call this too; harmless if unused.
func ResolveHostKeys(raw string) []HostKey {
	host, port, ok := ParseUpstreamHost(raw)
	if !ok {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		v4 := ip.To4()
		if v4 == nil {
			return nil
		}
		return []HostKey{{Daddr: binary.BigEndian.Uint32(v4), Dport: port}}
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil
	}
	out := make([]HostKey, 0, len(ips))
	seen := map[HostKey]struct{}{}
	for _, ip := range ips {
		v4 := ip.To4()
		if v4 == nil {
			continue
		}
		k := HostKey{Daddr: binary.BigEndian.Uint32(v4), Dport: port}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

// SyncChannelURLs registers channelID → baseURL with the active observer.
// Safe no-op when observer is nil (startup race) or pure-Go backend.
func SyncChannelURLs(entries map[int]string) {
	obs := GetObserver()
	if obs == nil {
		return
	}
	for id, base := range entries {
		if id <= 0 || strings.TrimSpace(base) == "" {
			continue
		}
		obs.ObserveChannel(id, base)
	}
}
