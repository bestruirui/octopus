package model

import (
	"fmt"
	"net/url"
	"strconv"
)

type SettingKey string

const (
	SettingKeyProxyURL                             SettingKey = "proxy_url"
	SettingKeyStatsSaveInterval                    SettingKey = "stats_save_interval"                      // 将统计信息写入数据库的周期(分钟)
	SettingKeyModelInfoUpdateInterval              SettingKey = "model_info_update_interval"               // 模型信息更新间隔(小时)
	SettingKeySyncLLMInterval                      SettingKey = "sync_llm_interval"                        // LLM 同步间隔(小时)
	SettingKeyRelayLogKeepPeriod                   SettingKey = "relay_log_keep_period"                    // 日志保存时间范围(天)
	SettingKeyRelayLogKeepEnabled                  SettingKey = "relay_log_keep_enabled"                   // 是否保留历史日志
	SettingKeyCORSAllowOrigins                     SettingKey = "cors_allow_origins"                       // 跨域白名单(逗号分隔, 如 "example.com,example2.com"). 为空不允许跨域, "*"允许所有
	SettingKeyRelayRetryCount                      SettingKey = "relay_retry_count"                        // 单个候选渠道失败后的最大重试次数
	SettingKeyCircuitBreakerThreshold              SettingKey = "circuit_breaker_threshold"                // 熔断触发阈值（连续失败次数）
	SettingKeyCircuitBreakerCooldown               SettingKey = "circuit_breaker_cooldown"                 // 熔断基础冷却时间（秒）
	SettingKeyCircuitBreakerMaxCooldown            SettingKey = "circuit_breaker_max_cooldown"             // 熔断最大冷却时间（秒），指数退避上限
	SettingKeyPublicAPIBaseURL                     SettingKey = "public_api_base_url"                      // 对外可访问的 API 基础地址，用于生成示例
	SettingKeyAlertNotifyLanguage                  SettingKey = "alert_notify_language"                    // 告警通知发送语言
	SettingKeyRatelimitCooldown                    SettingKey = "ratelimit_cooldown"                       // 429 限流冷却时间（秒）
	SettingKeyRelayMaxTotalAttempts                SettingKey = "relay_max_total_attempts"                 // 所有候选渠道的最大总尝试次数，0 表示不限制
	SettingKeyAutoStrategyMinSamples               SettingKey = "auto_strategy_min_samples"                // Auto策略最小样本数阈值
	SettingKeyAutoStrategyTimeWindow               SettingKey = "auto_strategy_time_window"                // Auto策略时间窗口（秒）
	SettingKeyAutoStrategySampleThreshold          SettingKey = "auto_strategy_sample_threshold"           // Auto策略滑动窗口大小
	SettingKeyAutoStrategyLatencyWeight            SettingKey = "auto_strategy_latency_weight"             // Auto策略延迟权重（0-100）
	SettingKeySemanticCacheEnabled                 SettingKey = "semantic_cache_enabled"                   // 语义缓存开关
	SettingKeySemanticCacheTTL                     SettingKey = "semantic_cache_ttl"                       // 语义缓存 TTL（秒）
	SettingKeySemanticCacheThreshold               SettingKey = "semantic_cache_threshold"                 // 语义缓存相似度阈值（0-1）
	SettingKeySemanticCacheMaxEntries              SettingKey = "semantic_cache_max_entries"               // 语义缓存最大条目数
	SettingKeySemanticCacheEmbeddingBaseURL        SettingKey = "semantic_cache_embedding_base_url"        // 语义缓存 embedding 服务 Base URL
	SettingKeySemanticCacheEmbeddingAPIKey         SettingKey = "semantic_cache_embedding_api_key"         // 语义缓存 embedding 服务 API Key
	SettingKeySemanticCacheEmbeddingModel          SettingKey = "semantic_cache_embedding_model"           // 语义缓存 embedding 模型名称
	SettingKeySemanticCacheEmbeddingTimeoutSeconds SettingKey = "semantic_cache_embedding_timeout_seconds" // 语义缓存 embedding 请求超时（秒）
	SettingKeyNavOrder                             SettingKey = "nav_order"                                // 顶级页面顺序(JSON)
	SettingKeyAIRouteGroupID                       SettingKey = "ai_route_group_id"                        // AI路由目标分组 ID
	SettingKeyAIRouteBaseURL                       SettingKey = "ai_route_base_url"                        // AI路由分析服务 Base URL
	SettingKeyAIRouteAPIKey                        SettingKey = "ai_route_api_key"                         // AI路由分析服务 API Key
	SettingKeyAIRouteModel                         SettingKey = "ai_route_model"                           // AI路由分析模型名称
	SettingKeyAIRouteTimeoutSeconds                SettingKey = "ai_route_timeout_seconds"                 // AI路由分析单次请求超时（秒）
	SettingKeyAIRouteParallelism                   SettingKey = "ai_route_parallelism"                     // AI路由分析批次最大并发数
	SettingKeyAIRouteServices                      SettingKey = "ai_route_services"                        // AI路由分析服务池(JSON)
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
		{Key: SettingKeyRatelimitCooldown, Value: "300"},         // 默认429冷却300秒（5分钟）
		{Key: SettingKeyRelayMaxTotalAttempts, Value: "0"},       // 默认不限制所有候选渠道的总尝试次数
		{Key: SettingKeyPublicAPIBaseURL, Value: ""},
		{Key: SettingKeyAlertNotifyLanguage, Value: "en"},
		{Key: SettingKeyAutoStrategyMinSamples, Value: "10"},       // 默认最小样本数10次
		{Key: SettingKeyAutoStrategyTimeWindow, Value: "300"},      // 默认时间窗口300秒（5分钟）
		{Key: SettingKeyAutoStrategySampleThreshold, Value: "100"}, // 默认滑动窗口大小100条
		{Key: SettingKeyAutoStrategyLatencyWeight, Value: "30"},    // 默认延迟权重30%
		{Key: SettingKeySemanticCacheEnabled, Value: "false"},      // 默认关闭语义缓存
		{Key: SettingKeySemanticCacheTTL, Value: "3600"},           // 默认TTL 1小时
		{Key: SettingKeySemanticCacheThreshold, Value: "98"},       // 默认相似度阈值 0.98（0-100）
		{Key: SettingKeySemanticCacheMaxEntries, Value: "1000"},    // 默认最大1000条
		{Key: SettingKeySemanticCacheEmbeddingBaseURL, Value: ""},
		{Key: SettingKeySemanticCacheEmbeddingAPIKey, Value: ""},
		{Key: SettingKeySemanticCacheEmbeddingModel, Value: ""},
		{Key: SettingKeySemanticCacheEmbeddingTimeoutSeconds, Value: "10"},
		{Key: SettingKeyNavOrder, Value: `["home","channel","group","model","analytics","log","alert","ops","setting","user"]`},
		{Key: SettingKeyAIRouteGroupID, Value: "0"},
		{Key: SettingKeyAIRouteBaseURL, Value: ""},
		{Key: SettingKeyAIRouteAPIKey, Value: ""},
		{Key: SettingKeyAIRouteModel, Value: ""},
		{Key: SettingKeyAIRouteTimeoutSeconds, Value: "180"},
		{Key: SettingKeyAIRouteParallelism, Value: "3"},
		{Key: SettingKeyAIRouteServices, Value: "[]"},
	}
}

func (s *Setting) Validate() error {
	switch s.Key {
	case SettingKeyModelInfoUpdateInterval, SettingKeySyncLLMInterval, SettingKeyRelayLogKeepPeriod,
		SettingKeyRelayRetryCount, SettingKeyCircuitBreakerThreshold, SettingKeyCircuitBreakerCooldown,
		SettingKeyCircuitBreakerMaxCooldown, SettingKeyRatelimitCooldown, SettingKeyRelayMaxTotalAttempts,
		SettingKeySemanticCacheTTL, SettingKeySemanticCacheThreshold, SettingKeySemanticCacheMaxEntries,
		SettingKeySemanticCacheEmbeddingTimeoutSeconds,
		SettingKeyAutoStrategyMinSamples, SettingKeyAutoStrategyTimeWindow, SettingKeyAutoStrategySampleThreshold,
		SettingKeyAutoStrategyLatencyWeight,
		SettingKeyAIRouteGroupID, SettingKeyAIRouteTimeoutSeconds, SettingKeyAIRouteParallelism:
		v, err := strconv.Atoi(s.Value)
		if err != nil {
			return fmt.Errorf("setting value must be an integer")
		}
		if s.Key == SettingKeyRelayRetryCount && v < 1 {
			return fmt.Errorf("relay retry count must be greater than 0")
		}
		if (s.Key == SettingKeyRatelimitCooldown || s.Key == SettingKeyRelayMaxTotalAttempts) && v < 0 {
			return fmt.Errorf("setting value must be greater than or equal to 0")
		}
		if (s.Key == SettingKeyAutoStrategyMinSamples || s.Key == SettingKeyAutoStrategyTimeWindow || s.Key == SettingKeyAutoStrategySampleThreshold) && v < 1 {
			return fmt.Errorf("auto strategy setting must be greater than 0")
		}
		if s.Key == SettingKeyAutoStrategyLatencyWeight && (v < 0 || v > 100) {
			return fmt.Errorf("auto strategy latency weight must be between 0 and 100")
		}
		if s.Key == SettingKeySemanticCacheTTL && v < 1 {
			return fmt.Errorf("semantic cache TTL must be greater than 0")
		}
		if s.Key == SettingKeySemanticCacheThreshold && (v < 0 || v > 100) {
			return fmt.Errorf("semantic cache threshold must be between 0 and 100")
		}
		if s.Key == SettingKeySemanticCacheMaxEntries && v < 1 {
			return fmt.Errorf("semantic cache max entries must be greater than 0")
		}
		if s.Key == SettingKeySemanticCacheEmbeddingTimeoutSeconds && v < 1 {
			return fmt.Errorf("semantic cache embedding timeout must be greater than 0")
		}
		if s.Key == SettingKeyAIRouteGroupID && v < 0 {
			return fmt.Errorf("ai route group id must be greater than or equal to 0")
		}
		if s.Key == SettingKeyAIRouteTimeoutSeconds && v < 1 {
			return fmt.Errorf("ai route timeout must be greater than 0")
		}
		if s.Key == SettingKeyAIRouteParallelism && v < 1 {
			return fmt.Errorf("ai route parallelism must be greater than 0")
		}
		return nil
	case SettingKeyRelayLogKeepEnabled, SettingKeySemanticCacheEnabled:
		if s.Value != "true" && s.Value != "false" {
			return fmt.Errorf("setting value must be true or false")
		}
		return nil
	case SettingKeyProxyURL, SettingKeySemanticCacheEmbeddingBaseURL, SettingKeyAIRouteBaseURL:
		if s.Value == "" {
			return nil
		}
		parsedURL, err := url.Parse(s.Value)
		if err != nil {
			if s.Key == SettingKeySemanticCacheEmbeddingBaseURL {
				return fmt.Errorf("semantic cache embedding base URL is invalid: %w", err)
			}
			if s.Key == SettingKeyAIRouteBaseURL {
				return fmt.Errorf("ai route base URL is invalid: %w", err)
			}
			return fmt.Errorf("proxy URL is invalid: %w", err)
		}
		if s.Key == SettingKeySemanticCacheEmbeddingBaseURL {
			if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
				return fmt.Errorf("semantic cache embedding base URL scheme must be http or https")
			}
			if parsedURL.Host == "" {
				return fmt.Errorf("semantic cache embedding base URL must have a host")
			}
			return nil
		}
		if s.Key == SettingKeyAIRouteBaseURL {
			if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
				return fmt.Errorf("ai route base URL scheme must be http or https")
			}
			if parsedURL.Host == "" {
				return fmt.Errorf("ai route base URL must have a host")
			}
			return nil
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
	case SettingKeyAlertNotifyLanguage:
		switch s.Value {
		case "zh-Hans", "zh-Hant", "en":
			return nil
		default:
			return fmt.Errorf("alert notify language must be zh-Hans, zh-Hant, or en")
		}
	case SettingKeyAIRouteServices:
		return ValidateAIRouteServiceConfigs(s.Value)
	}

	return nil
}
