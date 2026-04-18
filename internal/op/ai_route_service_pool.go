package op

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lingyuins/octopus/internal/model"
)

const (
	defaultAIRouteParallelism = 3
)

type aiRouteService struct {
	Index   int
	Name    string
	BaseURL string
	APIKey  string
	Model   string
}

type aiRouteServiceHint struct {
	PromptEndpointType string
	BucketIndex        int
}

type aiRouteServiceLease struct {
	Index   int
	Service aiRouteService
}

type aiRouteServiceOutcome struct {
	Success   bool
	Retryable bool
	Cooldown  time.Duration
	Err       error
}

type aiRouteServicePool interface {
	Acquire(ctx context.Context, hint aiRouteServiceHint, exclude map[int]struct{}) (*aiRouteServiceLease, error)
	Next(ctx context.Context, prev *aiRouteServiceLease, hint aiRouteServiceHint, exclude map[int]struct{}, cause error) (*aiRouteServiceLease, error)
	Release(lease *aiRouteServiceLease, outcome aiRouteServiceOutcome)
}

type aiRouteServiceState struct {
	service       aiRouteService
	inflight      int
	cooldownUntil time.Time
}

type roundRobinAIRouteServicePool struct {
	mu       sync.Mutex
	cursor   int
	services []aiRouteServiceState
}

func newAIRouteServicePool(services []aiRouteService) aiRouteServicePool {
	states := make([]aiRouteServiceState, len(services))
	for i := range services {
		states[i] = aiRouteServiceState{service: services[i]}
	}
	return &roundRobinAIRouteServicePool{services: states}
}

func (p *roundRobinAIRouteServicePool) Acquire(
	ctx context.Context,
	hint aiRouteServiceHint,
	exclude map[int]struct{},
) (*aiRouteServiceLease, error) {
	for {
		now := time.Now()

		p.mu.Lock()
		idx, waitUntil, ok := p.pickLocked(now, exclude)
		if ok && idx >= 0 {
			p.services[idx].inflight++
			p.cursor = (idx + 1) % len(p.services)
			lease := &aiRouteServiceLease{
				Index:   idx,
				Service: p.services[idx].service,
			}
			p.mu.Unlock()
			return lease, nil
		}
		p.mu.Unlock()

		if !ok {
			return nil, fmt.Errorf("没有可用的 AI 路由分析服务（batch %d）", hint.BucketIndex)
		}

		wait := time.Until(waitUntil)
		if wait <= 0 {
			continue
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (p *roundRobinAIRouteServicePool) Next(
	ctx context.Context,
	_ *aiRouteServiceLease,
	hint aiRouteServiceHint,
	exclude map[int]struct{},
	_ error,
) (*aiRouteServiceLease, error) {
	return p.Acquire(ctx, hint, exclude)
}

func (p *roundRobinAIRouteServicePool) Release(lease *aiRouteServiceLease, outcome aiRouteServiceOutcome) {
	if p == nil || lease == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if lease.Index < 0 || lease.Index >= len(p.services) {
		return
	}

	state := &p.services[lease.Index]
	if state.inflight > 0 {
		state.inflight--
	}

	if !outcome.Success && outcome.Retryable && outcome.Cooldown > 0 {
		nextCooldown := time.Now().Add(outcome.Cooldown)
		if nextCooldown.After(state.cooldownUntil) {
			state.cooldownUntil = nextCooldown
		}
	}
}

func (p *roundRobinAIRouteServicePool) pickLocked(
	now time.Time,
	exclude map[int]struct{},
) (idx int, waitUntil time.Time, ok bool) {
	if len(p.services) == 0 {
		return -1, time.Time{}, false
	}

	bestIdx := -1
	bestInflight := 0
	hasEligible := false
	earliestCooldown := time.Time{}

	for offset := 0; offset < len(p.services); offset++ {
		idx = (p.cursor + offset) % len(p.services)
		if exclude != nil {
			if _, skipped := exclude[idx]; skipped {
				continue
			}
		}

		hasEligible = true
		state := p.services[idx]
		if state.cooldownUntil.After(now) {
			if earliestCooldown.IsZero() || state.cooldownUntil.Before(earliestCooldown) {
				earliestCooldown = state.cooldownUntil
			}
			continue
		}

		if bestIdx < 0 || state.inflight < bestInflight {
			bestIdx = idx
			bestInflight = state.inflight
		}
	}

	if bestIdx >= 0 {
		return bestIdx, time.Time{}, true
	}
	if hasEligible {
		return -1, earliestCooldown, true
	}
	return -1, time.Time{}, false
}

func loadAIRouteParallelism() int {
	value, err := SettingGetInt(model.SettingKeyAIRouteParallelism)
	if err != nil || value < 1 {
		return defaultAIRouteParallelism
	}
	return value
}

func clampAIRouteParallelism(value int, bucketCount int) int {
	if bucketCount <= 0 {
		return 1
	}
	if value < 1 {
		value = defaultAIRouteParallelism
	}
	if value > bucketCount {
		return bucketCount
	}
	return value
}

func loadAIRouteServicesSnapshot() ([]aiRouteService, error) {
	configs, err := loadAIRouteServiceConfigs()
	if err != nil {
		return nil, err
	}

	services := make([]aiRouteService, 0, len(configs))
	for i, cfg := range configs {
		if !cfg.IsEnabled() {
			continue
		}

		services = append(services, aiRouteService{
			Index:   i,
			Name:    normalizeAIRouteServiceName(cfg, i+1),
			BaseURL: strings.TrimSpace(cfg.BaseURL),
			APIKey:  strings.TrimSpace(cfg.APIKey),
			Model:   strings.TrimSpace(cfg.Model),
		})
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("没有可用的 AI 路由分析服务")
	}
	return services, nil
}

func loadAIRouteServiceConfigs() ([]model.AIRouteServiceConfig, error) {
	rawServices, _ := SettingGetString(model.SettingKeyAIRouteServices)
	rawServices = strings.TrimSpace(rawServices)
	if rawServices != "" && rawServices != "[]" {
		var services []model.AIRouteServiceConfig
		if err := json.Unmarshal([]byte(rawServices), &services); err != nil {
			return nil, fmt.Errorf("AI路由服务池配置不完整")
		}
		return services, nil
	}

	baseURL, err := SettingGetString(model.SettingKeyAIRouteBaseURL)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("AI路由模型配置不完整")
	}

	apiKey, err := SettingGetString(model.SettingKeyAIRouteAPIKey)
	if err != nil || strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("AI路由模型配置不完整")
	}

	modelName, err := SettingGetString(model.SettingKeyAIRouteModel)
	if err != nil || strings.TrimSpace(modelName) == "" {
		return nil, fmt.Errorf("AI路由模型配置不完整")
	}

	enabled := true
	return []model.AIRouteServiceConfig{{
		Name:    "legacy",
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   modelName,
		Enabled: &enabled,
	}}, nil
}

func normalizeAIRouteServiceName(cfg model.AIRouteServiceConfig, ordinal int) string {
	if name := strings.TrimSpace(cfg.Name); name != "" {
		return name
	}

	if parsed, err := url.Parse(strings.TrimSpace(cfg.BaseURL)); err == nil && parsed.Host != "" {
		if modelName := strings.TrimSpace(cfg.Model); modelName != "" {
			return fmt.Sprintf("%s (%s)", parsed.Host, modelName)
		}
		return parsed.Host
	}

	return fmt.Sprintf("service-%d", ordinal)
}
