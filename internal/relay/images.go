package relay

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/price"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/relay/bodycache"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/transformer/outbound"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/gin-gonic/gin"
)

const imagesUpstreamErrorBodyLimit = 16 * 1024

// ImagesHandler 是 OpenAI Images API 的统一 relay 入口。
// endpoint 形如：/images/generations、/images/edits、/images/variations（不含 /v1 前缀）。
func ImagesHandler(endpoint string, c *gin.Context) {
	ctx := c.Request.Context()

	apiKeyID := c.GetInt("api_key_id")

	// 缓存请求体，支持多次重试重放
	bc, err := bodycache.New(c.Request.Body)
	if err != nil {
		var tooLarge *bodycache.BodyTooLargeError
		if errors.As(err, &tooLarge) {
			resp.Error(c, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() {
		if cerr := bc.Close(); cerr != nil {
			log.Warnf("failed to close images body cache: %v", cerr)
		}
	}()

	contentType := c.GetHeader("Content-Type")
	isMultipart := strings.Contains(strings.ToLower(contentType), "multipart/form-data")

	// 解析 requestModel 与 stream（严格模式：model 必填）
	var (
		requestModel string
		stream       bool
		boundary     string
		jsonPayload  map[string]any
	)
	if isMultipart {
		_, params, perr := mime.ParseMediaType(contentType)
		if perr != nil {
			resp.Error(c, http.StatusBadRequest, "invalid multipart content-type")
			return
		}
		boundary = strings.TrimSpace(params["boundary"])
		if boundary == "" {
			resp.Error(c, http.StatusBadRequest, "invalid multipart boundary")
			return
		}
		m, s, perr := parseMultipartModelAndStream(bc, boundary)
		if perr != nil {
			resp.Error(c, http.StatusBadRequest, perr.Error())
			return
		}
		requestModel = m
		stream = s
	} else {
		payload, m, s, perr := parseJSONModelAndStream(bc)
		if perr != nil {
			resp.Error(c, http.StatusBadRequest, perr.Error())
			return
		}
		jsonPayload = payload
		requestModel = m
		stream = s
	}

	// supported_models 校验（复用 APIKeyAuth 注入）
	supportedModels := strings.TrimSpace(c.GetString("supported_models"))
	if supportedModels != "" {
		supportedModelsArray := strings.Split(supportedModels, ",")
		if !slices.Contains(supportedModelsArray, requestModel) {
			resp.Error(c, http.StatusBadRequest, "model not supported")
			return
		}
	}

	// 获取通道分组
	group, err := op.GroupGetEnabledMap(requestModel, ctx)
	if err != nil {
		resp.Error(c, http.StatusNotFound, "model not found")
		return
	}

	// 创建迭代器（策略排序 + 粘性优先）
	iter := balancer.NewIterator(group, apiKeyID, requestModel)
	if iter.Len() == 0 {
		resp.Error(c, http.StatusServiceUnavailable, "no available channel")
		return
	}

	// 初始化 Metrics（Images 独立，避免 b64_json 内存膨胀）
	metrics := newImagesRelayMetrics(apiKeyID, requestModel)
	metrics.RequestContent = buildImagesRequestContentForLog(isMultipart, bc, jsonPayload)

	var lastErr error

	for iter.Next() {
		select {
		case <-ctx.Done():
			log.Infof("request context canceled, stopping retry")
			metrics.Save(ctx, false, context.Canceled, iter.Attempts())
			return
		default:
		}

		item := iter.Item()

		// 获取通道
		channel, err := op.ChannelGet(item.ChannelID, ctx)
		if err != nil {
			log.Warnf("failed to get channel %d: %v", item.ChannelID, err)
			iter.Skip(item.ChannelID, 0, fmt.Sprintf("channel_%d", item.ChannelID), fmt.Sprintf("channel not found: %v", err))
			lastErr = err
			continue
		}
		if !channel.Enabled {
			iter.Skip(channel.ID, 0, channel.Name, "channel disabled")
			continue
		}

		// channel.Type 限制：仅 OpenAI Chat/Responses
		if channel.Type != outbound.OutboundTypeOpenAIChat && channel.Type != outbound.OutboundTypeOpenAIResponse {
			iter.Skip(channel.ID, 0, channel.Name, fmt.Sprintf("unsupported channel type: %d", channel.Type))
			continue
		}

		usedKey := channel.GetChannelKey()
		if usedKey.ChannelKey == "" {
			iter.Skip(channel.ID, 0, channel.Name, "no available key")
			continue
		}

		// 熔断检查（熔断 key 使用 actualModel=item.ModelName）
		if iter.SkipCircuitBreak(channel.ID, usedKey.ID, channel.Name) {
			continue
		}

		log.Infof("images request model %s, mode: %d, forwarding to channel: %s model: %s (attempt %d/%d, sticky=%t, stream=%t)",
			requestModel, group.Mode, channel.Name, item.ModelName,
			iter.Index()+1, iter.Len(), iter.IsSticky(), stream)

		span := iter.StartAttempt(channel.ID, usedKey.ID, channel.Name)

		// 尝试一次转发
		statusCode, written, usage, upstreamCT, fwdErr := imagesAttempt(ctx, endpoint, c, bc, isMultipart, boundary, jsonPayload, stream, channel, usedKey.ChannelKey, group.FirstTokenTimeOut, metrics, item.ModelName)

		// 更新 channel key 状态
		usedKey.StatusCode = statusCode
		usedKey.LastUseTimeStamp = time.Now().Unix()

		if fwdErr == nil {
			// ====== 成功 ======
			metrics.ActualModel = item.ModelName
			if usage != nil {
				metrics.SetUsageFromImages(item.ModelName, *usage)
			}
			metrics.ResponseContent = buildImagesResponseContentForLog(stream, upstreamCT, usage)

			usedKey.TotalCost += metrics.Stats.InputCost + metrics.Stats.OutputCost
			op.ChannelKeyUpdate(usedKey)

			span.End(model.AttemptSuccess, statusCode, "")

			// Channel 维度统计
			op.StatsChannelUpdate(channel.ID, model.StatsMetrics{
				WaitTime:       span.Duration().Milliseconds(),
				RequestSuccess: 1,
			})

			// 熔断器：记录成功
			balancer.RecordSuccess(channel.ID, usedKey.ID, item.ModelName)
			// 会话保持：更新粘性记录
			balancer.SetSticky(apiKeyID, requestModel, channel.ID, usedKey.ID)

			metrics.Save(ctx, true, nil, iter.Attempts())
			return
		}

		// ====== 失败 ======
		op.ChannelKeyUpdate(usedKey)
		span.End(model.AttemptFailed, statusCode, fwdErr.Error())

		// Channel 维度统计
		op.StatsChannelUpdate(channel.ID, model.StatsMetrics{
			WaitTime:      span.Duration().Milliseconds(),
			RequestFailed: 1,
		})

		// 熔断器：记录失败
		balancer.RecordFailure(channel.ID, usedKey.ID, item.ModelName)

		if written || c.Writer.Written() {
			metrics.Save(ctx, false, fwdErr, iter.Attempts())
			return
		}

		lastErr = fmt.Errorf("channel %s failed: %v", channel.Name, fwdErr)
	}

	// 所有通道都失败
	metrics.Save(ctx, false, lastErr, iter.Attempts())
	resp.Error(c, http.StatusBadGateway, "all channels failed")
}

type imagesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type imagesRelayMetrics struct {
	APIKeyID     int
	RequestModel string
	ActualModel  string
	StartTime    time.Time
	FirstToken   time.Time

	Stats model.StatsMetrics

	RequestContent  string
	ResponseContent string
}

func newImagesRelayMetrics(apiKeyID int, requestModel string) *imagesRelayMetrics {
	return &imagesRelayMetrics{
		APIKeyID:     apiKeyID,
		RequestModel: requestModel,
		StartTime:    time.Now(),
	}
}

func (m *imagesRelayMetrics) SetFirstTokenTime(t time.Time) {
	if m.FirstToken.IsZero() {
		m.FirstToken = t
	}
}

func (m *imagesRelayMetrics) SetUsageFromImages(actualModel string, u imagesUsage) {
	m.ActualModel = actualModel
	m.Stats.InputToken = int64(u.InputTokens)
	m.Stats.OutputToken = int64(u.OutputTokens)

	modelPrice := price.GetLLMPrice(actualModel)
	if modelPrice == nil {
		return
	}

	m.Stats.InputCost = float64(u.InputTokens) * modelPrice.Input * 1e-6
	m.Stats.OutputCost = float64(u.OutputTokens) * modelPrice.Output * 1e-6
}

func (m *imagesRelayMetrics) Save(ctx context.Context, success bool, err error, attempts []model.ChannelAttempt) {
	duration := time.Since(m.StartTime)

	globalStats := model.StatsMetrics{
		WaitTime:    duration.Milliseconds(),
		InputToken:  m.Stats.InputToken,
		OutputToken: m.Stats.OutputToken,
		InputCost:   m.Stats.InputCost,
		OutputCost:  m.Stats.OutputCost,
	}
	if success {
		globalStats.RequestSuccess = 1
	} else {
		globalStats.RequestFailed = 1
	}

	channelID, channelName := finalChannel(attempts)
	op.StatsTotalUpdate(globalStats)
	op.StatsHourlyUpdate(globalStats)
	op.StatsDailyUpdate(context.Background(), globalStats)
	op.StatsAPIKeyUpdate(m.APIKeyID, globalStats)
	op.StatsChannelUpdate(channelID, globalStats)

	log.Infof("images relay complete: model=%s, channel=%d(%s), success=%t, duration=%dms, input_token=%d, output_token=%d, input_cost=%f, output_cost=%f, total_cost=%f, attempts=%d",
		m.RequestModel, channelID, channelName, success, duration.Milliseconds(),
		m.Stats.InputToken, m.Stats.OutputToken,
		m.Stats.InputCost, m.Stats.OutputCost, m.Stats.InputCost+m.Stats.OutputCost,
		len(attempts))

	m.saveLog(ctx, err, duration, attempts, channelID, channelName)
}

func (m *imagesRelayMetrics) saveLog(ctx context.Context, err error, duration time.Duration, attempts []model.ChannelAttempt, channelID int, channelName string) {
	actualModel := m.ActualModel
	if actualModel == "" {
		actualModel = m.RequestModel
	}

	relayLog := model.RelayLog{
		Time:             m.StartTime.Unix(),
		RequestModelName: m.RequestModel,
		ChannelName:      channelName,
		ChannelId:        channelID,
		ActualModelName:  actualModel,
		UseTime:          int(duration.Milliseconds()),
		Attempts:         attempts,
		TotalAttempts:    len(attempts),
		RequestContent:   m.RequestContent,
		ResponseContent:  m.ResponseContent,
	}

	if apiKey, getErr := op.APIKeyGet(m.APIKeyID, ctx); getErr == nil {
		relayLog.RequestAPIKeyName = apiKey.Name
	}

	// 首字时间
	if !m.FirstToken.IsZero() {
		relayLog.Ftut = int(m.FirstToken.Sub(m.StartTime).Milliseconds())
	}

	// Usage
	if m.Stats.InputToken > 0 || m.Stats.OutputToken > 0 {
		relayLog.InputTokens = int(m.Stats.InputToken)
		relayLog.OutputTokens = int(m.Stats.OutputToken)
		relayLog.Cost = m.Stats.InputCost + m.Stats.OutputCost
	}

	if err != nil {
		relayLog.Error = err.Error()
	}

	if logErr := op.RelayLogAdd(ctx, relayLog); logErr != nil {
		log.Warnf("failed to save relay log: %v", logErr)
	}
}

func buildImagesRequestContentForLog(isMultipart bool, bc *bodycache.BodyCache, jsonPayload map[string]any) string {
	if isMultipart {
		// multipart 可能包含图片文件，避免落库
		return fmt.Sprintf(`{"content_type":"multipart/form-data","size_bytes":%d,"note":"multipart request content omitted for storage"}`, bc.Size())
	}
	if jsonPayload == nil {
		return ""
	}
	b, err := json.Marshal(jsonPayload)
	if err != nil {
		return ""
	}
	return truncateString(string(b), 8*1024)
}

func buildImagesResponseContentForLog(stream bool, upstreamCT string, usage *imagesUsage) string {
	if usage == nil {
		return ""
	}
	// 不记录 b64_json，仅记录 usage
	type respForLog struct {
		Stream      bool        `json:"stream"`
		ContentType string      `json:"content_type,omitempty"`
		Usage       *imagesUsage `json:"usage,omitempty"`
		Note        string      `json:"note,omitempty"`
	}
	obj := respForLog{
		Stream:      stream,
		ContentType: upstreamCT,
		Usage:       usage,
		Note:        "image data omitted for storage",
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(b)
}

func truncateString(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

func parseJSONModelAndStream(bc *bodycache.BodyCache) (payload map[string]any, modelName string, stream bool, err error) {
	r, err := bc.NewReader()
	if err != nil {
		return nil, "", false, err
	}
	defer r.Close()

	body, err := io.ReadAll(r)
	if err != nil {
		return nil, "", false, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, "", false, errors.New("empty body")
	}

	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, "", false, errors.New("invalid json")
	}

	rawModel, ok := m["model"]
	if !ok {
		return nil, "", false, errors.New("model is required")
	}
	modelStr, ok := rawModel.(string)
	if !ok || strings.TrimSpace(modelStr) == "" {
		return nil, "", false, errors.New("model is required")
	}

	stream = false
	if v, ok := m["stream"]; ok {
		switch vv := v.(type) {
		case bool:
			stream = vv
		case string:
			stream = strings.EqualFold(strings.TrimSpace(vv), "true")
		case float64:
			stream = vv != 0
		}
	}

	return m, strings.TrimSpace(modelStr), stream, nil
}

func parseMultipartModelAndStream(bc *bodycache.BodyCache, boundary string) (modelName string, stream bool, err error) {
	r, err := bc.NewReader()
	if err != nil {
		return "", false, err
	}
	defer r.Close()

	mr := multipart.NewReader(r, boundary)

	stream = false
	for {
		part, err := mr.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", false, err
		}

		name := part.FormName()
		if name == "" {
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			continue
		}

		switch name {
		case "model":
			b, _ := io.ReadAll(io.LimitReader(part, 1024))
			modelName = strings.TrimSpace(string(b))
		case "stream":
			b, _ := io.ReadAll(io.LimitReader(part, 16))
			stream = strings.EqualFold(strings.TrimSpace(string(b)), "true")
		default:
			_, _ = io.Copy(io.Discard, part)
		}
		_ = part.Close()
	}

	if strings.TrimSpace(modelName) == "" {
		return "", false, errors.New("model is required")
	}
	return modelName, stream, nil
}

func imagesAttempt(
	ctx context.Context,
	endpoint string,
	c *gin.Context,
	bc *bodycache.BodyCache,
	isMultipart bool,
	boundary string,
	jsonPayload map[string]any,
	stream bool,
	channel *model.Channel,
	channelKey string,
	firstTokenTimeOutSec int,
	metrics *imagesRelayMetrics,
	actualModel string,
) (statusCode int, written bool, usage *imagesUsage, upstreamCT string, err error) {
	// 构建 URL（baseUrl.Path 后追加 endpoint）
	baseURL := channel.GetBaseUrl()
	parsedURL, err := url.Parse(strings.TrimSuffix(baseURL, "/"))
	if err != nil {
		return 0, false, nil, "", fmt.Errorf("failed to parse base url: %w", err)
	}
	parsedURL.Path = parsedURL.Path + endpoint

	var bodyReader io.Reader
	var contentType string

	if isMultipart {
		pr, pw := io.Pipe()
		mw := multipart.NewWriter(pw)
		contentType = mw.FormDataContentType()
		bodyReader = pr

		go func() {
			src, err := bc.NewReader()
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			defer src.Close()

			if err := copyMultipartReplaceModel(src, boundary, mw, actualModel); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			// 先关闭 multipart.Writer 写入结束 boundary，再关闭 pipe writer
			if err := mw.Close(); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			_ = pw.Close()
		}()
	} else {
		// JSON：仅改写 model 字段，其余保持不变
		// 注意：每次尝试都重新 marshal 生成 body，确保可重试重建
		if jsonPayload == nil {
			return 0, false, nil, "", errors.New("nil json payload")
		}
		jsonPayload["model"] = actualModel
		b, err := json.Marshal(jsonPayload)
		if err != nil {
			return 0, false, nil, "", fmt.Errorf("failed to marshal json: %w", err)
		}
		bodyReader = bytes.NewReader(b)
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "", bodyReader)
	if err != nil {
		return 0, false, nil, "", fmt.Errorf("failed to create request: %w", err)
	}
	req.URL = parsedURL
	req.Method = http.MethodPost

	// Header 透传：复制下游 header，过滤 hop-by-hop 与鉴权相关
	copyHeadersToUpstream(req, c, channel, channelKey, contentType, stream)

	// 发送请求
	httpClient, err := helper.ChannelHttpClient(channel)
	if err != nil {
		return 0, false, nil, "", err
	}

	respUp, err := httpClient.Do(req)
	if err != nil {
		return 0, false, nil, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer respUp.Body.Close()

	upstreamCT = respUp.Header.Get("Content-Type")

	// stream=true：逐行解析 event/data/空行边界透传
	if stream {
		if respUp.StatusCode < 200 || respUp.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(respUp.Body, imagesUpstreamErrorBodyLimit))
			return respUp.StatusCode, false, nil, upstreamCT, fmt.Errorf("upstream error: %d: %s", respUp.StatusCode, string(b))
		}
		u, w, err := proxySSE(ctx, c, respUp, firstTokenTimeOutSec, metrics)
		return respUp.StatusCode, w, u, upstreamCT, err
	}

	// 非流式：2xx 透传，否则读取限长错误体用于错误信息与重试判定
	if respUp.StatusCode < 200 || respUp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(respUp.Body, imagesUpstreamErrorBodyLimit))
		return respUp.StatusCode, false, nil, upstreamCT, fmt.Errorf("upstream error: %d: %s", respUp.StatusCode, string(b))
	}

	u, w, err := proxyNonStream(c, respUp)
	return respUp.StatusCode, w, u, upstreamCT, err
}

func copyHeadersToUpstream(req *http.Request, c *gin.Context, channel *model.Channel, channelKey string, contentType string, stream bool) {
	for k, values := range c.Request.Header {
		if hopByHopHeaders[strings.ToLower(k)] {
			continue
		}
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	} else if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+channelKey)

	if len(channel.CustomHeader) > 0 {
		for _, h := range channel.CustomHeader {
			req.Header.Set(h.HeaderKey, h.HeaderValue)
		}
	}
}

func copyMultipartReplaceModel(src io.Reader, boundary string, dst *multipart.Writer, newModel string) error {
	mr := multipart.NewReader(src, boundary)

	for {
		part, err := mr.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		hdr := make(textproto.MIMEHeader, len(part.Header))
		for k, vv := range part.Header {
			cp := make([]string, len(vv))
			copy(cp, vv)
			hdr[k] = cp
		}

		pw, err := dst.CreatePart(hdr)
		if err != nil {
			_ = part.Close()
			return err
		}

		if part.FormName() == "model" && part.FileName() == "" {
			// 丢弃原值，写入替换后的 model（继续复制后续 part）
			_, _ = io.Copy(io.Discard, part)
			_, werr := io.WriteString(pw, newModel)
			_ = part.Close()
			if werr != nil {
				return werr
			}
			continue
		}

		_, err = io.Copy(pw, part)
		_ = part.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// proxyNonStream 将上游非流式响应原样透传到下游，同时尽量提取 usage（避免解析巨大 b64_json）。
func proxyNonStream(c *gin.Context, respUp *http.Response) (*imagesUsage, bool, error) {
	ct := respUp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}
	c.Header("Content-Type", ct)
	c.Status(respUp.StatusCode)

	scanner := newUsageScanner()

	buf := make([]byte, 32*1024)
	for {
		n, rerr := respUp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			scanner.Feed(chunk)
			if _, werr := c.Writer.Write(chunk); werr != nil {
				return scanner.Usage(), true, werr
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			return scanner.Usage(), c.Writer.Written(), rerr
		}
	}

	return scanner.Usage(), c.Writer.Written(), nil
}

// proxySSE 将上游 SSE 逐行解析 event/data/空行并透传到下游；首事件计为 FirstTokenTime；支持 FirstTokenTimeOut 切换。
func proxySSE(ctx context.Context, c *gin.Context, respUp *http.Response, firstTokenTimeOutSec int, metrics *imagesRelayMetrics) (*imagesUsage, bool, error) {
	if ct := respUp.Header.Get("Content-Type"); ct != "" && !strings.Contains(strings.ToLower(ct), "text/event-stream") {
		b, _ := io.ReadAll(io.LimitReader(respUp.Body, imagesUpstreamErrorBodyLimit))
		return nil, false, fmt.Errorf("upstream returned non-SSE content-type %q for stream request: %s", ct, string(b))
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	type lineResult struct {
		line []byte
		err  error
		eof  bool
	}

	results := make(chan lineResult, 1)
	go func() {
		defer close(results)
		br := bufio.NewReaderSize(respUp.Body, 64*1024)
		for {
			line, err := readLineLimited(br, maxSSEEventSize)
			if err != nil {
				if errors.Is(err, io.EOF) {
					results <- lineResult{eof: true}
					return
				}
				results <- lineResult{err: err}
				return
			}
			results <- lineResult{line: line}
		}
	}()

	var firstTokenTimer *time.Timer
	var firstTokenC <-chan time.Time
	if firstTokenTimeOutSec > 0 {
		firstTokenTimer = time.NewTimer(time.Duration(firstTokenTimeOutSec) * time.Second)
		firstTokenC = firstTokenTimer.C
		defer func() {
			if firstTokenTimer != nil {
				firstTokenTimer.Stop()
			}
		}()
	}

	var (
		firstWrite      = true
		currentEvent    string
		completedScanner = newUsageScanner()
	)

	for {
		select {
		case <-ctx.Done():
			log.Infof("client disconnected, stopping stream")
			return completedScanner.Usage(), c.Writer.Written(), nil

		case <-firstTokenC:
			log.Warnf("first token timeout (%ds), switching channel", firstTokenTimeOutSec)
			_ = respUp.Body.Close()
			return completedScanner.Usage(), c.Writer.Written(), fmt.Errorf("first token timeout (%ds)", firstTokenTimeOutSec)

		case r, ok := <-results:
			if !ok {
				return completedScanner.Usage(), c.Writer.Written(), nil
			}
			if r.eof {
				return completedScanner.Usage(), c.Writer.Written(), nil
			}
			if r.err != nil {
				return completedScanner.Usage(), c.Writer.Written(), fmt.Errorf("failed to read stream line: %w", r.err)
			}

			line := r.line
			trimmed := bytes.TrimRight(line, "\r\n")
			if len(trimmed) == 0 {
				// 空行：事件边界
				currentEvent = ""
			} else if bytes.HasPrefix(trimmed, []byte("event:")) {
				currentEvent = strings.TrimSpace(string(trimmed[len("event:"):]))
			} else if bytes.HasPrefix(trimmed, []byte("data:")) {
				// 仅在 completed 事件上尝试提取 usage（避免解析/分配巨大 b64_json）
				payload := bytes.TrimSpace(trimmed[len("data:"):])
				if currentEvent == "image_generation.completed" || bytes.Contains(payload, []byte(`"type":"image_generation.completed"`)) {
					completedScanner.Feed(payload)
				}
			}

			if _, werr := c.Writer.Write(line); werr != nil {
				return completedScanner.Usage(), true, werr
			}
			c.Writer.Flush()

			if firstWrite {
				metrics.SetFirstTokenTime(time.Now())
				firstWrite = false
				if firstTokenTimer != nil {
					if !firstTokenTimer.Stop() {
						select {
						case <-firstTokenTimer.C:
						default:
						}
					}
					firstTokenTimer = nil
					firstTokenC = nil
				}
			}
		}
	}
}

func readLineLimited(br *bufio.Reader, limit int) ([]byte, error) {
	var out []byte
	for {
		part, err := br.ReadSlice('\n')
		out = append(out, part...)
		if len(out) > limit {
			return nil, fmt.Errorf("sse line exceeds limit %d bytes", limit)
		}
		if err == nil {
			return out, nil
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		// 允许返回已读部分 + err（调用方按 err 处理）
		return out, err
	}
}

type usageScanner struct {
	matchIdx       int
	waitForObject  bool
	collecting     bool
	braceDepth     int
	inString       bool
	escape         bool
	buf            bytes.Buffer
	usage          *imagesUsage
	done           bool
	maxCollectSize int
}

func newUsageScanner() *usageScanner {
	return &usageScanner{maxCollectSize: 64 * 1024}
}

// Feed 逐字节扫描输入，定位 "usage":{...} 并仅解析 usage 子对象。
// 该实现用于避免整体 json.Unmarshal 造成 b64_json 巨大内存分配。
func (s *usageScanner) Feed(p []byte) {
	if s.done || len(p) == 0 {
		return
	}
	const pat = `"usage":`

	for _, b := range p {
		if s.done {
			return
		}

		if s.collecting {
			if s.buf.Len() >= s.maxCollectSize {
				s.collecting = false
				s.done = true
				return
			}
			s.buf.WriteByte(b)

			if s.inString {
				if s.escape {
					s.escape = false
				} else if b == '\\' {
					s.escape = true
				} else if b == '"' {
					s.inString = false
				}
				continue
			}

			if b == '"' {
				s.inString = true
				continue
			}

			switch b {
			case '{':
				s.braceDepth++
			case '}':
				s.braceDepth--
				if s.braceDepth == 0 {
					var u imagesUsage
					if err := json.Unmarshal(s.buf.Bytes(), &u); err == nil {
						s.usage = &u
					}
					s.done = true
					s.collecting = false
					return
				}
			}
			continue
		}

		if s.waitForObject {
			if b == '{' {
				s.collecting = true
				s.braceDepth = 1
				s.buf.Reset()
				s.buf.WriteByte('{')
				s.inString = false
				s.escape = false
				s.waitForObject = false
				continue
			}
			// 跳过空白，遇到其他字符则放弃
			if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
				continue
			}
			s.waitForObject = false
			continue
		}

		// 匹配 "usage":
		if b == pat[s.matchIdx] {
			s.matchIdx++
			if s.matchIdx == len(pat) {
				s.waitForObject = true
				s.matchIdx = 0
			}
			continue
		}

		// 失败回退：若当前字符可能是 pat[0]，则 matchIdx=1
		if b == pat[0] {
			s.matchIdx = 1
		} else {
			s.matchIdx = 0
		}
	}
}

func (s *usageScanner) Usage() *imagesUsage {
	return s.usage
}