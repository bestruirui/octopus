package model

import "testing"

func TestValidateAIRouteServiceConfigs(t *testing.T) {
	valid := `[{"name":"svc-a","base_url":"https://example.com/v1","api_key":"key","model":"gpt-4o"}]`
	if err := ValidateAIRouteServiceConfigs(valid); err != nil {
		t.Fatalf("ValidateAIRouteServiceConfigs(valid) error = %v, want nil", err)
	}

	if err := ValidateAIRouteServiceConfigs(`[]`); err != nil {
		t.Fatalf("ValidateAIRouteServiceConfigs(emptyArray) error = %v, want nil", err)
	}

	if err := ValidateAIRouteServiceConfigs(`{"base_url":"https://example.com"}`); err == nil {
		t.Fatal("ValidateAIRouteServiceConfigs(object) error = nil, want non-nil")
	}

	invalidScheme := `[{"base_url":"ftp://example.com","api_key":"key","model":"gpt-4o"}]`
	if err := ValidateAIRouteServiceConfigs(invalidScheme); err == nil {
		t.Fatal("ValidateAIRouteServiceConfigs(invalidScheme) error = nil, want non-nil")
	}

	allDisabled := `[{"name":"svc-a","base_url":"https://example.com/v1","api_key":"key","model":"gpt-4o","enabled":false}]`
	if err := ValidateAIRouteServiceConfigs(allDisabled); err == nil {
		t.Fatal("ValidateAIRouteServiceConfigs(allDisabled) error = nil, want non-nil")
	}
}
