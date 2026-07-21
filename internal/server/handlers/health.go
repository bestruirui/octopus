package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/netobs"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/bestruirui/octopus/internal/task"
	"github.com/gin-gonic/gin"
)

func init() {
	// 健康状态 + 实时看板接口挂在 /api/v1/stats 下（复用 Auth）
	router.NewGroupRouter("/api/v1/stats").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/health", http.MethodGet).
				Handle(getChannelHealth),
		).
		AddRoute(
			router.NewRoute("/health/probe", http.MethodPost).
				Handle(triggerHealthProbe),
		).
		AddRoute(
			router.NewRoute("/realtime", http.MethodGet).
				Handle(getRealtimeDashboard),
		)
}

// getChannelHealth 返回所有渠道的探活/熔断健康状态 + 当前生效配置
func getChannelHealth(c *gin.Context) {
	list := balancer.ListChannelHealth()

	// 补齐尚未探活过的已启用渠道（标记 unknown）
	channels, err := op.ChannelList(c.Request.Context())
	if err == nil {
		known := make(map[int]struct{}, len(list))
		for _, h := range list {
			known[h.ChannelID] = struct{}{}
		}
		for _, ch := range channels {
			if _, ok := known[ch.ID]; ok {
				continue
			}
			list = append(list, balancer.ChannelHealth{
				ChannelID:   ch.ID,
				ChannelName: ch.Name,
				Status:      balancer.HealthUnknown,
				BaseURL:     ch.GetBaseUrl(),
			})
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].ChannelID < list[j].ChannelID
	})

	failThresh, degradeMS := balancer.GetHealthThresholds()
	method, _ := op.SettingGetString(model.SettingKeyHealthProbeMethod)
	interval, _ := op.SettingGetInt(model.SettingKeyHealthProbeInterval)
	enabled, _ := op.SettingGetBool(model.SettingKeyHealthProbeEnabled)
	tripOnFail, _ := op.SettingGetBool(model.SettingKeyHealthProbeTripOnFail)
	netSt := netobs.GetStatus()

	resp.Success(c, gin.H{
		"channels": list,
		"config": gin.H{
			"enabled":         enabled,
			"interval_sec":    interval,
			"method":          method,
			"fail_threshold":  failThresh,
			"degrade_ms":      degradeMS,
			"trip_on_fail":    tripOnFail,
			"net_obs_mode":    netSt.Mode,
			"net_obs_backend": netSt.Backend,
			"net_obs_active":  netSt.Active,
			"connect_hits":    netSt.ConnectHits,
		},
	})
}

// triggerHealthProbe 手动触发一次全量探活（异步），设置页/看板「立即探活」按钮用
func triggerHealthProbe(c *gin.Context) {
	go task.ChannelHealthProbeTask()
	resp.Success(c, gin.H{
		"message": "probe started",
		"at":      time.Now().Unix(),
	})
}

// realtimeDashboard 聚合今日 + 小时 + 渠道健康，给首页看板用
type realtimeDashboard struct {
	GeneratedAt int64                 `json:"generated_at"`
	Today       model.StatsDaily      `json:"today"`
	Hourly      []model.StatsHourly   `json:"hourly"`
	Total       model.StatsTotal      `json:"total"`
	Channels    []channelRealtimeItem `json:"channels"`
	Summary     realtimeSummary       `json:"summary"`
	NetObs      netobs.Status         `json:"net_obs"`
}

type channelRealtimeItem struct {
	ChannelID       int                     `json:"channel_id"`
	ChannelName     string                  `json:"channel_name"`
	Enabled         bool                    `json:"enabled"`
	Stats           model.StatsMetrics      `json:"stats"`
	Health          balancer.ChannelHealth  `json:"health"`
	SuccessRate     float64                 `json:"success_rate"`      // 0-100
	AvgLatencyMS    float64                 `json:"avg_latency_ms"`    // wait_time / request_count
	TotalCost       float64                 `json:"total_cost"`
	KernelRTTMS     float64                 `json:"kernel_rtt_ms"`     // eBPF 观测的建连时延（ms），无数据 = 0
	KernelFailRate  float64                 `json:"kernel_fail_rate"`  // eBPF 观测的建连失败率 [0,1]，无数据 = 0
	KernelStatus    string                  `json:"kernel_status"`     // good / slow / poor / idle
	KernelHint      string                  `json:"kernel_hint"`      // 人话结论
	KernelImpact    string                  `json:"kernel_impact"`    // 对选路的影响
}

type realtimeSummary struct {
	HealthyCount   int     `json:"healthy_count"`
	DegradedCount  int     `json:"degraded_count"`
	UnhealthyCount int     `json:"unhealthy_count"`
	UnknownCount   int     `json:"unknown_count"`
	SuccessRate    float64 `json:"success_rate"`
	AvgLatencyMS   float64 `json:"avg_latency_ms"`
	TotalCost      float64 `json:"total_cost"`
	RequestCount   int64   `json:"request_count"`
}

func getRealtimeDashboard(c *gin.Context) {
	today := op.StatsTodayGet()
	hourly := op.StatsHourlyGet()
	total := op.StatsTotalGet()

	channels, _ := op.ChannelList(c.Request.Context())
	items := make([]channelRealtimeItem, 0, len(channels))

	var sum realtimeSummary
	var totalWait int64
	var totalReq int64
	var totalSuccess int64

	for _, ch := range channels {
		stats := op.StatsChannelGet(ch.ID)
		health := balancer.GetChannelHealth(ch.ID)
		if health.ChannelName == "" {
			health.ChannelName = ch.Name
			health.ChannelID = ch.ID
		}

		reqCount := stats.RequestSuccess + stats.RequestFailed
		var successRate float64
		var avgLatency float64
		if reqCount > 0 {
			successRate = float64(stats.RequestSuccess) / float64(reqCount) * 100
			avgLatency = float64(stats.WaitTime) / float64(reqCount)
		}
		cost := stats.InputCost + stats.OutputCost

		// eBPF per-channel 内核指标 → 状态 + 人话结论 + 选路影响
		kRTT := netobs.GetObserver().ChannelRTTMS(ch.ID)
		kFail := netobs.GetObserver().ChannelRetransRate(ch.ID)
		kStatus, kHint, kImpact := kernelPathVerdict(kRTT, kFail)

		items = append(items, channelRealtimeItem{
			ChannelID:      ch.ID,
			ChannelName:    ch.Name,
			Enabled:        ch.Enabled,
			Stats:          stats.StatsMetrics,
			Health:         health,
			SuccessRate:    successRate,
			AvgLatencyMS:   avgLatency,
			TotalCost:      cost,
			KernelRTTMS:    kRTT,
			KernelFailRate: kFail,
			KernelStatus:   kStatus,
			KernelHint:     kHint,
			KernelImpact:   kImpact,
		})

		switch health.Status {
		case balancer.HealthHealthy:
			sum.HealthyCount++
		case balancer.HealthDegraded:
			sum.DegradedCount++
		case balancer.HealthUnhealthy:
			sum.UnhealthyCount++
		default:
			sum.UnknownCount++
		}

		totalWait += stats.WaitTime
		totalReq += reqCount
		totalSuccess += stats.RequestSuccess
		sum.TotalCost += cost
	}

	sum.RequestCount = totalReq
	if totalReq > 0 {
		sum.SuccessRate = float64(totalSuccess) / float64(totalReq) * 100
		sum.AvgLatencyMS = float64(totalWait) / float64(totalReq)
	}

	// 按费用降序
	sort.Slice(items, func(i, j int) bool {
		return items[i].TotalCost > items[j].TotalCost
	})

	resp.Success(c, realtimeDashboard{
		GeneratedAt: time.Now().Unix(),
		Today:       today,
		Hourly:      hourly,
		Total:       total,
		Channels:    items,
		Summary:     sum,
		NetObs:      netobs.GetStatus(),
	})
}

// kernelPathVerdict 把 eBPF 裸指标转成：状态 + 人话结论 + 选路影响
// 阈值对齐 ewma.go soft-bias 逻辑：
//   RTT  < 100ms  → 畅通
//   RTT  100-500  → 偏慢
//   RTT  > 500    → 拥堵
//   fail > 5%     → 不稳
//   fail > 20%    → 差
//   全 0          → idle（尚无样本 / 非 eBPF）
func kernelPathVerdict(rttMS, failRate float64) (status, hint, impact string) {
	if rttMS <= 0 && failRate <= 0 {
		return "idle", "暂无内核路径数据", "不影响选路"
	}

	// 失败率优先（比延迟更致命）
	if failRate >= 0.20 {
		return "poor",
			"上游建连经常失败，网络路径不稳",
			"系统已自动降低此渠道优先级"
	}
	if failRate >= 0.05 {
		return "slow",
			"上游偶有建连失败，路径略不稳定",
			"系统已轻微降低此渠道优先级"
	}

	// 按 RTT 分级
	switch {
	case rttMS >= 500:
		return "poor",
			"上游建连很慢，网络拥堵或跨洲链路",
			"系统已自动降低此渠道优先级"
	case rttMS >= 100:
		return "slow",
			"上游建连偏慢，高峰期可能更明显",
			"系统已轻微降低此渠道优先级"
	default:
		return "good",
			"上游网络畅通，建连快速稳定",
			"优先保留此渠道"
	}
}
