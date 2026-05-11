package model

import (
	"math/rand"
	"sort"
	"time"

	"github.com/bestruirui/octopus/internal/transformer/outbound"
)

type AutoGroupType int

const (
	AutoGroupTypeNone  AutoGroupType = 0 //不自动分组
	AutoGroupTypeFuzzy AutoGroupType = 1 //模糊匹配
	AutoGroupTypeExact AutoGroupType = 2 //准确匹配
	AutoGroupTypeRegex AutoGroupType = 3 //正则匹配
)

type ChannelKeyMode int

const (
	ChannelKeyModeCost           ChannelKeyMode = 0 // 按总成本最低
	ChannelKeyModeRoundRobin     ChannelKeyMode = 1 // 轮询
	ChannelKeyModeWeightedRandom ChannelKeyMode = 2 // 加权随机
)

type Channel struct {
	ID            int                   `json:"id" gorm:"primaryKey"`
	Name          string                `json:"name" gorm:"unique;not null"`
	Type          outbound.OutboundType `json:"type"`
	Enabled       bool                  `json:"enabled" gorm:"default:true"`
	BaseUrls      []BaseUrl             `json:"base_urls" gorm:"serializer:json"`
	Keys          []ChannelKey          `json:"keys" gorm:"foreignKey:ChannelID"`
	KeyMode       ChannelKeyMode        `json:"key_mode" gorm:"default:0"`
	Model         string                `json:"model"`
	CustomModel   string                `json:"custom_model"`
	Proxy         bool                  `json:"proxy" gorm:"default:false"`
	AutoSync      bool                  `json:"auto_sync" gorm:"default:false"`
	AutoGroup     AutoGroupType         `json:"auto_group" gorm:"default:0"`
	CustomHeader  []CustomHeader        `json:"custom_header" gorm:"serializer:json"`
	ParamOverride *string               `json:"param_override"`
	ChannelProxy  *string               `json:"channel_proxy"`
	Stats         *StatsChannel         `json:"stats,omitempty" gorm:"foreignKey:ChannelID"`
	MatchRegex    *string               `json:"match_regex"`
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
	Weight           int     `json:"weight" gorm:"default:1"`
	StatusCode       int     `json:"status_code"`
	LastUseTimeStamp int64   `json:"last_use_time_stamp"`
	TotalCost        float64 `json:"total_cost"`
	Remark           string  `json:"remark"`
}

// ChannelUpdateRequest 渠道更新请求 - 仅包含变更的数据
type ChannelUpdateRequest struct {
	ID            int                    `json:"id" binding:"required"`
	Name          *string                `json:"name,omitempty"`
	Type          *outbound.OutboundType `json:"type,omitempty"`
	Enabled       *bool                  `json:"enabled,omitempty"`
	BaseUrls      *[]BaseUrl             `json:"base_urls,omitempty"`
	KeyMode       *ChannelKeyMode        `json:"key_mode,omitempty"`
	Model         *string                `json:"model,omitempty"`
	CustomModel   *string                `json:"custom_model,omitempty"`
	Proxy         *bool                  `json:"proxy,omitempty"`
	AutoSync      *bool                  `json:"auto_sync,omitempty"`
	AutoGroup     *AutoGroupType         `json:"auto_group,omitempty"`
	CustomHeader  *[]CustomHeader        `json:"custom_header,omitempty"`
	ChannelProxy  *string                `json:"channel_proxy,omitempty"`
	ParamOverride *string                `json:"param_override,omitempty"`
	MatchRegex    *string                `json:"match_regex,omitempty"`

	KeysToAdd    []ChannelKeyAddRequest    `json:"keys_to_add,omitempty"`
	KeysToUpdate []ChannelKeyUpdateRequest `json:"keys_to_update,omitempty"`
	KeysToDelete []int                     `json:"keys_to_delete,omitempty"`
}

type ChannelKeyAddRequest struct {
	Enabled    bool   `json:"enabled"`
	ChannelKey string `json:"channel_key" binding:"required"`
	Weight     int    `json:"weight"`
	Remark     string `json:"remark"`
}

type ChannelKeyUpdateRequest struct {
	ID         int     `json:"id" binding:"required"`
	Enabled    *bool   `json:"enabled,omitempty"`
	ChannelKey *string `json:"channel_key,omitempty"`
	Weight     *int    `json:"weight,omitempty"`
	Remark     *string `json:"remark,omitempty"`
}

// ChannelFetchModelRequest is used by /channel/fetch-model (not persisted).
type ChannelFetchModelRequest struct {
	Type    outbound.OutboundType `json:"type" binding:"required"`
	BaseURL string                `json:"base_url" binding:"required"`
	Key     string                `json:"key" binding:"required"`
	Proxy   bool                  `json:"proxy"`
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

// GetChannelKeys 按 KeyMode 策略返回排序后的可用 key 列表。
// 第一个元素是按策略选中的最优 key，剩余元素按同策略排序作为 fallback。
// 当调用方希望在一个 channel 内进行 key 级重试时，直接遍历返回值即可。
func (c *Channel) GetChannelKeys() []ChannelKey {
	candidates := c.availableKeys()
	if len(candidates) == 0 {
		return candidates
	}

	switch c.KeyMode {
	case ChannelKeyModeRoundRobin:
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].LastUseTimeStamp < candidates[j].LastUseTimeStamp
		})
	case ChannelKeyModeWeightedRandom:
		c.weightedRandomSort(candidates)
	default: // ChannelKeyModeCost
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].TotalCost < candidates[j].TotalCost
		})
	}
	return candidates
}

// GetChannelKey 按 KeyMode 策略返回最优 key
func (c *Channel) GetChannelKey() ChannelKey {
	keys := c.GetChannelKeys()
	if len(keys) == 0 {
		return ChannelKey{}
	}
	return keys[0]
}

const key429CooldownSeconds int64 = 60

// availableKeys 过滤出可用的 key（enabled + 非空 + 非 429 冷却）
func (c *Channel) availableKeys() []ChannelKey {
	nowSec := time.Now().Unix()
	var result []ChannelKey
	for _, k := range c.Keys {
		if !k.Enabled || k.ChannelKey == "" {
			continue
		}
		if k.StatusCode == 429 && k.LastUseTimeStamp > 0 {
			if nowSec-k.LastUseTimeStamp < key429CooldownSeconds {
				continue
			}
		}
		result = append(result, k)
	}
	return result
}

// weightedRandomSort 按加权随机分数降序排序
func (c *Channel) weightedRandomSort(keys []ChannelKey) {
	if len(keys) <= 1 {
		return
	}
	totalWeight := 0
	for _, k := range keys {
		w := k.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}

	// 给每个 key 生成加权随机分数，分数越高越优先
	type scoredKey struct {
		key   ChannelKey
		score float64
	}
	scored := make([]scoredKey, len(keys))
	for i, k := range keys {
		w := k.Weight
		if w <= 0 {
			w = 1
		}
		// weight 越大，score 范围越大，被选中的概率越高
		scored[i] = scoredKey{
			key:   k,
			score: rand.Float64() * float64(w) / float64(totalWeight),
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	for i := range keys {
		keys[i] = scored[i].key
	}
}
