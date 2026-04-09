package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type SettingKey string

const (
	SettingKeyProxyURL                      SettingKey = "proxy_url"
	SettingKeyStatsSaveInterval             SettingKey = "stats_save_interval"                // 将统计信息写入数据库的周期(分钟)
	SettingKeyModelInfoUpdateInterval       SettingKey = "model_info_update_interval"         // 模型信息更新间隔(小时)
	SettingKeySyncLLMInterval               SettingKey = "sync_llm_interval"                  // LLM 同步间隔(小时)
	SettingKeyRelayLogKeepPeriod            SettingKey = "relay_log_keep_period"              // 日志保存时间范围(天)
	SettingKeyRelayLogKeepEnabled           SettingKey = "relay_log_keep_enabled"             // 是否保留历史日志
	SettingKeyCORSAllowOrigins              SettingKey = "cors_allow_origins"                 // 跨域白名单(逗号分隔, 如 "example.com,example2.com"). 为空不允许跨域, "*"允许所有
	SettingKeyCircuitBreakerThreshold       SettingKey = "circuit_breaker_threshold"          // 熔断触发阈值（连续失败次数）
	SettingKeyCircuitBreakerCooldown        SettingKey = "circuit_breaker_cooldown"           // 熔断基础冷却时间（秒）
	SettingKeyCircuitBreakerMaxCooldown     SettingKey = "circuit_breaker_max_cooldown"       // 熔断最大冷却时间（秒），指数退避上限
	SettingKeyHealthScoreWeights            SettingKey = "health_score_weights"               // 健康评分权重(JSON)
	SettingKeyGroupDefaultFirstTokenTimeOut SettingKey = "group_default_first_token_time_out" // 分组默认首字超时（秒）
	SettingKeyGroupDefaultSessionKeepTime   SettingKey = "group_default_session_keep_time"    // 分组默认会话保持（秒）
)

type HealthScoreWeights struct {
	SuccessRate      float64 `json:"success_rate"`
	AvgWait          float64 `json:"avg_wait"`
	KeyAvailability  float64 `json:"key_availability"`
	BaseDelay        float64 `json:"base_delay"`
	PriorityBoost    float64 `json:"priority_boost"`
	WeightBoost      float64 `json:"weight_boost"`
	RecentUseBonus   float64 `json:"recent_use_bonus"`
	RateLimitPenalty float64 `json:"rate_limit_penalty"`
	CostPenalty      float64 `json:"cost_penalty"`
	ColdStartScore   float64 `json:"cold_start_score"`
}

func DefaultHealthScoreWeights() HealthScoreWeights {
	return HealthScoreWeights{
		SuccessRate:      70,
		AvgWait:          20,
		KeyAvailability:  10,
		BaseDelay:        10,
		PriorityBoost:    10,
		WeightBoost:      0.5,
		RecentUseBonus:   5,
		RateLimitPenalty: 30,
		CostPenalty:      2,
		ColdStartScore:   70,
	}
}

func DefaultHealthScoreWeightsJSON() string {
	b, _ := json.Marshal(DefaultHealthScoreWeights())
	return string(b)
}

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
		{Key: SettingKeyHealthScoreWeights, Value: DefaultHealthScoreWeightsJSON()},
		{Key: SettingKeyGroupDefaultFirstTokenTimeOut, Value: "0"},
		{Key: SettingKeyGroupDefaultSessionKeepTime, Value: "0"},
	}
}

func (s *Setting) Validate() error {
	switch s.Key {
	case SettingKeyModelInfoUpdateInterval, SettingKeySyncLLMInterval, SettingKeyRelayLogKeepPeriod,
		SettingKeyCircuitBreakerThreshold, SettingKeyCircuitBreakerCooldown, SettingKeyCircuitBreakerMaxCooldown,
		SettingKeyGroupDefaultFirstTokenTimeOut, SettingKeyGroupDefaultSessionKeepTime:
		value, err := strconv.Atoi(s.Value)
		if err != nil {
			return fmt.Errorf("model info update interval must be an integer")
		}
		if value < 0 {
			return fmt.Errorf("setting must be non-negative")
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
	case SettingKeyHealthScoreWeights:
		var weights HealthScoreWeights
		if err := json.Unmarshal([]byte(s.Value), &weights); err != nil {
			return fmt.Errorf("health score weights must be valid JSON: %w", err)
		}
		if weights.SuccessRate < 0 || weights.AvgWait < 0 || weights.KeyAvailability < 0 || weights.BaseDelay < 0 ||
			weights.PriorityBoost < 0 || weights.WeightBoost < 0 || weights.RecentUseBonus < 0 ||
			weights.RateLimitPenalty < 0 || weights.CostPenalty < 0 || weights.ColdStartScore < 0 {
			return fmt.Errorf("health score weights must be non-negative")
		}
		return nil
	}

	return nil
}
