package model

import "time"

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	CircuitBreakerClosed   CircuitBreakerState = 0 // 正常通行
	CircuitBreakerOpen     CircuitBreakerState = 1 // 熔断中，拒绝所有请求
	CircuitBreakerHalfOpen CircuitBreakerState = 2 // 半开，仅允许单个试探请求
)

// CircuitBreaker 熔断器持久化状态，粒度：渠道(Provider) + 模型。
// 手动禁用(ManualDisabled)是永久状态，用于管理员手动关闭某个渠道的某个模型；
// State/TripCount/ConsecutiveFailures 为运行时熔断状态，重启后可从 DB 恢复。
type CircuitBreaker struct {
	ID                  int       `json:"id" gorm:"primaryKey"`
	ChannelID           int       `json:"channel_id" gorm:"not null;uniqueIndex:idx_cb_channel_model"`
	ChannelName         string    `json:"channel_name"`
	ModelName           string    `json:"model_name" gorm:"not null;uniqueIndex:idx_cb_channel_model"`
	State               int       `json:"state" gorm:"not null;default:0"`
	TripCount           int       `json:"trip_count" gorm:"default:0"`
	ConsecutiveFailures int64     `json:"consecutive_failures" gorm:"default:0"`
	ManualDisabled      bool      `json:"manual_disabled" gorm:"default:false"`
	LastErrorTime       int64     `json:"last_error_time"` // 上次报错时间（Unix 秒）
	LastError           string    `json:"last_error"`      // 上次报错信息
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// CircuitBreakerManualRequest 手动启用/禁用熔断请求
type CircuitBreakerManualRequest struct {
	ChannelID int    `json:"channel_id" binding:"required"`
	ModelName string `json:"model_name" binding:"required"`
	Disabled  bool   `json:"disabled"`
}

// CircuitBreakerResetRequest 取消熔断请求
type CircuitBreakerResetRequest struct {
	ChannelID int    `json:"channel_id" binding:"required"`
	ModelName string `json:"model_name" binding:"required"`
}
