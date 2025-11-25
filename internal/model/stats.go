package model

import "time"

type StatsDaily struct {
	Date         time.Time `json:"date" gorm:"primaryKey"`
	InputToken   int64     `json:"input_token" gorm:"bigint;default:0"`
	OutputToken  int64     `json:"output_token" gorm:"bigint;default:0"`
	RequestCount int64     `json:"request_count" gorm:"bigint;default:0"`
	Money        int64     `json:"money" gorm:"bigint;default:0"`
	WaitTime     int64     `json:"wait_time" gorm:"bigint;default:0"`
}

type StatsModel struct {
	ID           int    `json:"id" gorm:"primaryKey"`
	Name         string `json:"name" gorm:"not null"`
	ChannelID    int    `json:"channel_id" gorm:"not null"`
	InputToken   int64  `json:"input_token" gorm:"bigint;default:0"`
	OutputToken  int64  `json:"output_token" gorm:"bigint;default:0"`
	RequestCount int64  `json:"request_count" gorm:"bigint;default:0"`
}
