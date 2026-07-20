package model

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type SettingKey string

const (
	SettingKeyProxyURL                SettingKey = "proxy_url"
	SettingKeyStatsSaveInterval       SettingKey = "stats_save_interval"          // 将统计信息写入数据库的周期(分钟)
	SettingKeyModelInfoUpdateInterval SettingKey = "model_info_update_interval"   // 模型信息更新间隔(小时)
	SettingKeySyncLLMInterval         SettingKey = "sync_llm_interval"            // LLM 同步间隔(小时)
	SettingKeyRelayLogKeepPeriod      SettingKey = "relay_log_keep_period"        // 日志保存时间范围(天)
	SettingKeyRelayLogKeepEnabled     SettingKey = "relay_log_keep_enabled"       // 是否保留历史日志
	SettingKeyCORSAllowOrigins        SettingKey = "cors_allow_origins"           // 跨域白名单(逗号分隔, 如 "example.com,example2.com"). 为空不允许跨域, "*"允许所有
	SettingKeyCircuitBreakerThreshold SettingKey = "circuit_breaker_threshold"    // 熔断触发阈值（连续失败次数）
	SettingKeyCircuitBreakerCooldown  SettingKey = "circuit_breaker_cooldown"     // 熔断基础冷却时间（秒）
	SettingKeyCircuitBreakerMaxCooldown SettingKey = "circuit_breaker_max_cooldown" // 熔断最大冷却时间（秒），指数退避上限

	// 渠道健康探活（全部前端可配，改完立即生效）
	SettingKeyHealthProbeEnabled       SettingKey = "health_probe_enabled"        // 是否启用主动探活 true/false
	SettingKeyHealthProbeInterval      SettingKey = "health_probe_interval"       // 探活间隔（秒）
	SettingKeyHealthProbeTimeout       SettingKey = "health_probe_timeout"        // 单次探活超时（秒）
	SettingKeyHealthProbeMethod        SettingKey = "health_probe_method"         // auto|models|head|chat|custom
	SettingKeyHealthProbePath          SettingKey = "health_probe_path"           // custom 方法时的相对路径，如 /v1/chat/completions
	// 注意：chat 探活模型名不再用全局 setting——直接取渠道 Model/CustomModel 列表，避免与渠道编辑重复
	SettingKeyHealthProbeFailThreshold SettingKey = "health_probe_fail_threshold" // 连续失败多少次才判 unhealthy
	SettingKeyNetObsMode               SettingKey = "net_obs_mode"               // auto|go|ebpf 网络观测后端
	SettingKeyHealthProbeDegradeMS     SettingKey = "health_probe_degrade_ms"     // 延迟超过多少 ms 判 degraded
	SettingKeyHealthProbeTripOnFail    SettingKey = "health_probe_trip_on_fail"   // 探活失败是否强制打开熔断 true/false
	SettingKeyHealthDashboardRefresh   SettingKey = "health_dashboard_refresh"    // 前端看板刷新间隔（秒）

	// 流式内存优化
	SettingKeyStreamLogMaxEvents SettingKey = "stream_log_max_events" // 流式日志最多保留事件数（0=不保留完整流，仅 usage）
	SettingKeyStreamLogMaxBytes  SettingKey = "stream_log_max_bytes"  // 流式日志最多保留字节

	// 健康分选路（被动 EWMA，不新增分组模式，只在现有候选上重排）
	SettingKeyHealthScoreRouting SettingKey = "health_score_routing" // true/false

	// 轻量语义路由（规则复杂度，不嵌 embedding）
	SettingKeySemanticRouteEnabled SettingKey = "semantic_route_enabled" // true/false

	// 混沌工程
	SettingKeyChaosEnabled   SettingKey = "chaos_enabled"    // true/false
	SettingKeyChaosDelayMS   SettingKey = "chaos_delay_ms"   // 注入延迟 ms
	SettingKeyChaosErrorRate SettingKey = "chaos_error_rate" // 0-100 百分比，随机返回 503
	SettingKeyChaosDropRate  SettingKey = "chaos_drop_rate"  // 0-100 百分比，直接断开（无响应体）
)

// 探活方式常量
const (
	HealthProbeMethodAuto   = "auto"   // 智能级联：models → head → base
	HealthProbeMethodModels = "models" // GET /v1/models（标准 OpenAI 兼容）
	HealthProbeMethodHead   = "head"   // HEAD/GET base_url（最轻量，兼容不返回 models 的中转）
	HealthProbeMethodChat   = "chat"   // POST 最小 chat/completions（最准，适合 sub2api 号池）
	HealthProbeMethodCustom = "custom" // 自定义路径
)

type Setting struct {
	Key   SettingKey `json:"key" gorm:"primaryKey"`
	Value string     `json:"value" gorm:"not null"`
}

func DefaultSettings() []Setting {
	return []Setting{
		{Key: SettingKeyProxyURL, Value: ""},
		{Key: SettingKeyStatsSaveInterval, Value: "10"},          // 默认10分钟保存一次统计信息
		{Key: SettingKeyCORSAllowOrigins, Value: ""},             // CORS 默认不允许跨域，设置为 "*" 才允许所有来源
		{Key: SettingKeyModelInfoUpdateInterval, Value: "24"},    // 默认24小时更新一次模型信息
		{Key: SettingKeySyncLLMInterval, Value: "24"},            // 默认24小时同步一次LLM
		{Key: SettingKeyRelayLogKeepPeriod, Value: "7"},          // 默认日志保存7天
		{Key: SettingKeyRelayLogKeepEnabled, Value: "true"},      // 默认保留历史日志
		{Key: SettingKeyCircuitBreakerThreshold, Value: "5"},     // 默认连续失败5次触发熔断
		{Key: SettingKeyCircuitBreakerCooldown, Value: "60"},     // 默认基础冷却60秒
		{Key: SettingKeyCircuitBreakerMaxCooldown, Value: "600"}, // 默认最大冷却600秒（10分钟）

		// 探活默认偏保守：不强制熔断、间隔 2 分钟、失败 3 次才 unhealthy
		{Key: SettingKeyHealthProbeEnabled, Value: "true"},
		{Key: SettingKeyHealthProbeInterval, Value: "120"},
		{Key: SettingKeyHealthProbeTimeout, Value: "8"},
		{Key: SettingKeyHealthProbeMethod, Value: HealthProbeMethodAuto},
		{Key: SettingKeyHealthProbePath, Value: ""},
		{Key: SettingKeyHealthProbeFailThreshold, Value: "3"},
		{Key: SettingKeyHealthProbeDegradeMS, Value: "5000"},
		{Key: SettingKeyHealthProbeTripOnFail, Value: "false"}, // 默认不强制熔断，避免误杀
		{Key: SettingKeyHealthDashboardRefresh, Value: "15"},
		// 网络观测：auto=优先 eBPF（失败回落 Go），go=纯应用层，ebpf=强制内核态
		{Key: SettingKeyNetObsMode, Value: "auto"},

		// 流式：默认保留最多 32 个事件 / 64KB 用于日志聚合，避免长对话 MB 级抖动
		{Key: SettingKeyStreamLogMaxEvents, Value: "32"},
		{Key: SettingKeyStreamLogMaxBytes, Value: "65536"},

		// 健康分选路默认开：degraded 往后排，不改变分组模式
		{Key: SettingKeyHealthScoreRouting, Value: "true"},
		// 语义路由默认关：需要时再开
		{Key: SettingKeySemanticRouteEnabled, Value: "false"},

		// 混沌工程默认全关
		{Key: SettingKeyChaosEnabled, Value: "false"},
		{Key: SettingKeyChaosDelayMS, Value: "0"},
		{Key: SettingKeyChaosErrorRate, Value: "0"},
		{Key: SettingKeyChaosDropRate, Value: "0"},
	}
}

func (s *Setting) Validate() error {
	switch s.Key {
	case SettingKeyModelInfoUpdateInterval, SettingKeySyncLLMInterval, SettingKeyRelayLogKeepPeriod,
		SettingKeyCircuitBreakerThreshold, SettingKeyCircuitBreakerCooldown, SettingKeyCircuitBreakerMaxCooldown,
		SettingKeyHealthProbeInterval, SettingKeyHealthProbeTimeout,
		SettingKeyHealthProbeFailThreshold, SettingKeyHealthProbeDegradeMS,
		SettingKeyHealthDashboardRefresh, SettingKeyStatsSaveInterval,
		SettingKeyStreamLogMaxEvents, SettingKeyStreamLogMaxBytes,
		SettingKeyChaosDelayMS, SettingKeyChaosErrorRate, SettingKeyChaosDropRate:
		v, err := strconv.Atoi(s.Value)
		if err != nil {
			return fmt.Errorf("%s must be an integer", s.Key)
		}
		// 合理范围校验，防止写成 0 秒狂刷或负值
		switch s.Key {
		case SettingKeyHealthProbeInterval:
			if v < 0 {
				return fmt.Errorf("health_probe_interval must be >= 0 (0=disable)")
			}
			if v > 0 && v < 15 {
				return fmt.Errorf("health_probe_interval minimum is 15 seconds when enabled")
			}
		case SettingKeyHealthProbeTimeout:
			if v < 1 || v > 120 {
				return fmt.Errorf("health_probe_timeout must be 1-120 seconds")
			}
		case SettingKeyHealthProbeFailThreshold:
			if v < 1 || v > 100 {
				return fmt.Errorf("health_probe_fail_threshold must be 1-100")
			}
		case SettingKeyHealthProbeDegradeMS:
			if v < 100 || v > 300000 {
				return fmt.Errorf("health_probe_degrade_ms must be 100-300000")
			}
		case SettingKeyHealthDashboardRefresh:
			if v < 3 || v > 600 {
				return fmt.Errorf("health_dashboard_refresh must be 3-600 seconds")
			}
		case SettingKeyStreamLogMaxEvents:
			if v < 0 || v > 10000 {
				return fmt.Errorf("stream_log_max_events must be 0-10000")
			}
		case SettingKeyStreamLogMaxBytes:
			if v < 0 || v > 16*1024*1024 {
				return fmt.Errorf("stream_log_max_bytes must be 0-16777216")
			}
		case SettingKeyChaosDelayMS:
			if v < 0 || v > 30000 {
				return fmt.Errorf("chaos_delay_ms must be 0-30000")
			}
		case SettingKeyChaosErrorRate, SettingKeyChaosDropRate:
			if v < 0 || v > 100 {
				return fmt.Errorf("%s must be 0-100", s.Key)
			}
		}
		return nil
	case SettingKeyRelayLogKeepEnabled, SettingKeyHealthProbeEnabled, SettingKeyHealthProbeTripOnFail,
		SettingKeyHealthScoreRouting, SettingKeySemanticRouteEnabled, SettingKeyChaosEnabled:
		if s.Value != "true" && s.Value != "false" {
			return fmt.Errorf("%s must be true or false", s.Key)
		}
		return nil
	case SettingKeyHealthProbeMethod:
		m := strings.ToLower(strings.TrimSpace(s.Value))
		switch m {
		case HealthProbeMethodAuto, HealthProbeMethodModels, HealthProbeMethodHead,
			HealthProbeMethodChat, HealthProbeMethodCustom:
			s.Value = m
			return nil
		default:
			return fmt.Errorf("health_probe_method must be one of: auto, models, head, chat, custom")
		}
	case SettingKeyHealthProbePath:
		// 任意字符串均可
		return nil
	case SettingKeyNetObsMode:
		m := strings.ToLower(strings.TrimSpace(s.Value))
		switch m {
		case "auto", "go", "ebpf":
			s.Value = m
			return nil
		default:
			return fmt.Errorf("net_obs_mode must be one of: auto, go, ebpf")
		}
	case SettingKeyProxyURL:
		if s.Value == "" {
			return nil
		}
		parsedURL, err := url.Parse(s.Value)
		if err != nil {
			return fmt.Errorf("proxy URL is invalid: %w", err)
		}
		validSchemes := map[string]bool{
			"http":   true,
			"https":  true,
			"socks5": true,
		}
		if !validSchemes[parsedURL.Scheme] {
			return fmt.Errorf("proxy URL scheme must be http, https, socks, or socks5")
		}
		if parsedURL.Host == "" {
			return fmt.Errorf("proxy URL must have a host")
		}
		return nil
	}

	return nil
}
