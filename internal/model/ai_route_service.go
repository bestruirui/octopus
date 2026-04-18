package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type AIRouteServiceConfig struct {
	Name    string `json:"name,omitempty"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
	Enabled *bool  `json:"enabled,omitempty"`
}

func (c AIRouteServiceConfig) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

func (AIRouteServiceConfig) TableName() string { return "-" }

func ValidateAIRouteServiceConfigs(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var services []AIRouteServiceConfig
	if err := json.Unmarshal([]byte(raw), &services); err != nil {
		return fmt.Errorf("ai route services must be a valid JSON array")
	}

	enabledCount := 0
	for i, service := range services {
		if service.IsEnabled() {
			enabledCount++
		}

		baseURL := strings.TrimSpace(service.BaseURL)
		apiKey := strings.TrimSpace(service.APIKey)
		modelName := strings.TrimSpace(service.Model)
		if baseURL == "" {
			return fmt.Errorf("ai route service #%d base_url is required", i+1)
		}
		if apiKey == "" {
			return fmt.Errorf("ai route service #%d api_key is required", i+1)
		}
		if modelName == "" {
			return fmt.Errorf("ai route service #%d model is required", i+1)
		}

		parsedURL, err := url.Parse(baseURL)
		if err != nil {
			return fmt.Errorf("ai route service #%d base_url is invalid: %w", i+1, err)
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("ai route service #%d base_url scheme must be http or https", i+1)
		}
		if parsedURL.Host == "" {
			return fmt.Errorf("ai route service #%d base_url must have a host", i+1)
		}
	}

	if len(services) > 0 && enabledCount == 0 {
		return fmt.Errorf("ai route services must include at least one enabled service")
	}

	return nil
}
