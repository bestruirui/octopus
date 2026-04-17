package model

type AutoStrategyRecord struct {
	Timestamp int64 `json:"timestamp"`
	Success   bool  `json:"success"`
}

type AutoStrategyState struct {
	Key       string               `json:"key" gorm:"primaryKey"`
	ChannelID int                  `json:"channel_id" gorm:"index"`
	ModelName string               `json:"model_name"`
	Records   []AutoStrategyRecord `json:"records" gorm:"serializer:json"`
	UpdatedAt int64                `json:"updated_at"`
}

type CircuitBreakerState struct {
	Key                 string `json:"key" gorm:"primaryKey"`
	ChannelID           int    `json:"channel_id" gorm:"index"`
	ChannelKeyID        int    `json:"channel_key_id" gorm:"index"`
	ModelName           string `json:"model_name"`
	State               int    `json:"state"`
	ConsecutiveFailures int64  `json:"consecutive_failures"`
	LastFailureTime     int64  `json:"last_failure_time"`
	TripCount           int    `json:"trip_count"`
	UpdatedAt           int64  `json:"updated_at"`
}
