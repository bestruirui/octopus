package breaker

import (
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
)

var defaultConfig = model.CircuitBreakerConfig{
	Enabled:          true,
	FailureThreshold: 3,
	BaseCooldownMS:   180_000,
	MaxCooldownMS:    3_600_000,
	BackoffFactor:    2,
	JitterMin:        0.5,
	JitterMax:        1.5,
	DecayWindowMS:    21_600_000, // 6h
}

func loadGlobalConfig() model.CircuitBreakerConfig {
	cfg := defaultConfig

	if v, err := op.SettingGetBool(model.SettingKeyCBEnabled); err == nil {
		cfg.Enabled = v
	}
	if v, err := op.SettingGetInt(model.SettingKeyCBFailureThreshold); err == nil {
		cfg.FailureThreshold = v
	}
	if v, err := op.SettingGetInt(model.SettingKeyCBBaseCooldownMS); err == nil {
		cfg.BaseCooldownMS = v
	}
	if v, err := op.SettingGetInt(model.SettingKeyCBMaxCooldownMS); err == nil {
		cfg.MaxCooldownMS = v
	}
	if v, err := op.SettingGetFloat(model.SettingKeyCBBackoffFactor); err == nil {
		cfg.BackoffFactor = v
	}
	return sanitizeConfig(cfg)
}

func ResolveEffectiveConfig() model.CircuitBreakerConfig {
	return loadGlobalConfig()
}

func sanitizeConfig(cfg model.CircuitBreakerConfig) model.CircuitBreakerConfig {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = defaultConfig.FailureThreshold
	}
	if cfg.BaseCooldownMS <= 0 {
		cfg.BaseCooldownMS = defaultConfig.BaseCooldownMS
	}
	if cfg.MaxCooldownMS <= 0 {
		cfg.MaxCooldownMS = defaultConfig.MaxCooldownMS
	}
	if cfg.MaxCooldownMS < cfg.BaseCooldownMS {
		cfg.MaxCooldownMS = cfg.BaseCooldownMS
	}
	if cfg.BackoffFactor < 1 {
		cfg.BackoffFactor = defaultConfig.BackoffFactor
	}
	if cfg.JitterMin <= 0 {
		cfg.JitterMin = defaultConfig.JitterMin
	}
	if cfg.JitterMax < cfg.JitterMin {
		cfg.JitterMax = cfg.JitterMin
	}
	if cfg.DecayWindowMS <= 0 {
		cfg.DecayWindowMS = defaultConfig.DecayWindowMS
	}
	return cfg
}
