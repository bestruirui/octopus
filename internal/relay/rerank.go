package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	dbmodel "github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/llm/transformer"
)

func RerankHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, "failed to read request body")
			return
		}

		var req struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.Model == "" {
			resp.Error(c, http.StatusBadRequest, "missing or invalid model field")
			return
		}

		group, err := op.GroupGetEnabledMap(req.Model, c.Request.Context())
		if err != nil {
			resp.Error(c, http.StatusNotFound, "model not found")
			return
		}

		apiKeyID := c.GetInt("api_key_id")
		iter := balancer.NewIterator(group, apiKeyID, req.Model)
		if iter.Len() == 0 {
			resp.Error(c, http.StatusServiceUnavailable, "no available channel")
			return
		}
		startTime := time.Now()
		ctx := c.Request.Context()
		var lastErr error

		for iter.Next() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			item := iter.Item()
			channel, err := op.ChannelGet(item.ChannelID, ctx)
			if err != nil {
				iter.Skip(item.ChannelID, 0, fmt.Sprintf("channel_%d", item.ChannelID), fmt.Sprintf("channel not found: %v", err))
				lastErr = err
				continue
			}
			if !channel.Enabled {
				iter.Skip(channel.ID, 0, channel.Name, "channel disabled")
				continue
			}

			usedKey := channel.GetChannelKey()
			if usedKey.ChannelKey == "" {
				iter.Skip(channel.ID, 0, channel.Name, "no available key")
				continue
			}
			if iter.SkipCircuitBreak(channel.ID, usedKey.ID, channel.Name) {
				continue
			}

			span := iter.StartAttempt(channel.ID, usedKey.ID, channel.Name)

			statusCode, fwdErr := rerankForward(ctx, c, channel, usedKey, item.ModelName, req.Model, body)
			if fwdErr == nil {
				usedKey.StatusCode = statusCode
				usedKey.LastUseTimeStamp = time.Now().Unix()
				op.ChannelKeyUpdate(usedKey)
				span.End(dbmodel.AttemptSuccess, "")
				op.StatsChannelUpdate(channel.ID, dbmodel.StatsMetrics{
					WaitTime:       span.Duration().Milliseconds(),
					RequestSuccess: 1,
				})
				balancer.RecordSuccess(channel.ID, usedKey.ID, req.Model)
				saveRerankMetrics(ctx, apiKeyID, req.Model, item.ModelName, startTime, true, nil, iter.Attempts(), channel.ID)
				return
			}

			usedKey.StatusCode = statusCode
			usedKey.LastUseTimeStamp = time.Now().Unix()
			op.ChannelKeyUpdate(usedKey)
			span.End(dbmodel.AttemptFailed, fwdErr.Error())
			op.StatsChannelUpdate(channel.ID, dbmodel.StatsMetrics{
				WaitTime:      span.Duration().Milliseconds(),
				RequestFailed: 1,
			})
			balancer.RecordFailure(channel.ID, usedKey.ID, req.Model)
			lastErr = fmt.Errorf("channel %s failed: %v", channel.Name, fwdErr)
		}

		if lastErr == nil {
			lastErr = errors.New("all channels failed")
		}
		saveRerankMetrics(ctx, apiKeyID, req.Model, req.Model, startTime, false, lastErr, iter.Attempts(), 0)
		resp.Error(c, http.StatusBadGateway, lastErr.Error())
	}
}

func rerankForward(ctx context.Context, c *gin.Context, channel *dbmodel.Channel, usedKey dbmodel.ChannelKey, upstreamModel, requestModel string, body []byte) (int, error) {
	httpClient, err := helper.ChannelHttpClient(channel)
	if err != nil {
		return 0, err
	}

	outBody := body
	if upstreamModel != requestModel {
		var bodyMap map[string]any
		if err := json.Unmarshal(body, &bodyMap); err == nil {
			bodyMap["model"] = upstreamModel
			if modified, err := json.Marshal(bodyMap); err == nil {
				outBody = modified
			}
		}
	}

	baseURL := transformer.NormalizeBaseURL(channel.GetBaseUrl(), "v1")
	upstreamURL := baseURL + "/rerank"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(outBody))
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+usedKey.ChannelKey)
	for _, h := range channel.CustomHeader {
		if h.HeaderKey != "" {
			httpReq.Header.Set(h.HeaderKey, h.HeaderValue)
		}
	}

	upstream, err := httpClient.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer upstream.Body.Close()

	if upstream.StatusCode >= 400 {
		respBody, _ := io.ReadAll(upstream.Body)
		return upstream.StatusCode, fmt.Errorf("upstream returned %d: %s", upstream.StatusCode, string(respBody))
	}

	for key, values := range upstream.Header {
		for _, v := range values {
			c.Header(key, v)
		}
	}
	c.Status(upstream.StatusCode)
	_, _ = io.Copy(c.Writer, upstream.Body)
	return upstream.StatusCode, nil
}

func saveRerankMetrics(ctx context.Context, apiKeyID int, requestModel, actualModel string, startTime time.Time, success bool, err error, attempts []dbmodel.ChannelAttempt, channelID int) {
	duration := time.Since(startTime)
	globalStats := dbmodel.StatsMetrics{
		WaitTime: duration.Milliseconds(),
	}
	if success {
		globalStats.RequestSuccess = 1
	} else {
		globalStats.RequestFailed = 1
	}

	op.StatsTotalUpdate(globalStats)
	op.StatsHourlyUpdate(globalStats)
	op.StatsDailyUpdate(context.Background(), globalStats)
	op.StatsAPIKeyUpdate(apiKeyID, globalStats)

	log.Infof("rerank relay complete: model=%s, success=%t, duration=%dms, attempts=%d",
		requestModel, success, duration.Milliseconds(), len(attempts))

	relayLog := dbmodel.RelayLog{
		Time:             startTime.Unix(),
		RequestModelName: requestModel,
		ActualModelName:  actualModel,
		UseTime:          int(duration.Milliseconds()),
		Attempts:         attempts,
		TotalAttempts:    len(attempts),
	}
	if apiKey, getErr := op.APIKeyGet(apiKeyID, ctx); getErr == nil {
		relayLog.RequestAPIKeyName = apiKey.Name
	}
	if channelID > 0 {
		if ch, chErr := op.ChannelGet(channelID, ctx); chErr == nil {
			relayLog.ChannelName = ch.Name
			relayLog.ChannelId = channelID
		}
	}
	if err != nil {
		relayLog.Error = err.Error()
	}
	if logErr := op.RelayLogAdd(context.WithoutCancel(ctx), relayLog); logErr != nil {
		log.Warnf("failed to save rerank relay log: %v", logErr)
	}
}
