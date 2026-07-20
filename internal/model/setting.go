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
	SettingKeyHealthProbeModel         SettingKey = "health_probe_model"          // chat 探活用的模型名（兼容 sub2api 等中转）
	SettingKeyHealthProbeFailThreshold SettingKey = "health_probe_fail_threshold" // 连续失败多少次才判 unhealthy
	SettingKeyHealthProbeDegradeMS     SettingKey = "health_probe_degrade_ms"     // 延迟超过多少 ms 判 degraded
	SettingKeyHealthProbeTripOnFail    SettingKey = "health_probe_trip_on_fail"   // 探活失败是否强制打开熔断 true/false
	SettingKeyHealthDashboardRefresh   SettingKey = "health_dashboard_refresh"    // 前端看板刷新间隔（秒）
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
		{Key: SettingKeyHealthProbeModel, Value: "gpt-4o-mini"},
		{Key: SettingKeyHealthProbeFailThreshold, Value: "3"},
		{Key: SettingKeyHealthProbeDegradeMS, Value: "5000"},
		{Key: SettingKeyHealthProbeTripOnFail, Value: "false"}, // 默认不强制熔断，避免误杀
		{Key: SettingKeyHealthDashboardRefresh, Value: "15"},
	}
}

func (s *Setting) Validate() error {
	switch s.Key {
	case SettingKeyModelInfoUpdateInterval, SettingKeySyncLLMInterval, SettingKeyRelayLogKeepPeriod,
		SettingKeyCircuitBreakerThreshold, SettingKeyCircuitBreakerCooldown, SettingKeyCircuitBreakerMaxCooldown,
		SettingKeyHealthProbeInterval, SettingKeyHealthProbeTimeout,
		SettingKeyHealthProbeFailThreshold, SettingKeyHealthProbeDegradeMS,
		SettingKeyHealthDashboardRefresh, SettingKeyStatsSaveInterval:
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
		}
		return nil
	case SettingKeyRelayLogKeepEnabled, SettingKeyHealthProbeEnabled, SettingKeyHealthProbeTripOnFail:
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
	case SettingKeyHealthProbePath, SettingKeyHealthProbeModel:
		// 任意字符串均可
		return nil
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
