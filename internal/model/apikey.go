package model

type APIKey struct {
	ID              int     `json:"id" gorm:"primaryKey"`
	Name            string  `json:"name" gorm:"not null"`
	APIKey          string  `json:"api_key" gorm:"not null"`
	Enabled         bool    `json:"enabled" gorm:"default:true"`
	ExpireAt        int64   `json:"expire_at,omitempty"`
	MaxCost         float64 `json:"max_cost,omitempty"`
	SupportedModels string  `json:"supported_models,omitempty"`
	AutoResetQuota  bool    `json:"auto_reset_quota" gorm:"default:false"`
	ResetDuration   int64   `json:"reset_duration" gorm:"default:0"`
	NextResetTime   int64   `json:"next_reset_time" gorm:"default:0"`
}
