package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lingyuins/octopus/internal/model"
)

// AlertWebhookPayload is the JSON body sent to webhook endpoints on alert state changes.
type AlertWebhookPayload struct {
	RuleID        int                       `json:"rule_id"`
	RuleName      string                    `json:"rule_name"`
	ConditionType model.AlertRuleConditionType `json:"condition_type"`
	State         string                    `json:"state"`
	Message       string                    `json:"message"`
	Threshold     float64                   `json:"threshold"`
	CurrentValue  float64                   `json:"current_value"`
	Time          string                    `json:"time"`
}

// SendWebhook sends an alert notification to the configured webhook URL.
func SendWebhook(channel *model.AlertNotifChannel, payload AlertWebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", channel.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if channel.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+channel.Secret)
	}
	if channel.Headers != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(channel.Headers), &headers); err == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook responded %d", resp.StatusCode)
	}
	return nil
}
