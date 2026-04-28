package model

type APIKey struct {
	ID                int     `json:"id" gorm:"primaryKey"`
	Name              string  `json:"name" gorm:"not null"`
	APIKey            string  `json:"api_key" gorm:"not null"`
	Enabled           bool    `json:"enabled" gorm:"default:true"`
	ExpireAt          int64   `json:"expire_at,omitempty"`
	MaxCost           float64 `json:"max_cost,omitempty"`
	SupportedModels   string  `json:"supported_models,omitempty"`
	RateLimitRPM      int     `json:"rate_limit_rpm,omitempty" gorm:"default:0"`
	RateLimitTPM      int     `json:"rate_limit_tpm,omitempty" gorm:"default:0"`
	PerModelQuotaJSON string  `json:"per_model_quota_json,omitempty" gorm:"column:per_model_quota_json"`
}
