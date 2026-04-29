package model

type OpsCacheStatus struct {
	Enabled        bool    `json:"enabled"`
	RuntimeEnabled bool    `json:"runtime_enabled"`
	TTLSeconds     int     `json:"ttl_seconds"`
	Threshold      int     `json:"threshold"`
	MaxEntries     int     `json:"max_entries"`
	CurrentEntries int     `json:"current_entries"`
	Hits           int64   `json:"hits"`
	Misses         int64   `json:"misses"`
	HitRate        float64 `json:"hit_rate"`
	UsageRate      float64 `json:"usage_rate"`
}

type OpsQuotaKeyItem struct {
	APIKeyID            int     `json:"api_key_id"`
	Name                string  `json:"name"`
	Enabled             bool    `json:"enabled"`
	Expired             bool    `json:"expired"`
	Status              string  `json:"status"`
	SupportedModelCount int     `json:"supported_model_count"`
	HasPerModelQuota    bool    `json:"has_per_model_quota"`
	RateLimitRPM        int     `json:"rate_limit_rpm"`
	RateLimitTPM        int     `json:"rate_limit_tpm"`
	MaxCost             float64 `json:"max_cost"`
	RequestCount        int64   `json:"request_count"`
	TotalCost           float64 `json:"total_cost"`
}

type OpsQuotaSummary struct {
	TotalKeyCount         int               `json:"total_key_count"`
	EnabledKeyCount       int               `json:"enabled_key_count"`
	AvailableKeyCount     int               `json:"available_key_count"`
	ExpiredKeyCount       int               `json:"expired_key_count"`
	LimitedKeyCount       int               `json:"limited_key_count"`
	UnlimitedKeyCount     int               `json:"unlimited_key_count"`
	ExhaustedKeyCount     int               `json:"exhausted_key_count"`
	PerModelQuotaKeyCount int               `json:"per_model_quota_key_count"`
	ActiveUsageKeyCount   int               `json:"active_usage_key_count"`
	TotalRPM              int               `json:"total_rpm"`
	TotalTPM              int               `json:"total_tpm"`
	TotalMaxCost          float64           `json:"total_max_cost"`
	Keys                  []OpsQuotaKeyItem `json:"keys"`
}

type OpsHealthGroupItem struct {
	GroupID      int    `json:"group_id"`
	GroupName    string `json:"group_name"`
	EndpointType string `json:"endpoint_type"`
	Status       string `json:"status"`
	FailureCount int64  `json:"failure_count"`
	HealthScore  int    `json:"health_score"`
}

type OpsHealthStatus struct {
	DatabaseOK         bool                 `json:"database_ok"`
	CacheOK            bool                 `json:"cache_ok"`
	TaskRuntimeOK      bool                 `json:"task_runtime_ok"`
	RecentErrorCount   int64                `json:"recent_error_count"`
	HealthyGroupCount  int                  `json:"healthy_group_count"`
	WarningGroupCount  int                  `json:"warning_group_count"`
	DegradedGroupCount int                  `json:"degraded_group_count"`
	DownGroupCount     int                  `json:"down_group_count"`
	EmptyGroupCount    int                  `json:"empty_group_count"`
	FailingGroups      []OpsHealthGroupItem `json:"failing_groups"`
	CheckedAt          int64                `json:"checked_at"`
}

type OpsAIRouteServiceSummary struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	Enabled bool   `json:"enabled"`
}

type OpsSystemSummary struct {
	Version                      string                     `json:"version"`
	Commit                       string                     `json:"commit"`
	BuildTime                    string                     `json:"build_time"`
	Repo                         string                     `json:"repo"`
	DatabaseType                 string                     `json:"database_type"`
	PublicAPIBaseURL             string                     `json:"public_api_base_url"`
	ProxyURL                     string                     `json:"proxy_url"`
	RelayLogKeepEnabled          bool                       `json:"relay_log_keep_enabled"`
	RelayLogKeepDays             int                        `json:"relay_log_keep_days"`
	StatsSaveIntervalMinutes     int                        `json:"stats_save_interval_minutes"`
	SyncLLMIntervalHours         int                        `json:"sync_llm_interval_hours"`
	ModelInfoUpdateIntervalHours int                        `json:"model_info_update_interval_hours"`
	ImportEnabled                bool                       `json:"import_enabled"`
	ExportEnabled                bool                       `json:"export_enabled"`
	AIRouteGroupID               int                        `json:"ai_route_group_id"`
	AIRouteTimeoutSeconds        int                        `json:"ai_route_timeout_seconds"`
	AIRouteParallelism           int                        `json:"ai_route_parallelism"`
	AIRouteLegacyMode            bool                       `json:"ai_route_legacy_mode"`
	AIRouteServiceCount          int                        `json:"ai_route_service_count"`
	AIRouteEnabledServiceCount   int                        `json:"ai_route_enabled_service_count"`
	AIRouteServices              []OpsAIRouteServiceSummary `json:"ai_route_services"`
	ChannelCount                 int                        `json:"channel_count"`
	GroupCount                   int                        `json:"group_count"`
	APIKeyCount                  int                        `json:"api_key_count"`
}
