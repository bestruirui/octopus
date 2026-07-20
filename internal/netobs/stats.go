package netobs

// Status 当前网络观测后端状态（给 API / 前端展示）
type Status struct {
	Mode        string `json:"mode"`         // auto|go|ebpf
	Backend     string `json:"backend"`      // go|ebpf|none
	Active      bool   `json:"active"`
	ConnectHits uint64 `json:"connect_hits"` // eBPF 探针计数（go 后端为 0）
}

// currentMode 在 Init 时写入
var currentMode = "auto"

// GetStatus 汇总当前观测状态
func GetStatus() Status {
	st := Status{
		Mode:    currentMode,
		Backend: "none",
		Active:  false,
	}
	obs := GetObserver()
	if obs == nil {
		return st
	}
	st.Backend = obs.Name()
	st.Active = obs.Active()
	st.ConnectHits = obs.ConnectHits()
	return st
}
