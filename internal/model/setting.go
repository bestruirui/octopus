package model

import (
	"fmt"
	"net/url"
	"strconv"
)

type SettingKey string

const (
	SettingKeyProxyURL                  SettingKey = "proxy_url"
	SettingKeyStatsSaveInterval         SettingKey = "stats_save_interval"          // 将统计信息写入数据库的周期(分钟)
	SettingKeyModelInfoUpdateInterval   SettingKey = "model_info_update_interval"   // 模型信息更新间隔(小时)
	SettingKeySyncLLMInterval           SettingKey = "sync_llm_interval"            // LLM 同步间隔(小时)
	SettingKeyRelayLogKeepPeriod        SettingKey = "relay_log_keep_period"        // 日志保存时间范围(天)
	SettingKeyRelayLogKeepEnabled       SettingKey = "relay_log_keep_enabled"       // 是否保留历史日志
	SettingKeyCORSAllowOrigins          SettingKey = "cors_allow_origins"           // 跨域白名单(逗号分隔, 如 "example.com,example2.com"). 为空不允许跨域, "*"允许所有
	SettingKeyRelayRetryCount           SettingKey = "relay_retry_count"            // 单个候选渠道失败后的最大重试次数
	SettingKeyCircuitBreakerThreshold   SettingKey = "circuit_breaker_threshold"    // 熔断触发阈值（连续失败次数）
	SettingKeyCircuitBreakerCooldown    SettingKey = "circuit_breaker_cooldown"     // 熔断基础冷却时间（秒）
	SettingKeyCircuitBreakerMaxCooldown SettingKey = "circuit_breaker_max_cooldown" // 熔断最大冷却时间（秒），指数退避上限
	SettingKeyPublicAPIBaseURL          SettingKey = "public_api_base_url"          // 对外可访问的 API 基础地址，用于生成示例
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
		{Key: SettingKeyRelayRetryCount, Value: "3"},             // 默认单个候选渠道失败后重试3次
		{Key: SettingKeyCircuitBreakerThreshold, Value: "5"},     // 默认连续失败5次触发熔断
		{Key: SettingKeyCircuitBreakerCooldown, Value: "60"},     // 默认基础冷却60秒
		{Key: SettingKeyCircuitBreakerMaxCooldown, Value: "600"}, // 默认最大冷却600秒（10分钟）
		{Key: SettingKeyPublicAPIBaseURL, Value: ""},
	}
}

func (s *Setting) Validate() error {
	switch s.Key {
	case SettingKeyModelInfoUpdateInterval, SettingKeySyncLLMInterval, SettingKeyRelayLogKeepPeriod,
		SettingKeyRelayRetryCount, SettingKeyCircuitBreakerThreshold, SettingKeyCircuitBreakerCooldown, SettingKeyCircuitBreakerMaxCooldown:
		v, err := strconv.Atoi(s.Value)
		if err != nil {
			return fmt.Errorf("setting value must be an integer")
		}
		if s.Key == SettingKeyRelayRetryCount && v < 1 {
			return fmt.Errorf("relay retry count must be greater than 0")
		}
		return nil
	case SettingKeyRelayLogKeepEnabled:
		if s.Value != "true" && s.Value != "false" {
			return fmt.Errorf("relay log keep enabled must be true or false")
		}
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
	case SettingKeyPublicAPIBaseURL:
		if s.Value == "" {
			return nil
		}
		parsedURL, err := url.Parse(s.Value)
		if err != nil {
			return fmt.Errorf("public API base URL is invalid: %w", err)
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("public API base URL scheme must be http or https")
		}
		if parsedURL.Host == "" {
			return fmt.Errorf("public API base URL must have a host")
		}
		return nil
	}

	return nil
}
