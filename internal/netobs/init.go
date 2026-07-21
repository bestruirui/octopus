package netobs

import (
	"context"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/utils/log"
)

// InitNetObserver 按 setting net_obs_mode 初始化网络观测后端。
//
//	auto : 优先 eBPF，失败回落 Go（默认）
//	go   : 纯 Go
//	ebpf : 强制 eBPF，失败仍回落 Go（避免启动失败）
//
// 默认构建不含 eBPF 实现（//go:build ebpf），Available()=false，auto 会走 Go。
// 要启用内核态：go build -tags ebpf（需 CAP_BPF + /sys/fs/bpf + cilium/ebpf）。
func InitNetObserver() {
	mode := "auto"
	if v, err := op.SettingGetString(model.SettingKeyNetObsMode); err == nil && v != "" {
		mode = strings.ToLower(strings.TrimSpace(v))
	}
	currentMode = mode

	var obs NetworkObserver

	tryEBPF := func() NetworkObserver {
		e := NewEBPFObserver()
		if !e.Available() {
			return nil
		}
		if err := e.Start(); err != nil {
			log.Warnf("netobs: eBPF start failed: %v", err)
			return nil
		}
		return e
	}

	switch mode {
	case "go":
		obs = NewGoObserver()
		log.Infof("netobs: using Go observer (mode=go)")

	case "ebpf":
		if e := tryEBPF(); e != nil {
			obs = e
			log.Infof("netobs: using eBPF observer (mode=ebpf)")
		} else {
			obs = NewGoObserver()
			log.Warnf("netobs: eBPF forced but unavailable — falling back to Go")
		}

	default: // auto
		if e := tryEBPF(); e != nil {
			obs = e
			log.Infof("netobs: using eBPF observer (mode=auto)")
		} else {
			obs = NewGoObserver()
			log.Infof("netobs: eBPF not available — using Go observer (mode=auto)")
		}
	}

	// tryEBPF 已 Start；GoObserver 需在此 Start
	if !obs.Active() {
		if err := obs.Start(); err != nil {
			log.Errorf("netobs: observer start failed: %v", err)
		}
	}
	SetObserver(obs)

	// 注册已启用的渠道上游地址供 eBPF 关联（纯 Go 后端 ObserveChannel 无害，仅记录）
	syncChannelsWithObserver(obs)
}

// syncChannelsWithObserver 将当前缓存中已启用的渠道注册到观测后端。
// Repeatable: 纯 Go 后端 ObserveChannel 仅记录；eBPF 后端解析 host 并关联 map。
func syncChannelsWithObserver(obs NetworkObserver) {
	if obs == nil {
		return
	}
	channels, err := op.ChannelList(context.Background())
	if err != nil {
		return
	}
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		if base := ch.GetBaseUrl(); base != "" {
			obs.ObserveChannel(ch.ID, base)
		}
	}
	log.Infof("netobs: registered %d enabled channels", len(channels))
}

// SyncChannels refreshes channel → observer registration. Call after channel
// create/update/refresh. Safe no-op if observer not initialized.
func SyncChannels() {
	obs := GetObserver()
	if obs == nil {
		return
	}
	syncChannelsWithObserver(obs)
}

// StopNetObserver 关闭当前网络观测后端
func StopNetObserver() error {
	if obs := GetObserver(); obs != nil {
		return obs.Stop()
	}
	return nil
}

// BackendName 返回当前活跃后端名（go/ebpf/none）
func BackendName() string {
	if obs := GetObserver(); obs != nil {
		return obs.Name()
	}
	return "none"
}
