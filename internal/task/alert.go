package task

import (
	"context"
	"fmt"
	"time"

	"github.com/lingyuins/octopus/internal/helper"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/utils/log"
)

const TaskAlertEvaluate = "alert_evaluate"

const (
	alertNotifyLanguageDefault = "en"
	alertStateFiring           = "firing"
	alertStateResolved         = "resolved"
)

func EvaluateAlertRules() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rules, err := op.AlertRuleList(ctx)
	if err != nil {
		log.Warnf("alert evaluate: failed to list rules: %v", err)
		return
	}

	channels, err := op.AlertNotifChannelList(ctx)
	if err != nil {
		log.Warnf("alert evaluate: failed to list channels: %v", err)
		channels = nil
	}
	channelMap := make(map[int]*model.AlertNotifChannel)
	for i := range channels {
		channelMap[channels[i].ID] = &channels[i]
	}

	now := time.Now().UnixMilli()
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		currentState := op.AlertStateGet(rule.ID)
		firing := evaluateRule(&rule)
		prevState := currentState.State

		switch {
		case firing && prevState != model.AlertStateFiring:
			op.AlertStateSet(rule.ID, model.AlertStateFiring)
			currentState = op.AlertStateGet(rule.ID)
			notifyAlert(&rule, channelMap, alertStateFiring, currentState)
			recordHistory(&rule, model.AlertStateFiring, "alert triggered")

		case !firing && prevState == model.AlertStateFiring:
			op.AlertStateSet(rule.ID, model.AlertStateResolved)
			currentState = op.AlertStateGet(rule.ID)
			notifyAlert(&rule, channelMap, alertStateResolved, currentState)
			recordHistory(&rule, model.AlertStateResolved, "alert resolved")

		default:
			op.AlertStateSet(rule.ID, prevState) // update LastCheckedAt
		}
		_ = now
	}
}

func evaluateRule(rule *model.AlertRule) bool {
	// For now, check error rate using recent stats
	switch rule.ConditionType {
	case model.AlertConditionErrorRate:
		// Check if error count exceeds threshold
		return evaluateErrorRate(rule)
	case model.AlertConditionCostThreshold:
		return evaluateCostThreshold(rule)
	case model.AlertConditionChannelDown:
		return evaluateChannelDown(rule)
	case model.AlertConditionQuotaExceeded:
		return evaluateQuotaExceeded(rule)
	default:
		return false
	}
}

func evaluateErrorRate(rule *model.AlertRule) bool {
	stats := op.StatsTotalGet()
	if stats.RequestSuccess+stats.RequestFailed == 0 {
		return false
	}
	rate := float64(stats.RequestFailed) / float64(stats.RequestSuccess+stats.RequestFailed) * 100
	return rate >= rule.Threshold
}

func evaluateCostThreshold(rule *model.AlertRule) bool {
	stats := op.StatsTotalGet()
	totalCost := stats.StatsMetrics.InputCost + stats.StatsMetrics.OutputCost
	return totalCost >= rule.Threshold
}

func evaluateChannelDown(rule *model.AlertRule) bool {
	if rule.ScopeChannelID == 0 {
		return false
	}
	channels, err := op.ChannelList(context.Background())
	if err != nil {
		return false
	}
	for _, ch := range channels {
		if ch.ID == rule.ScopeChannelID {
			return !ch.Enabled
		}
	}
	return true // channel not found = down
}

func evaluateQuotaExceeded(rule *model.AlertRule) bool {
	if rule.ScopeAPIKeyID == 0 {
		return false
	}
	stats := op.StatsAPIKeyGet(rule.ScopeAPIKeyID)
	totalCost := stats.StatsMetrics.InputCost + stats.StatsMetrics.OutputCost
	return totalCost >= rule.Threshold
}

func notifyAlert(rule *model.AlertRule, channelMap map[int]*model.AlertNotifChannel, state string, current model.AlertStateRecord) {
	ch, ok := channelMap[rule.NotifChannelID]
	if !ok || ch == nil {
		return
	}
	language := resolveAlertNotifyLanguage()

	payload := helper.AlertWebhookPayload{
		RuleID:        rule.ID,
		RuleName:      rule.Name,
		ConditionType: rule.ConditionType,
		State:         state,
		Message:       buildAlertNotificationMessage(rule.Name, state, language),
		Threshold:     rule.Threshold,
		Time:          time.UnixMilli(current.LastFiredAt).Format(time.RFC3339),
	}

	if err := helper.SendNotification(ch, payload); err != nil {
		log.Warnf("alert notify: failed to send notification via %s for rule %d: %v", ch.Type, rule.ID, err)
	}
}

func recordHistory(rule *model.AlertRule, state model.AlertState, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	entry := &model.AlertHistory{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		State:    state,
		Message:  message,
	}
	if err := op.AlertHistoryAdd(ctx, entry); err != nil {
		log.Warnf("alert history: failed to record for rule %d: %v", rule.ID, err)
	}
}

func resolveAlertNotifyLanguage() string {
	language, err := op.SettingGetString(model.SettingKeyAlertNotifyLanguage)
	if err != nil {
		return alertNotifyLanguageDefault
	}
	return normalizeAlertNotifyLanguage(language)
}

func normalizeAlertNotifyLanguage(language string) string {
	switch language {
	case "zh-Hans", "zh-Hant", "en":
		return language
	default:
		return alertNotifyLanguageDefault
	}
}

func buildAlertNotificationMessage(ruleName, state, language string) string {
	switch normalizeAlertNotifyLanguage(language) {
	case "zh-Hans":
		switch state {
		case alertStateFiring:
			return fmt.Sprintf("告警规则 \"%s\" 已触发", ruleName)
		case alertStateResolved:
			return fmt.Sprintf("告警规则 \"%s\" 已恢复", ruleName)
		default:
			return fmt.Sprintf("告警规则 \"%s\" 状态已变更为 %s", ruleName, state)
		}
	case "zh-Hant":
		switch state {
		case alertStateFiring:
			return fmt.Sprintf("告警規則 \"%s\" 已觸發", ruleName)
		case alertStateResolved:
			return fmt.Sprintf("告警規則 \"%s\" 已恢復", ruleName)
		default:
			return fmt.Sprintf("告警規則 \"%s\" 狀態已變更為 %s", ruleName, state)
		}
	default:
		return fmt.Sprintf("Alert '%s' is %s", ruleName, state)
	}
}
