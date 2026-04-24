package rewrite

import (
	appmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/transformer/outbound"
)

type EffectiveConfig struct {
	Profile                             appmodel.RequestRewriteProfile
	FlattenTextBlockArrays              bool
	NilContentAsEmptyString             bool
	EnsureAssistantContentWithToolCalls bool
	ToolRoleStrategy                    appmodel.ToolRoleStrategy
	SystemMessageStrategy               appmodel.SystemMessageStrategy
}

func Resolve(channelType outbound.OutboundType, cfg *appmodel.RequestRewriteConfig) (*EffectiveConfig, bool, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, false, nil
	}
	if err := cfg.Validate(channelType); err != nil {
		return nil, false, err
	}

	effective := &EffectiveConfig{
		Profile:                             cfg.Profile,
		FlattenTextBlockArrays:              true,
		NilContentAsEmptyString:             true,
		EnsureAssistantContentWithToolCalls: true,
		ToolRoleStrategy:                    appmodel.ToolRoleStrategyKeep,
		SystemMessageStrategy:               appmodel.SystemMessageStrategyKeep,
	}

	if cfg.ToolRoleStrategy != "" {
		effective.ToolRoleStrategy = cfg.ToolRoleStrategy
	}
	if cfg.SystemMessageStrategy != "" {
		effective.SystemMessageStrategy = cfg.SystemMessageStrategy
	}

	return effective, true, nil
}
