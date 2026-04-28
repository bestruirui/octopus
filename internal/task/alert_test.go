package task

import "testing"

func TestNormalizeAlertNotifyLanguage(t *testing.T) {
	tests := []struct {
		name     string
		language string
		want     string
	}{
		{name: "simplified chinese", language: "zh-Hans", want: "zh-Hans"},
		{name: "traditional chinese", language: "zh-Hant", want: "zh-Hant"},
		{name: "english", language: "en", want: "en"},
		{name: "fallback", language: "ja", want: "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeAlertNotifyLanguage(tt.language); got != tt.want {
				t.Fatalf("normalizeAlertNotifyLanguage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildAlertNotificationMessage(t *testing.T) {
	tests := []struct {
		name     string
		ruleName string
		state    string
		language string
		want     string
	}{
		{name: "simplified firing", ruleName: "CPU", state: alertStateFiring, language: "zh-Hans", want: "告警规则 \"CPU\" 已触发"},
		{name: "simplified resolved", ruleName: "CPU", state: alertStateResolved, language: "zh-Hans", want: "告警规则 \"CPU\" 已恢复"},
		{name: "traditional firing", ruleName: "CPU", state: alertStateFiring, language: "zh-Hant", want: "告警規則 \"CPU\" 已觸發"},
		{name: "traditional resolved", ruleName: "CPU", state: alertStateResolved, language: "zh-Hant", want: "告警規則 \"CPU\" 已恢復"},
		{name: "english default", ruleName: "CPU", state: alertStateFiring, language: "en", want: "Alert 'CPU' is firing"},
		{name: "fallback language", ruleName: "CPU", state: alertStateResolved, language: "ja", want: "Alert 'CPU' is resolved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildAlertNotificationMessage(tt.ruleName, tt.state, tt.language); got != tt.want {
				t.Fatalf("buildAlertNotificationMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
