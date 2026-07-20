package task

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/bestruirui/octopus/internal/utils/xstrings"
)

// ChannelHealthProbeTask 主动探活所有已启用渠道。
// 所有参数从 setting 读取，前端可热更新；默认偏保守，避免误杀。
//
// 探活方法（health_probe_method）：
//   - auto   : models → head → base 级联，兼容 OpenAI / sub2api / 只认模型名的中转
//   - models : 仅 GET /v1/models（或 /models）
//   - head   : 仅 HEAD/GET base_url（最轻量，不依赖 models 列表）
//   - chat   : POST 最小 chat/completions（最准，适合号池/中转）
//   - custom : 自定义路径（health_probe_path）
func ChannelHealthProbeTask() {
	enabled, err := op.SettingGetBool(model.SettingKeyHealthProbeEnabled)
	if err != nil {
		enabled = true
	}
	if !enabled {
		log.Debugf("channel health probe skipped: disabled")
		return
	}

	cfg := loadProbeConfig()
	log.Debugf("channel health probe task started method=%s timeout=%ds trip_on_fail=%v",
		cfg.Method, cfg.TimeoutSec, cfg.TripOnFail)
	start := time.Now()
	defer func() {
		log.Debugf("channel health probe task finished in %s", time.Since(start))
	}()

	// 总超时 = 单次超时 * 2 + 缓冲，避免拖死调度器
	totalTimeout := time.Duration(cfg.TimeoutSec*2+30) * time.Second
	if totalTimeout < 2*time.Minute {
		totalTimeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	channels, err := op.ChannelList(ctx)
	if err != nil {
		log.Errorf("health probe: list channels failed: %v", err)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for i := range channels {
		ch := channels[i]
		if !ch.Enabled {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(channel model.Channel) {
			defer wg.Done()
			defer func() { <-sem }()
			probeChannel(ctx, &channel, cfg)
		}(ch)
	}
	wg.Wait()
}

type probeConfig struct {
	Method     string
	Path       string
	TimeoutSec int
	TripOnFail bool
	FailThresh int
	DegradeMS  int
}

func loadProbeConfig() probeConfig {
	cfg := probeConfig{
		Method:     model.HealthProbeMethodAuto,
		TimeoutSec: 8,
		TripOnFail: false,
		FailThresh: 3,
		DegradeMS:  5000,
	}
	if v, err := op.SettingGetString(model.SettingKeyHealthProbeMethod); err == nil && v != "" {
		cfg.Method = strings.ToLower(strings.TrimSpace(v))
	}
	if v, err := op.SettingGetString(model.SettingKeyHealthProbePath); err == nil {
		cfg.Path = strings.TrimSpace(v)
	}
	if v, err := op.SettingGetInt(model.SettingKeyHealthProbeTimeout); err == nil && v > 0 {
		cfg.TimeoutSec = v
	}
	if v, err := op.SettingGetBool(model.SettingKeyHealthProbeTripOnFail); err == nil {
		cfg.TripOnFail = v
	}
	if v, err := op.SettingGetInt(model.SettingKeyHealthProbeFailThreshold); err == nil && v > 0 {
		cfg.FailThresh = v
	}
	if v, err := op.SettingGetInt(model.SettingKeyHealthProbeDegradeMS); err == nil && v > 0 {
		cfg.DegradeMS = v
	}
	// 同步阈值到健康状态判定
	balancer.SetHealthThresholds(cfg.FailThresh, cfg.DegradeMS)
	return cfg
}

// pickProbeModel 从渠道已配置的 model/custom_model 列表取探活模型，不再用全局 setting。
// 优先 CustomModel（用户手动指定），其次 Model（同步列表），都没有则空串（chat 探活跳过）。
func pickProbeModel(channel *model.Channel) string {
	if channel == nil {
		return ""
	}
	names := xstrings.SplitTrimCompact(",", channel.CustomModel, channel.Model)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func probeChannel(ctx context.Context, channel *model.Channel, cfg probeConfig) {
	baseURL := channel.GetBaseUrl()
	if baseURL == "" {
		balancer.UpdateProbeResult(channel.ID, channel.Name, "", false, 0, "no base url")
		return
	}

	httpClient, err := helper.ChannelHttpClient(channel)
	if err != nil {
		balancer.UpdateProbeResult(channel.ID, channel.Name, baseURL, false, 0, err.Error())
		if cfg.TripOnFail {
			balancer.ForceTrip(channel.ID, err.Error())
		}
		return
	}

	client := *httpClient
	client.Timeout = time.Duration(cfg.TimeoutSec) * time.Second

	// 取一个可用 key（chat 探活需要鉴权；其他方式可选）
	apiKey := ""
	if k := channel.GetChannelKey(); k.ChannelKey != "" {
		apiKey = k.ChannelKey
	}

	// 探活模型：渠道自己的 model 列表，不再读全局 health_probe_model
	probeModel := pickProbeModel(channel)

	delay, probeErr := doProbe(ctx, &client, baseURL, apiKey, string(channel.Type), cfg, probeModel)
	ok := probeErr == nil

	errMsg := ""
	if probeErr != nil {
		errMsg = probeErr.Error()
	}

	balancer.UpdateProbeResult(channel.ID, channel.Name, baseURL, ok, delay, errMsg)

	if ok {
		balancer.ForceRecover(channel.ID)
		if delay > 0 {
			updateBaseURLDelay(channel, baseURL, delay)
		}
		log.Debugf("health probe ok: channel=%s delay=%dms method=%s model=%s", channel.Name, delay, cfg.Method, probeModel)
	} else {
		// 默认只记录失败，不强制熔断；连续失败达到阈值后由 buildHealth 判 unhealthy 跳过
		// 仅当用户显式开启 trip_on_fail 时才 ForceTrip
		if cfg.TripOnFail {
			// 再检查连续失败次数，避免一次抖动就熔
			health := balancer.GetChannelHealth(channel.ID)
			if health.FailStreak >= cfg.FailThresh {
				balancer.ForceTrip(channel.ID, errMsg)
			}
		}
		log.Warnf("health probe fail: channel=%s err=%v streak_thresh=%d", channel.Name, probeErr, cfg.FailThresh)
	}
}

func doProbe(ctx context.Context, client *http.Client, baseURL, apiKey, channelType string, cfg probeConfig, probeModel string) (int, error) {
	base := strings.TrimRight(baseURL, "/")
	method := cfg.Method

	switch method {
	case model.HealthProbeMethodModels:
		return probeModels(ctx, client, base, apiKey)
	case model.HealthProbeMethodHead:
		return probeHead(ctx, client, base, apiKey)
	case model.HealthProbeMethodChat:
		if probeModel == "" {
			return 0, fmt.Errorf("chat probe: channel has no model/custom_model configured")
		}
		return probeChat(ctx, client, base, apiKey, channelType, probeModel)
	case model.HealthProbeMethodCustom:
		path := cfg.Path
		if path == "" {
			return 0, fmt.Errorf("custom probe path is empty")
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		return probeGET(ctx, client, base+path, apiKey)
	default: // auto
		// 级联策略（参考 NewAPI / One API 社区实践 + sub2api 兼容）：
		// 1) models 列表（标准兼容）
		// 2) HEAD/GET base（中转不返回 models 也能通）
		// 3) 若渠道配置了模型名且有 key，再试一次 chat（最准但最贵，仅作最后手段）
		if delay, err := probeModels(ctx, client, base, apiKey); err == nil {
			return delay, nil
		}
		if delay, err := probeHead(ctx, client, base, apiKey); err == nil {
			return delay, nil
		}
		// auto 下 chat 作为最后手段：用渠道自己的模型列表
		if apiKey != "" && probeModel != "" {
			if delay, err := probeChat(ctx, client, base, apiKey, channelType, probeModel); err == nil {
				return delay, nil
			}
		}
		return 0, fmt.Errorf("auto probe: models/head/chat all failed")
	}
}

func probeModels(ctx context.Context, client *http.Client, base, apiKey string) (int, error) {
	candidates := []string{
		base + "/v1/models",
		base + "/models",
		// 有些部署把 base 已含 /v1
		base + "/v1/model/list",
	}
	var lastErr error
	for _, u := range candidates {
		delay, err := probeGET(ctx, client, u, apiKey)
		if err == nil {
			return delay, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("models probe failed")
	}
	return 0, lastErr
}

func probeHead(ctx context.Context, client *http.Client, base, apiKey string) (int, error) {
	// 先 HEAD，不支持再 GET
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, base, nil)
	if err != nil {
		return 0, err
	}
	setAuthHeaders(req, apiKey)
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		delay := int(time.Since(start).Milliseconds())
		if isReachableStatus(resp.StatusCode) {
			return delay, nil
		}
		// HEAD 返回 405/501 时回退 GET
		if resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotImplemented {
			return 0, fmt.Errorf("head status %d", resp.StatusCode)
		}
	}
	return probeGET(ctx, client, base, apiKey)
}

func probeGET(ctx context.Context, client *http.Client, url, apiKey string) (int, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	setAuthHeaders(req, apiKey)
	resp, err := client.Do(req)
	delay := int(time.Since(start).Milliseconds())
	if err != nil {
		return 0, err
	}
	// 读一点点 body 再关，避免连接复用不干净
	_, _ = io.CopyN(io.Discard, resp.Body, 512)
	resp.Body.Close()
	// 2xx/3xx/401/403/404 都算「服务可达」
	// 401/403：鉴权失败也说明 upstream 活着
	// 404：路径不对但服务在（很多中转不实现 /models）
	if isReachableStatus(resp.StatusCode) {
		return delay, nil
	}
	return 0, fmt.Errorf("status %d from %s", resp.StatusCode, url)
}

// probeChat 发一个 max_tokens=1 的最小请求，兼容：
// - 标准 OpenAI /v1/chat/completions
// - base 已含 /v1 的部署
// - Anthropic /v1/messages（简化探测）
// sub2api / 只认模型名的中转：不依赖 models 列表，直接打 chat 即可
func probeChat(ctx context.Context, client *http.Client, base, apiKey, channelType, modelName string) (int, error) {
	if apiKey == "" {
		return 0, fmt.Errorf("chat probe requires api key")
	}
	if modelName == "" {
		return 0, fmt.Errorf("chat probe requires a model from channel model/custom_model")
	}

	// 根据渠道类型选路径和 body
	var url string
	var body []byte
	var contentType = "application/json"

	ct := strings.ToLower(channelType)
	switch {
	case strings.Contains(ct, "anthropic"):
		url = joinPath(base, "/v1/messages")
		body = []byte(fmt.Sprintf(
			`{"model":%q,"max_tokens":1,"messages":[{"role":"user","content":"ping"}]}`,
			modelName,
		))
	case strings.Contains(ct, "gemini"):
		// Gemini generateContent 探测成本较高，退化为 models
		return probeModels(ctx, client, base, apiKey)
	default:
		// OpenAI 兼容（含 sub2api / one-api / new-api 中转）
		url = joinPath(base, "/v1/chat/completions")
		// 若 base 已以 /v1 结尾，避免 /v1/v1
		if strings.HasSuffix(strings.TrimRight(base, "/"), "/v1") {
			url = joinPath(base, "/chat/completions")
		}
		body = []byte(fmt.Sprintf(
			`{"model":%q,"max_tokens":1,"stream":false,"messages":[{"role":"user","content":"ping"}]}`,
			modelName,
		))
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", contentType)
	setAuthHeaders(req, apiKey)
	if strings.Contains(ct, "anthropic") {
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("x-api-key", apiKey)
	}

	resp, err := client.Do(req)
	delay := int(time.Since(start).Milliseconds())
	if err != nil {
		return 0, err
	}
	_, _ = io.CopyN(io.Discard, resp.Body, 1024)
	resp.Body.Close()

	// chat 探活：200 最佳；400/422 可能是模型名不对但服务活着；401/403 服务活着
	// 5xx / 429 视作失败（过载也算不健康，但 429 在号池场景很常见——仍记为可达但 degraded 由延迟阈值处理）
	if resp.StatusCode < 500 {
		// 429 也算可达（限流说明 upstream 在）
		return delay, nil
	}
	return 0, fmt.Errorf("chat probe status %d from %s", resp.StatusCode, url)
}

func setAuthHeaders(req *http.Request, apiKey string) {
	if apiKey == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
}

func isReachableStatus(code int) bool {
	// 服务进程活着即可：2xx/3xx/4xx（含 401/403/404/405/429）
	// 仅 5xx 和 0 算挂
	return code > 0 && code < 500
}

func joinPath(base, path string) string {
	b := strings.TrimRight(base, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	// 避免 /v1 + /v1/xxx
	if strings.HasSuffix(b, "/v1") && strings.HasPrefix(path, "/v1/") {
		path = strings.TrimPrefix(path, "/v1")
	}
	return b + path
}

func updateBaseURLDelay(channel *model.Channel, baseURL string, delay int) {
	if channel == nil || len(channel.BaseUrls) == 0 {
		return
	}
	urls := make([]model.BaseUrl, len(channel.BaseUrls))
	copy(urls, channel.BaseUrls)
	changed := false
	for i := range urls {
		if urls[i].URL == baseURL {
			if urls[i].Delay != delay {
				urls[i].Delay = delay
				changed = true
			}
			break
		}
	}
	if !changed {
		return
	}
	if err := op.ChannelBaseUrlUpdate(channel.ID, urls); err != nil {
		log.Debugf("health probe: update delay for channel %d failed: %v", channel.ID, err)
	}
}
