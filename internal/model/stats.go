package model

type StatsMetrics struct {
	InputToken     int64   `json:"input_token" gorm:"bigint"`
	OutputToken    int64   `json:"output_token" gorm:"bigint"`
	InputCost      float64 `json:"input_cost" gorm:"type:real"`
	OutputCost     float64 `json:"output_cost" gorm:"type:real"`
	WaitTime       int64   `json:"wait_time" gorm:"bigint"`
	RequestSuccess int64   `json:"request_success" gorm:"bigint"`
	RequestFailed  int64   `json:"request_failed" gorm:"bigint"`
}

type StatsTotal struct {
	ID int `gorm:"primaryKey"`
	StatsMetrics
}

type StatsHourly struct {
	Hour int    `json:"hour" gorm:"primaryKey"`
	Date string `json:"date" gorm:"not null"` // 记录最后更新日期，格式：20060102
	StatsMetrics
}

type StatsDaily struct {
	Date string `json:"date" gorm:"primaryKey"`
	StatsMetrics
}

type StatsModel struct {
	ID        int    `json:"id" gorm:"primaryKey"`
	Name      string `json:"name" gorm:"not null"`
	ChannelID int    `json:"channel_id" gorm:"not null"`
	StatsMetrics
}

type StatsChannel struct {
	ChannelID int `json:"channel_id" gorm:"primaryKey"`
	StatsMetrics
}

type StatsAPIKey struct {
	APIKeyID int `json:"api_key_id" gorm:"primaryKey"`
	StatsMetrics
}

type HealthGrade string

const (
	HealthGradeExcellent HealthGrade = "excellent"
	HealthGradeGood      HealthGrade = "good"
	HealthGradeWarning   HealthGrade = "warning"
	HealthGradeCritical  HealthGrade = "critical"
)

type StatsModelHealth struct {
	ChannelID      int         `json:"channel_id"`
	ModelName      string      `json:"model_name"`
	Score          float64     `json:"score"`
	Grade          HealthGrade `json:"grade"`
	RequestSuccess int64       `json:"request_success"`
	RequestFailed  int64       `json:"request_failed"`
	RequestCount   int64       `json:"request_count"`
	SuccessRate    float64     `json:"success_rate"`
	AvgWaitTime    int64       `json:"avg_wait_time"`
	Healthy        bool        `json:"healthy"`
}

type StatsChannelHealth struct {
	ChannelID      int                `json:"channel_id"`
	ChannelName    string             `json:"channel_name"`
	Enabled        bool               `json:"enabled"`
	Type           int                `json:"type"`
	Score          float64            `json:"score"`
	Grade          HealthGrade        `json:"grade"`
	RequestSuccess int64              `json:"request_success"`
	RequestFailed  int64              `json:"request_failed"`
	RequestCount   int64              `json:"request_count"`
	SuccessRate    float64            `json:"success_rate"`
	AvgWaitTime    int64              `json:"avg_wait_time"`
	BaseURLDelay   int                `json:"base_url_delay"`
	EnabledKeys    int                `json:"enabled_keys"`
	TotalKeys      int                `json:"total_keys"`
	HealthyModels  int                `json:"healthy_models"`
	TotalModels    int                `json:"total_models"`
	Models         []StatsModelHealth `json:"models,omitempty"`
}

// Add aggregates another StatsMetrics into the current one.
func (s *StatsMetrics) Add(delta StatsMetrics) {
	s.InputToken += delta.InputToken
	s.OutputToken += delta.OutputToken
	s.InputCost += delta.InputCost
	s.OutputCost += delta.OutputCost
	s.WaitTime += delta.WaitTime
	s.RequestSuccess += delta.RequestSuccess
	s.RequestFailed += delta.RequestFailed
}
