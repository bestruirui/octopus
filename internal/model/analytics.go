package model

import (
	"fmt"
	"strings"
)

type AnalyticsRange string

const (
	AnalyticsRange1D  AnalyticsRange = "1d"
	AnalyticsRange7D  AnalyticsRange = "7d"
	AnalyticsRange30D AnalyticsRange = "30d"
	AnalyticsRange90D AnalyticsRange = "90d"
	AnalyticsRangeYTD AnalyticsRange = "ytd"
	AnalyticsRangeAll AnalyticsRange = "all"
)

func ParseAnalyticsRange(raw string) (AnalyticsRange, error) {
	value := AnalyticsRange(strings.ToLower(strings.TrimSpace(raw)))
	if value == "" {
		return AnalyticsRange7D, nil
	}

	switch value {
	case AnalyticsRange1D, AnalyticsRange7D, AnalyticsRange30D, AnalyticsRange90D, AnalyticsRangeYTD, AnalyticsRangeAll:
		return value, nil
	default:
		return "", fmt.Errorf("invalid analytics range: %s", raw)
	}
}

type AnalyticsMetrics struct {
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalCost    float64 `json:"total_cost"`
	SuccessRate  float64 `json:"success_rate"`
}

type AnalyticsOverview struct {
	AnalyticsMetrics
	ProviderCount int     `json:"provider_count"`
	APIKeyCount   int     `json:"api_key_count"`
	ModelCount    int     `json:"model_count"`
	FallbackRate  float64 `json:"fallback_rate"`
}

type AnalyticsEvaluationSummary struct {
	SemanticCache SemanticCacheEvaluationSummary `json:"semantic_cache"`
}

type SemanticCacheEvaluationSummary struct {
	Enabled           bool    `json:"enabled"`
	RuntimeEnabled    bool    `json:"runtime_enabled"`
	TTLSeconds        int     `json:"ttl_seconds"`
	Threshold         int     `json:"threshold"`
	MaxEntries        int     `json:"max_entries"`
	CurrentEntries    int     `json:"current_entries"`
	Hits              int64   `json:"hits"`
	Misses            int64   `json:"misses"`
	HitRate           float64 `json:"hit_rate"`
	UsageRate         float64 `json:"usage_rate"`
	EvaluatedRequests int64   `json:"evaluated_requests"`
	CacheHitResponses int64   `json:"cache_hit_responses"`
	CacheMissRequests int64   `json:"cache_miss_requests"`
	BypassedRequests  int64   `json:"bypassed_requests"`
	StoredResponses   int64   `json:"stored_responses"`
}

type AnalyticsProviderBreakdownItem struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Enabled     bool   `json:"enabled"`
	AnalyticsMetrics
}

type AnalyticsModelBreakdownItem struct {
	ModelName string `json:"model_name"`
	AnalyticsMetrics
}

type AnalyticsAPIKeyBreakdownItem struct {
	APIKeyID *int   `json:"api_key_id,omitempty"`
	Name     string `json:"name"`
	AnalyticsMetrics
}

type AnalyticsUtilization struct {
	ProviderBreakdown []AnalyticsProviderBreakdownItem `json:"provider_breakdown"`
	ModelBreakdown    []AnalyticsModelBreakdownItem    `json:"model_breakdown"`
	APIKeyBreakdown   []AnalyticsAPIKeyBreakdownItem   `json:"apikey_breakdown"`
}

type AnalyticsGroupHealthItem struct {
	GroupID           int    `json:"group_id"`
	GroupName         string `json:"group_name"`
	EndpointType      string `json:"endpoint_type"`
	ItemCount         int    `json:"item_count"`
	EnabledItemCount  int    `json:"enabled_item_count"`
	DisabledItemCount int    `json:"disabled_item_count"`
	FailureCount      int64  `json:"failure_count"`
	LastFailureAt     int64  `json:"last_failure_at"`
	HealthScore       int    `json:"health_score"`
	Status            string `json:"status"`
}
