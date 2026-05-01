package model

import (
	"testing"

	"github.com/lingyuins/octopus/internal/transformer/outbound"
)

func TestRequestRewriteConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *RequestRewriteConfig
		channelType outbound.OutboundType
		wantErr     string
	}{
		{
			name:        "nil config is valid",
			cfg:         nil,
			channelType: outbound.OutboundTypeOpenAIChat,
		},
		{
			name: "disabled config is valid",
			cfg: &RequestRewriteConfig{
				Enabled: false,
			},
			channelType: outbound.OutboundTypeOpenAIChat,
		},
		{
			name: "missing profile when enabled",
			cfg: &RequestRewriteConfig{
				Enabled: true,
			},
			channelType: outbound.OutboundTypeOpenAIChat,
			wantErr:     "request rewrite profile is required when enabled",
		},
		{
			name: "unsupported profile",
			cfg: &RequestRewriteConfig{
				Enabled: true,
				Profile: "unknown",
			},
			channelType: outbound.OutboundTypeOpenAIChat,
			wantErr:     "unsupported request rewrite profile: unknown",
		},
		{
			name: "unsupported channel type",
			cfg: &RequestRewriteConfig{
				Enabled: true,
				Profile: RequestRewriteProfileOpenAIChatCompat,
			},
			channelType: outbound.OutboundTypeOpenAIResponse,
			wantErr:     "request rewrite profile openai_chat_compat is not supported for channel type 1",
		},
		{
			name: "unsupported tool role strategy",
			cfg: &RequestRewriteConfig{
				Enabled:          true,
				Profile:          RequestRewriteProfileOpenAIChatCompat,
				ToolRoleStrategy: "broken",
			},
			channelType: outbound.OutboundTypeOpenAIChat,
			wantErr:     "unsupported tool role strategy: broken",
		},
		{
			name: "unsupported system message strategy",
			cfg: &RequestRewriteConfig{
				Enabled:               true,
				Profile:               RequestRewriteProfileOpenAIChatCompat,
				SystemMessageStrategy: "broken",
			},
			channelType: outbound.OutboundTypeOpenAIChat,
			wantErr:     "unsupported system message strategy: broken",
		},
		{
			name: "valid openai chat compat config",
			cfg: &RequestRewriteConfig{
				Enabled:               true,
				Profile:               RequestRewriteProfileOpenAIChatCompat,
				ToolRoleStrategy:      ToolRoleStrategyStringifyToUser,
				SystemMessageStrategy: SystemMessageStrategyMerge,
			},
			channelType: outbound.OutboundTypeOpenAIChat,
		},
		{
			name: "valid mimo chat compat config",
			cfg: &RequestRewriteConfig{
				Enabled:               true,
				Profile:               RequestRewriteProfileOpenAIChatCompat,
				ToolRoleStrategy:      ToolRoleStrategyKeep,
				SystemMessageStrategy: SystemMessageStrategyKeep,
			},
			channelType: outbound.OutboundTypeMimo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate(tt.channelType)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() error = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("Validate() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}
