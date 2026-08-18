package model

import (
	"time"

	"github.com/looplj/axonhub/llm"
)

type AutoGroupType int

const (
	AutoGroupTypeNone  AutoGroupType = 0 //不自动分组
	AutoGroupTypeFuzzy AutoGroupType = 1 //模糊匹配
	AutoGroupTypeExact AutoGroupType = 2 //准确匹配
	AutoGroupTypeRegex AutoGroupType = 3 //正则匹配
)

const ChannelTypeDoubao llm.APIFormat = "doubao"

type Channel struct {
	ID            int            `json:"id" gorm:"primaryKey"`
	Name          string         `json:"name" gorm:"unique;not null"`
	Type          llm.APIFormat  `json:"type"`
	Enabled       bool           `json:"enabled" gorm:"default:true"`
	BaseUrls      []BaseUrl      `json:"base_urls" gorm:"serializer:json"`
	Keys          []ChannelKey   `json:"keys" gorm:"foreignKey:ChannelID"`
	Model         string         `json:"model"`
	CustomModel   string         `json:"custom_model"`
	Proxy         bool           `json:"proxy" gorm:"default:false"`
	AutoSync      bool           `json:"auto_sync" gorm:"default:false"`
	AutoGroup     AutoGroupType  `json:"auto_group" gorm:"default:0"`
	CustomHeader  []CustomHeader `json:"custom_header" gorm:"serializer:json"`
	ParamOverride *string        `json:"param_override"`
	ChannelProxy  *string        `json:"channel_proxy"`
	Stats         *StatsChannel  `json:"stats,omitempty" gorm:"foreignKey:ChannelID"`
	MatchRegex    *string        `json:"match_regex"`
	BalanceQuery  *BalanceQuery  `json:"balance_query,omitempty" gorm:"serializer:json"` // BalanceQuery 是余额查询配置。
	Balance       *Balance       `json:"balance,omitempty" gorm:"serializer:json"`      // Balance 是最近一次余额查询结果。
}

// BalanceQueryType 余额查询方式。Custom 是自定义脚本;DeepSeek 是内置预设,后端按固定脚本查询。
type BalanceQueryType string

const (
	BalanceQueryTypeCustom   BalanceQueryType = "custom"   // Custom 表示自定义查询脚本。
	BalanceQueryTypeDeepSeek BalanceQueryType = "deepseek" // DeepSeek 表示 DeepSeek 官方余额查询。
)

// BalanceQuery 渠道余额查询配置。
// Custom 类型的 Script 形如 ({request: {...}, extractor: function (response) {...}});预设类型无需填写脚本。
type BalanceQuery struct {
	Enabled  bool             `json:"enabled"`            // Enabled 表示是否启用余额查询。
	Type     BalanceQueryType `json:"type,omitempty"`     // Type 是查询方式,为空视为 custom。
	Script   string           `json:"script,omitempty"`   // Script 是完整查询脚本,仅 custom 类型使用。
	Timeout  int              `json:"timeout,omitempty"`  // Timeout 是单次查询超时(秒),0 表示默认 10。
	Interval *int             `json:"interval,omitempty"` // Interval 是自动查询间隔(分钟),未设置默认 5,0 表示不自动查询。
}

// Balance 渠道余额快照。
type Balance struct {
	Total     float64 `json:"total"`               // Total 是总额度/总金额。
	Used      float64 `json:"used"`                // Used 是已用额度/已用金额。
	Remaining float64 `json:"remaining"`           // Remaining 是剩余额度/剩余金额。
	Unit      string  `json:"unit"`                // Unit 是单位:quota / CNY / USD 等。
	PlanName  string  `json:"plan_name,omitempty"` // PlanName 是套餐名称(由提取函数返回)。
	Extra     string  `json:"extra,omitempty"`     // Extra 是扩展展示文本(由提取函数返回)。
	UpdatedAt int64   `json:"updated_at"`          // UpdatedAt 是查询时间(unix 秒)。
	Error     string  `json:"error,omitempty"`     // Error 是最近一次查询失败原因。
}

type BaseUrl struct {
	URL   string `json:"url"`
	Delay int    `json:"delay"`
}

type CustomHeader struct {
	HeaderKey   string `json:"header_key"`
	HeaderValue string `json:"header_value"`
}

type ChannelKey struct {
	ID               int     `json:"id" gorm:"primaryKey"`
	ChannelID        int     `json:"channel_id"`
	Enabled          bool    `json:"enabled" gorm:"default:true"`
	ChannelKey       string  `json:"channel_key"`
	StatusCode       int     `json:"status_code"`
	LastUseTimeStamp int64   `json:"last_use_time_stamp"`
	TotalCost        float64 `json:"total_cost"`
	Remark           string  `json:"remark"`
}

// ChannelUpdateRequest 渠道更新请求 - 仅包含变更的数据
type ChannelUpdateRequest struct {
	ID            int             `json:"id" binding:"required"`
	Name          *string         `json:"name,omitempty"`
	Type          *llm.APIFormat  `json:"type,omitempty"`
	Enabled       *bool           `json:"enabled,omitempty"`
	BaseUrls      *[]BaseUrl      `json:"base_urls,omitempty"`
	Model         *string         `json:"model,omitempty"`
	CustomModel   *string         `json:"custom_model,omitempty"`
	Proxy         *bool           `json:"proxy,omitempty"`
	AutoSync      *bool           `json:"auto_sync,omitempty"`
	AutoGroup     *AutoGroupType  `json:"auto_group,omitempty"`
	CustomHeader  *[]CustomHeader `json:"custom_header,omitempty"`
	ChannelProxy  *string         `json:"channel_proxy,omitempty"`
	ParamOverride *string         `json:"param_override,omitempty"`
	MatchRegex    *string         `json:"match_regex,omitempty"`
	BalanceQuery  *BalanceQuery   `json:"balance_query,omitempty"` // BalanceQuery 是新的余额查询配置。

	KeysToAdd    []ChannelKeyAddRequest    `json:"keys_to_add,omitempty"`
	KeysToUpdate []ChannelKeyUpdateRequest `json:"keys_to_update,omitempty"`
	KeysToDelete []int                     `json:"keys_to_delete,omitempty"`
}

type ChannelKeyAddRequest struct {
	Enabled    bool   `json:"enabled"`
	ChannelKey string `json:"channel_key" binding:"required"`
	Remark     string `json:"remark"`
}

type ChannelKeyUpdateRequest struct {
	ID         int     `json:"id" binding:"required"`
	Enabled    *bool   `json:"enabled,omitempty"`
	ChannelKey *string `json:"channel_key,omitempty"`
	Remark     *string `json:"remark,omitempty"`
}

func (c *Channel) GetBaseUrl() string {
	if c == nil || len(c.BaseUrls) == 0 {
		return ""
	}

	bestURL := ""
	bestDelay := 0
	bestSet := false

	for _, bu := range c.BaseUrls {
		if bu.URL == "" {
			continue
		}
		if !bestSet || bu.Delay < bestDelay {
			bestURL = bu.URL
			bestDelay = bu.Delay
			bestSet = true
		}
	}

	return bestURL
}

func (c *Channel) GetChannelKey() ChannelKey {
	if c == nil || len(c.Keys) == 0 {
		return ChannelKey{}
	}

	nowSec := time.Now().Unix()

	best := ChannelKey{}
	bestCost := 0.0
	bestSet := false

	for _, k := range c.Keys {
		if !k.Enabled || k.ChannelKey == "" {
			continue
		}
		if k.StatusCode == 429 && k.LastUseTimeStamp > 0 {
			if nowSec-k.LastUseTimeStamp < int64(5*time.Minute/time.Second) {
				continue
			}
		}
		if !bestSet || k.TotalCost < bestCost {
			best = k
			bestCost = k.TotalCost
			bestSet = true
		}
	}

	if !bestSet {
		return ChannelKey{}
	}
	return best
}
