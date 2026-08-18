package task

import (
	"context"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/utils/log"
)

// BalanceQueryCheckInterval 余额查询检查周期:每分钟检查一次各渠道是否到达各自的自动查询间隔。
const BalanceQueryCheckInterval = 1 * time.Minute

// BalanceQueryChannel 查询指定渠道的余额并落库。
// 查询失败时也会把带 Error 的余额快照写入渠道,方便面板展示失败原因。
func BalanceQueryChannel(id int, ctx context.Context) (*model.Balance, error) {
	channel, err := op.ChannelGet(id, ctx)
	if err != nil {
		return nil, err
	}

	balance, err := helper.FetchBalance(ctx, *channel)
	if err != nil {
		balance = &model.Balance{
			Error: err.Error(),
		}
		log.Warnf("failed to query balance for channel %d (%s): %v", id, channel.Name, err)
	}
	if balance.UpdatedAt == 0 {
		balance.UpdatedAt = time.Now().Unix()
	}
	if err := op.BalanceSave(id, balance, ctx); err != nil {
		return nil, err
	}
	return balance, nil
}

// BalanceQueryTask 检查全部启用了余额查询的渠道,对到达自动查询间隔的渠道更新余额快照。
// 间隔配置为 0 的渠道不参与自动查询,只支持手动刷新。
func BalanceQueryTask() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	channels, err := op.ChannelList(ctx)
	if err != nil {
		log.Warnf("failed to list channels for balance query: %v", err)
		return
	}
	now := time.Now().Unix()
	for i := range channels {
		channel := &channels[i]
		if !channel.Enabled || channel.BalanceQuery == nil || !channel.BalanceQuery.Enabled {
			continue
		}
		// 未设置间隔时按默认 5 分钟自动查询;显式设置为 0 才不自动查询。
		interval := 5
		if channel.BalanceQuery.Interval != nil {
			interval = *channel.BalanceQuery.Interval
		}
		if interval <= 0 {
			continue
		}
		lastQueried := int64(0)
		if channel.Balance != nil {
			lastQueried = channel.Balance.UpdatedAt
		}
		if lastQueried > 0 && now-lastQueried < int64(interval)*60 {
			continue
		}
		if _, err := BalanceQueryChannel(channel.ID, ctx); err != nil {
			log.Warnf("failed to query balance for channel %d: %v", channel.ID, err)
		}
	}
}
