package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/lingyuins/octopus/internal/model"
)

// AlertWebhookPayload is the JSON body sent to webhook endpoints on alert state changes.
type AlertWebhookPayload struct {
	RuleID        int                          `json:"rule_id"`
	RuleName      string                       `json:"rule_name"`
	ConditionType model.AlertRuleConditionType  `json:"condition_type"`
	State         string                       `json:"state"`
	Message       string                       `json:"message"`
	Threshold     float64                      `json:"threshold"`
	CurrentValue  float64                      `json:"current_value"`
	Time          string                       `json:"time"`
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

// SendGotify sends an alert notification to a Gotify server.
func SendGotify(channel *model.AlertNotifChannel, payload AlertWebhookPayload) error {
	var cfg model.GotifyConfig
	if channel.Config != "" {
		if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
			return fmt.Errorf("parse gotify config: %w", err)
		}
	}

	// Fallback: if server_url is empty, use channel.URL; if token is empty, use channel.Secret
	serverURL := cfg.ServerURL
	if serverURL == "" {
		serverURL = strings.TrimRight(channel.URL, "/")
	}
	token := cfg.Token
	if token == "" {
		token = channel.Secret
	}

	if serverURL == "" || token == "" {
		return fmt.Errorf("gotify: server_url and token are required")
	}

	priority := cfg.Priority
	if priority <= 0 {
		priority = 5
	}

	gotifyMsg := map[string]interface{}{
		"title":    fmt.Sprintf("Octopus Alert: %s", payload.RuleName),
		"message":  fmt.Sprintf("%s\n\nCondition: %s\nState: %s\nThreshold: %.2f\nTime: %s", payload.Message, payload.ConditionType, payload.State, payload.Threshold, payload.Time),
		"priority": priority,
	}

	msgBody, err := json.Marshal(gotifyMsg)
	if err != nil {
		return fmt.Errorf("marshal gotify message: %w", err)
	}

	endpoint := fmt.Sprintf("%s/message?token=%s", strings.TrimRight(serverURL, "/"), token)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(msgBody))
	if err != nil {
		return fmt.Errorf("create gotify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send gotify: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("gotify responded %d", resp.StatusCode)
	}
	return nil
}

// SendEmail sends an alert notification via SMTP email.
func SendEmail(channel *model.AlertNotifChannel, payload AlertWebhookPayload) error {
	var cfg model.EmailConfig
	if channel.Config != "" {
		if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
			return fmt.Errorf("parse email config: %w", err)
		}
	}

	if cfg.SMTPHost == "" || cfg.From == "" || cfg.To == "" {
		return fmt.Errorf("email: smtp_host, from, and to are required")
	}

	port := cfg.SMTPPort
	if port == 0 {
		port = 587
	}

	subject := fmt.Sprintf("Octopus Alert: %s - %s", payload.RuleName, payload.State)
	body := fmt.Sprintf(
		"Rule: %s\nCondition: %s\nState: %s\nMessage: %s\nThreshold: %.2f\nTime: %s\n",
		payload.RuleName, payload.ConditionType, payload.State, payload.Message, payload.Threshold, payload.Time,
	)

	// Build MIME message
	fromHeader := cfg.From
	toAddrs := strings.Split(cfg.To, ",")
	for i, a := range toAddrs {
		toAddrs[i] = strings.TrimSpace(a)
	}

	var msg strings.Builder
	msg.WriteString("From: " + fromHeader + "\r\n")
	msg.WriteString("To: " + cfg.To + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, port)
	useTLS := cfg.UseTLS
	if !useTLS && port == 465 {
		useTLS = true
	}

	if cfg.Username == "" && cfg.Password == "" {
		// No auth - try plain send
		err := smtp.SendMail(addr, nil, fromHeader, toAddrs, []byte(msg.String()))
		if err != nil {
			return fmt.Errorf("send email (no auth): %w", err)
		}
		return nil
	}

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	err := smtp.SendMail(addr, auth, fromHeader, toAddrs, []byte(msg.String()))
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// SendNotification dispatches an alert notification to the appropriate channel based on type.
func SendNotification(channel *model.AlertNotifChannel, payload AlertWebhookPayload) error {
	switch model.AlertNotifChannelType(channel.Type) {
	case model.AlertNotifGotify:
		return SendGotify(channel, payload)
	case model.AlertNotifEmail:
		return SendEmail(channel, payload)
	case model.AlertNotifWebhook:
		fallthrough
	default:
		return SendWebhook(channel, payload)
	}
}

// FormatEmailAddress validates and formats an email address.
func FormatEmailAddress(addr string) (string, error) {
	a, err := mail.ParseAddress(addr)
	if err != nil {
		return "", fmt.Errorf("invalid email address %q: %w", addr, err)
	}
	return a.String(), nil
}
