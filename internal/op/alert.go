package op

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
)

var alertStateCache sync.Map // int(ruleID) -> model.AlertStateRecord

func AlertRuleList(ctx context.Context) ([]model.AlertRule, error) {
	var rules []model.AlertRule
	if err := db.GetDB().WithContext(ctx).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func AlertRuleCreate(ctx context.Context, rule *model.AlertRule) error {
	return db.GetDB().WithContext(ctx).Create(rule).Error
}

func AlertRuleUpdate(ctx context.Context, rule *model.AlertRule) error {
	return db.GetDB().WithContext(ctx).Save(rule).Error
}

func AlertRuleDelete(ctx context.Context, id int) error {
	res := db.GetDB().WithContext(ctx).Delete(&model.AlertRule{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("alert rule not found")
	}
	return nil
}

func AlertNotifChannelList(ctx context.Context) ([]model.AlertNotifChannel, error) {
	var channels []model.AlertNotifChannel
	if err := db.GetDB().WithContext(ctx).Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func AlertNotifChannelCreate(ctx context.Context, ch *model.AlertNotifChannel) error {
	return db.GetDB().WithContext(ctx).Create(ch).Error
}

func AlertNotifChannelUpdate(ctx context.Context, ch *model.AlertNotifChannel) error {
	return db.GetDB().WithContext(ctx).Save(ch).Error
}

func AlertNotifChannelDelete(ctx context.Context, id int) error {
	return db.GetDB().WithContext(ctx).Delete(&model.AlertNotifChannel{}, id).Error
}

func AlertStateGet(ruleID int) model.AlertStateRecord {
	if v, ok := alertStateCache.Load(ruleID); ok {
		return v.(model.AlertStateRecord)
	}
	return model.AlertStateRecord{RuleID: ruleID, State: model.AlertStateOK}
}

func AlertStateSet(ruleID int, state model.AlertState) {
	record := AlertStateGet(ruleID)
	record.State = state
	now := timeNow()
	if state == model.AlertStateFiring {
		record.LastFiredAt = now
		record.FiredCount++
	} else if state == model.AlertStateResolved {
		record.LastResolvedAt = now
	}
	record.LastCheckedAt = now
	alertStateCache.Store(ruleID, record)
}

func AlertHistoryList(ctx context.Context, limit int) ([]model.AlertHistory, error) {
	if limit <= 0 {
		limit = 100
	}
	var history []model.AlertHistory
	if err := db.GetDB().WithContext(ctx).Order("time DESC").Limit(limit).Find(&history).Error; err != nil {
		return nil, err
	}
	return history, nil
}

func AlertHistoryAdd(ctx context.Context, entry *model.AlertHistory) error {
	return db.GetDB().WithContext(ctx).Create(entry).Error
}

var timeNow = func() int64 { return time.Now().UnixMilli() }
