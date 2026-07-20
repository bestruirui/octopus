package task

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/utils/log"
)

// ChannelHealthProbeTask 主动探活所有已启用渠道：
// 1. 对每个 base_url 做轻量 HTTP 探测（优先 /v1/models，失败回退 HEAD/GET base）
// 2. 写入健康状态供看板展示
// 3. 连续失败触发熔断；恢复成功则解除探活熔断
func ChannelHealthProbeTask() {
	log.Debugf("channel health probe task started")
	start := time.Now()
	defer func() {
		log.Debugf("channel health probe task finished in %s", time.Since(start))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	channels, err := op.ChannelList(ctx)
	if err != nil {
		log.Errorf("health probe: list channels failed: %v", err)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 8) // 最多 8 并发

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
			probeChannel(ctx, &channel)
		}(ch)
	}
	wg.Wait()
}

func probeChannel(ctx context.Context, channel *model.Channel) {
	baseURL := channel.GetBaseUrl()
	if baseURL == "" {
		balancer.UpdateProbeResult(channel.ID, channel.Name, "", false, 0, "no base url")
		return
	}

	httpClient, err := helper.ChannelHttpClient(channel)
	if err != nil {
		balancer.UpdateProbeResult(channel.ID, channel.Name, baseURL, false, 0, err.Error())
		balancer.ForceTrip(channel.ID, err.Error())
		return
	}

	// 缩短单次超时，避免拖死整个任务
	client := *httpClient
	client.Timeout = 8 * time.Second

	delay, probeErr := doProbe(ctx, &client, baseURL)
	ok := probeErr == nil

	errMsg := ""
	if probeErr != nil {
		errMsg = probeErr.Error()
	}

	balancer.UpdateProbeResult(channel.ID, channel.Name, baseURL, ok, delay, errMsg)

	if ok {
		balancer.ForceRecover(channel.ID)
		// 同步更新 base_url delay，供选路使用
		if delay > 0 {
			updateBaseURLDelay(channel, baseURL, delay)
		}
		log.Debugf("health probe ok: channel=%s delay=%dms", channel.Name, delay)
	} else {
		balancer.ForceTrip(channel.ID, errMsg)
		log.Warnf("health probe fail: channel=%s err=%v", channel.Name, probeErr)
	}
}

func doProbe(ctx context.Context, client *http.Client, baseURL string) (int, error) {
	base := strings.TrimRight(baseURL, "/")
	candidates := []string{
		base + "/v1/models",
		base + "/models",
		base,
	}

	var lastErr error
	for _, url := range candidates {
		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := client.Do(req)
		delay := int(time.Since(start).Milliseconds())
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		// 2xx / 3xx / 401 / 403 都算「服务可达」（鉴权失败也说明 upstream 活着）
		if resp.StatusCode < 500 {
			return delay, nil
		}
		lastErr = fmt.Errorf("status %d from %s", resp.StatusCode, url)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("all probe endpoints failed")
	}
	return 0, lastErr
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
	// 仅更新内存缓存中的 delay，供选路使用；完整落库由定时 base_url_delay 任务负责
	if err := op.ChannelBaseUrlUpdate(channel.ID, urls); err != nil {
		log.Debugf("health probe: update delay for channel %d failed: %v", channel.ID, err)
	}
}
