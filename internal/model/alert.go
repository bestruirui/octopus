package model

// AlertRuleConditionType defines the trigger condition for an alert rule.
type AlertRuleConditionType string

const (
	AlertConditionCostThreshold AlertRuleConditionType = "cost_threshold"
	AlertConditionErrorRate     AlertRuleConditionType = "error_rate"
	AlertConditionQuotaExceeded AlertRuleConditionType = "quota_exceeded"
	AlertConditionChannelDown   AlertRuleConditionType = "channel_down"
)

// AlertRule defines an alert rule with condition, threshold, and notification channel.
type AlertRule struct {
	ID             int                    `json:"id" gorm:"primaryKey"`
	Name           string                 `json:"name" gorm:"not null"`
	Enabled        bool                   `json:"enabled" gorm:"default:true"`
	ConditionType  AlertRuleConditionType `json:"condition_type" gorm:"not null"`
	Threshold      float64                `json:"threshold"`
	ConditionJSON  string                 `json:"condition_json,omitempty"`
	NotifChannelID int                    `json:"notif_channel_id"`
	CooldownSec    int                    `json:"cooldown_sec" gorm:"default:300"`
	ScopeChannelID int                    `json:"scope_channel_id,omitempty"`
	ScopeAPIKeyID  int                    `json:"scope_api_key_id,omitempty"`
}

// AlertNotifChannel defines a notification channel (currently webhook only).
type AlertNotifChannel struct {
	ID      int    `json:"id" gorm:"primaryKey"`
	Name    string `json:"name" gorm:"not null"`
	Type    string `json:"type" gorm:"not null;default:'webhook'"`
	URL     string `json:"url" gorm:"not null"`
	Secret  string `json:"secret,omitempty"`
	Headers string `json:"headers,omitempty"`
}

// AlertState represents the current firing state of an alert rule.
type AlertState int

const (
	AlertStateOK       AlertState = 0
	AlertStateFiring   AlertState = 1
	AlertStateResolved AlertState = 2
)

// AlertStateRecord tracks the current state of an alert rule.
type AlertStateRecord struct {
	RuleID         int        `json:"rule_id" gorm:"primaryKey"`
	State          AlertState `json:"state"`
	LastFiredAt    int64      `json:"last_fired_at"`
	LastResolvedAt int64      `json:"last_resolved_at"`
	LastCheckedAt  int64      `json:"last_checked_at"`
	FiredCount     int64      `json:"fired_count"`
}

// AlertHistory records a single alert event.
type AlertHistory struct {
	ID         int64      `json:"id" gorm:"primaryKey"`
	RuleID     int        `json:"rule_id"`
	RuleName   string     `json:"rule_name"`
	State      AlertState `json:"state"`
	Message    string     `json:"message"`
	DetailJSON string     `json:"detail_json,omitempty"`
	Time       int64      `json:"time" gorm:"autoCreateTime:milli"`
}
