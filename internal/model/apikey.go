package model

type APIKey struct {
	ID              int     `json:"id" gorm:"primaryKey"`
	Name            string  `json:"name" gorm:"not null"`
	APIKey          string  `json:"api_key" gorm:"uniqueIndex"`
	Enabled         bool    `json:"enabled" gorm:"default:true"`
	ExpireAt        int64   `json:"expire_at"`                                    // 过期时间 unix 秒，0=永不过期
	MaxCost         float64 `json:"max_cost"`                                     // 最大花费，0=不限
	SupportedModels string  `json:"supported_models" gorm:"default:''"`           // 支持的模型名，逗号分隔，空=全部
	RateLimitRPM    int     `json:"rate_limit_rpm" gorm:"default:0"`             // 每分钟请求限制，0=不限
}
