package netobs

import (
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
