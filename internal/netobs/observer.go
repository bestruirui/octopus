package netobs

// NetworkObserver 网络观测抽象接口。
// 两种实现：
//   - GoObserver   : 纯应用层（L7 HTTP 探活 + EWMA），无内核依赖
//   - EBPFObserver : 内核态 TCP 探针（重传率/RTT），需要 CAP_BPF + /sys/fs/bpf
//
// 启动时按 SettingKeyNetObsMode 选择：
//   - auto : 优先 eBPF，加载失败回落 Go
//   - go   : 纯 Go
//   - ebpf : 强制 eBPF，加载失败报错（不回落）
type NetworkObserver interface {
	// Name 后端名称 "go" / "ebpf"
	Name() string

	// Available 检查后端是否可用（eBPF 需要内核支持 + 权限）
	Available() bool

	// Start 启动观测（eBPF 加载探针；Go 版无操作）
	Start() error

	// Stop 停止观测并释放资源
	Stop() error

	// ChannelRTTMS 返回渠道最近 RTT（毫秒），无数据返回 0
	ChannelRTTMS(channelID int) float64

	// ChannelRetransRate 返回渠道 TCP 重传率（0~1），无数据返回 0
	ChannelRetransRate(channelID int) float64

	// ObserveChannel 标记某渠道的上游地址（eBPF 按此过滤；Go 版忽略）
	ObserveChannel(channelID int, upstreamHost string)

	// Active 后端是否正在运行
	Active() bool
}

// 全局实例
var activeObserver NetworkObserver

// SetObserver 设置当前活跃的网络观测后端
func SetObserver(obs NetworkObserver) {
	activeObserver = obs
}

// GetObserver 获取当前活跃的网络观测后端
func GetObserver() NetworkObserver {
	return activeObserver
}
