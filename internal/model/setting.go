package model

import (
	"fmt"
	"net/url"
	"strconv"
)

type SettingKey string

const (
	SettingKeyProxyURL                SettingKey = "proxy_url"
	SettingKeyStatsSaveInterval       SettingKey = "stats_save_interval"        // 将统计信息写入数据库的周期(分钟)
	SettingKeyModelInfoUpdateInterval SettingKey = "model_info_update_interval" // 模型信息更新间隔(小时)
	SettingKeySyncLLMInterval         SettingKey = "sync_llm_interval"          // LLM 同步间隔(小时)
	SettingKeyRelayLogKeepPeriod      SettingKey = "relay_log_keep_period"      // 日志保存时间范围(天)
	SettingKeyRelayLogKeepEnabled     SettingKey = "relay_log_keep_enabled"     // 是否保留历史日志
	SettingKeyCORSAllowOrigins        SettingKey = "cors_allow_origins"         // 跨域白名单(逗号分隔, 如 "example.com,example2.com"). 为空不允许跨域, "*"允许所有

	SettingKeyCBEnabled          SettingKey = "cb_enabled"
	SettingKeyCBFailureThreshold SettingKey = "cb_failure_threshold"
	SettingKeyCBBaseCooldownMS   SettingKey = "cb_base_cooldown_ms"
	SettingKeyCBMaxCooldownMS    SettingKey = "cb_max_cooldown_ms"
	SettingKeyCBBackoffFactor    SettingKey = "cb_backoff_factor"
	SettingKeyCBJitterMin        SettingKey = "cb_jitter_min"
	SettingKeyCBJitterMax        SettingKey = "cb_jitter_max"
	SettingKeyCBDecayWindowMS    SettingKey = "cb_decay_window_ms"
)

type Setting struct {
	Key   SettingKey `json:"key" gorm:"primaryKey"`
	Value string     `json:"value" gorm:"not null"`
}

func DefaultSettings() []Setting {
	return []Setting{
		{Key: SettingKeyProxyURL, Value: ""},
		{Key: SettingKeyStatsSaveInterval, Value: "10"},       // 默认10分钟保存一次统计信息
		{Key: SettingKeyCORSAllowOrigins, Value: ""},          // CORS 默认不允许跨域，设置为 "*" 才允许所有来源
		{Key: SettingKeyModelInfoUpdateInterval, Value: "24"}, // 默认24小时更新一次模型信息
		{Key: SettingKeySyncLLMInterval, Value: "24"},         // 默认24小时同步一次LLM
		{Key: SettingKeyRelayLogKeepPeriod, Value: "7"},       // 默认日志保存7天
		{Key: SettingKeyRelayLogKeepEnabled, Value: "true"},   // 默认保留历史日志
		{Key: SettingKeyCBEnabled, Value: "true"},
		{Key: SettingKeyCBFailureThreshold, Value: "3"},
		{Key: SettingKeyCBBaseCooldownMS, Value: "180000"},
		{Key: SettingKeyCBMaxCooldownMS, Value: "3600000"},
		{Key: SettingKeyCBBackoffFactor, Value: "2"},
		{Key: SettingKeyCBJitterMin, Value: "0.5"},
		{Key: SettingKeyCBJitterMax, Value: "1.5"},
		{Key: SettingKeyCBDecayWindowMS, Value: "21600000"}, // 6h
	}
}

func (s *Setting) Validate() error {
	switch s.Key {
	case SettingKeyModelInfoUpdateInterval, SettingKeySyncLLMInterval, SettingKeyRelayLogKeepPeriod,
		SettingKeyStatsSaveInterval, SettingKeyCBFailureThreshold, SettingKeyCBBaseCooldownMS,
		SettingKeyCBMaxCooldownMS, SettingKeyCBDecayWindowMS:
		_, err := strconv.Atoi(s.Value)
		if err != nil {
			return fmt.Errorf("%s must be an integer", s.Key)
		}
		return nil
	case SettingKeyRelayLogKeepEnabled, SettingKeyCBEnabled:
		if s.Value != "true" && s.Value != "false" {
			return fmt.Errorf("%s must be true or false", s.Key)
		}
		return nil
	case SettingKeyCBBackoffFactor, SettingKeyCBJitterMin, SettingKeyCBJitterMax:
		v, err := strconv.ParseFloat(s.Value, 64)
		if err != nil {
			return fmt.Errorf("%s must be a number", s.Key)
		}
		if v <= 0 {
			return fmt.Errorf("%s must be greater than 0", s.Key)
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
			"http":  true,
			"https": true,
			"socks": true,
		}
		if !validSchemes[parsedURL.Scheme] {
			return fmt.Errorf("proxy URL scheme must be http, https, or socks")
		}
		if parsedURL.Host == "" {
			return fmt.Errorf("proxy URL must have a host")
		}
		return nil
	}

	return nil
}
