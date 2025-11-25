package model

type ChannelType int8

const (
	ChannelTypeOpenAIChat ChannelType = iota
	ChannelTypeOpenAIResponse
	ChannelTypeAnthropic
	ChannelTypeOneAPI
)

type Channel struct {
	ID              int         `json:"id" gorm:"primaryKey"`
	Name            string      `json:"name" gorm:"unique;not null"`
	Type            ChannelType `json:"type" gorm:"default:0"`
	Enabled         bool        `json:"enabled" gorm:"default:true"`
	BaseURL         string      `json:"base_url" gorm:"not null"`
	Key             string      `json:"key" gorm:"not null"`
	Model           string      `json:"model" gorm:"not null"`
	Proxy           bool        `json:"proxy" gorm:"default:false"`
	UsedInputToken  int64       `json:"used_input_token" gorm:"bigint;default:0"`
	UsedOutputToken int64       `json:"used_output_token" gorm:"bigint;default:0"`
	UsedMoney       float64     `json:"used_money" gorm:"default:0"`
	RequestCount    int64       `json:"request_count" gorm:"bigint;default:0"`
	RequestSuccess  int64       `json:"request_success" gorm:"bigint;default:0"`
	RequestFailed   int64       `json:"request_failed" gorm:"bigint;default:0"`
}
