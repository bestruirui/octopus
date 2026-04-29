package model

import "time"

type LLMPrice struct {
	Input      float64 `json:"input" gorm:"column:input"`
	Output     float64 `json:"output" gorm:"column:output"`
	CacheRead  float64 `json:"cache_read" gorm:"column:cache_read"`
	CacheWrite float64 `json:"cache_write" gorm:"column:cache_write"`
}

type LLMInfo struct {
	Name string `json:"name" gorm:"primaryKey;not null"`
	LLMPrice
}

type LLMChannel struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
}

type ModelMarketChannel struct {
	ChannelID       int    `json:"channel_id"`
	ChannelName     string `json:"channel_name"`
	Enabled         bool   `json:"enabled"`
	EnabledKeyCount int    `json:"enabled_key_count"`
}

type ModelMarketItem struct {
	Name             string               `json:"name"`
	Input            float64              `json:"input"`
	Output           float64              `json:"output"`
	CacheRead        float64              `json:"cache_read"`
	CacheWrite       float64              `json:"cache_write"`
	ChannelCount     int                  `json:"channel_count"`
	EnabledKeyCount  int                  `json:"enabled_key_count"`
	AverageLatencyMS int64                `json:"average_latency_ms"`
	SuccessRate      float64              `json:"success_rate"`
	RequestSuccess   int64                `json:"request_success"`
	RequestFailed    int64                `json:"request_failed"`
	Channels         []ModelMarketChannel `json:"channels"`
}

type ModelMarketSummary struct {
	ModelCount         int       `json:"model_count"`
	CoverageCount      int       `json:"coverage_count"`
	UniqueChannelCount int       `json:"unique_channel_count"`
	AverageLatencyMS   int64     `json:"average_latency_ms"`
	LastUpdateTime     time.Time `json:"last_update_time"`
}

type ModelMarketResponse struct {
	Summary ModelMarketSummary `json:"summary"`
	Items   []ModelMarketItem  `json:"items"`
}

type GeminiModel struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

type GeminiModelList struct {
	Models        []GeminiModel `json:"models"`
	NextPageToken string        `json:"nextPageToken"`
}

type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type OpenAIModelList struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}
type AnthropicModel struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

type AnthropicModelList struct {
	Data    []AnthropicModel `json:"data"`
	FirstID string           `json:"first_id"`
	HasMore bool             `json:"has_more"`
	LastID  string           `json:"last_id"`
}

// TableName explicitly returns "-" for DTO structs to prevent GORM auto-mapping.
func (LLMChannel) TableName() string             { return "-" }
func (ModelMarketChannel) TableName() string     { return "-" }
func (ModelMarketItem) TableName() string        { return "-" }
func (ModelMarketSummary) TableName() string     { return "-" }
func (ModelMarketResponse) TableName() string    { return "-" }
func (GeminiModel) TableName() string            { return "-" }
func (GeminiModelList) TableName() string        { return "-" }
func (OpenAIModel) TableName() string            { return "-" }
func (OpenAIModelList) TableName() string        { return "-" }
func (AnthropicModel) TableName() string         { return "-" }
func (AnthropicModelList) TableName() string { return "-" }
