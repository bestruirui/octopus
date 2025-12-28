package relay

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/client"
	dbmodel "github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/transformer/inbound"
	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/tmaxmax/go-sse"
)

// hopByHopHeaders 定义不应转发的 HTTP 头
var hopByHopHeaders = map[string]bool{
	"authorization":       true,
	"x-api-key":           true,
	"connection":          true,
	"keep-alive":          true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailer":             true,
	"transfer-encoding":   true,
	"upgrade":             true,
	"content-length":      true,
	"host":                true,
	"accept-encoding":     true,
}

// relayContext 保存请求转发过程中的上下文信息
type relayContext struct {
	c               *gin.Context
	inAdapter       model.Inbound
	outAdapter      model.Outbound
	internalRequest *model.InternalLLMRequest
	channel         *dbmodel.Channel
	metrics         *RelayMetrics
}

// Handler 处理入站请求并转发到上游服务
func Handler(inboundType inbound.InboundType, c *gin.Context) {
	// 解析请求
	internalRequest, inAdapter, err := parseRequest(inboundType, c)
	if err != nil {
		return
	}
	supportedModels := c.GetString("supported_models")
	if supportedModels != "" {
		supportedModelsArray := strings.Split(supportedModels, ",")
		if !slices.Contains(supportedModelsArray, internalRequest.Model) {
			resp.Error(c, http.StatusBadRequest, "model not supported")
			return
		}
	}

	// 初始化统计和日志
	apiKeyID := c.GetInt("api_key_id")
	metrics := NewRelayMetrics(internalRequest.Model)
	metrics.SetInternalRequest(internalRequest)
	metrics.SetAPIKeyID(apiKeyID)
	// 获取通道分组
	group, err := op.GroupGetMap(internalRequest.Model, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusNotFound, "model not found")
		return
	}

	const maxRounds = 3
	var lastErr error
	itemCount := len(group.Items)
	b := balancer.GetBalancer(group.Mode)
	for round := 0; round < maxRounds; round++ {
		item := b.Select(group.Items)
		if item == nil {
			resp.Error(c, http.StatusServiceUnavailable, "no available channel")
			return
		}

		for i := 0; i < itemCount; i++ {
			select {
			case <-c.Request.Context().Done():
				log.Infof("request context canceled, stopping retry")
				return
			default:
			}

			channel, err := op.ChannelGet(item.ChannelID, c.Request.Context())
			if err != nil {
				log.Warnf("failed to get channel: %v", err)
				lastErr = err
				item = b.Next(group.Items, item)
				continue
			}
			if channel.Enabled == false {
				log.Warnf("channel %s is disabled", channel.Name)
				lastErr = fmt.Errorf("channel %s is disabled", channel.Name)
				item = b.Next(group.Items, item)
				continue
			}

			log.Infof("request model %s, mode: %d, forwarding to channel: %s model: %s (round %d/%d, item %d/%d)", internalRequest.Model, group.Mode, channel.Name, item.ModelName, round+1, maxRounds, i+1, itemCount)

			internalRequest.Model = item.ModelName
			metrics.SetChannel(channel.ID, channel.Name, item.ModelName)

			outAdapter := outbound.Get(channel.Type)
			if outAdapter == nil {
				log.Warnf("unsupported channel type: %d for channel: %s", channel.Type, channel.Name)
				lastErr = fmt.Errorf("unsupported channel type: %d", channel.Type)
				item = b.Next(group.Items, item)
				continue
			}

			rc := &relayContext{
				c:               c,
				inAdapter:       inAdapter,
				outAdapter:      outAdapter,
				internalRequest: internalRequest,
				channel:         channel,
				metrics:         metrics,
			}

			if err := rc.forward(); err == nil {
				rc.collectResponse()
				metrics.Save(c.Request.Context(), true, nil)
				return
			} else {
				lastErr = fmt.Errorf("channel %s failed: %v", channel.Name, err)
			}
			item = b.Next(group.Items, item)
		}
	}

	// 所有通道都失败
	metrics.Save(c.Request.Context(), false, lastErr)
	resp.Error(c, http.StatusBadGateway, "all channels failed")
}

// parseRequest 解析并验证入站请求
func parseRequest(inboundType inbound.InboundType, c *gin.Context) (*model.InternalLLMRequest, model.Inbound, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return nil, nil, err
	}

	inAdapter := inbound.Get(inboundType)
	internalRequest, err := inAdapter.TransformRequest(c.Request.Context(), body)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return nil, nil, err
	}

	if err := internalRequest.Validate(); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return nil, nil, err
	}

	return internalRequest, inAdapter, nil
}

// forward 转发请求到上游服务
func (rc *relayContext) forward() error {
	ctx := rc.c.Request.Context()

	// 构建出站请求
	outboundRequest, err := rc.outAdapter.TransformRequest(
		ctx,
		rc.internalRequest,
		rc.channel.BaseURL,
		rc.channel.Key,
	)
	if err != nil {
		log.Warnf("failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 复制请求头
	rc.copyHeaders(outboundRequest)

	// 发送请求
	response, err := rc.sendRequest(outboundRequest)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	// 检查响应状态
	if response.StatusCode != http.StatusOK {
		errMsg := rc.readUpstreamError(response)
		return fmt.Errorf("upstream error %d: %s", response.StatusCode, errMsg)
	}

	// 处理响应
	if rc.isStreamRequest() {
		return rc.handleStreamResponse(ctx, response)
	}
	return rc.handleResponse(ctx, response)
}

// copyHeaders 复制请求头，过滤 hop-by-hop 头
func (rc *relayContext) copyHeaders(outboundRequest *http.Request) {
	for key, values := range rc.c.Request.Header {
		if hopByHopHeaders[strings.ToLower(key)] {
			continue
		}
		for _, value := range values {
			outboundRequest.Header.Set(key, value)
		}
	}
}

// sendRequest 发送 HTTP 请求
func (rc *relayContext) sendRequest(req *http.Request) (*http.Response, error) {
	httpClient, err := client.GetHTTPClient(rc.channel.Proxy)
	if err != nil {
		log.Warnf("failed to get http client: %v", err)
		return nil, err
	}

	response, err := httpClient.Do(req)
	if err != nil {
		log.Warnf("failed to send request: %v", err)
		return nil, err
	}

	return response, nil
}

// readUpstreamError 读取上游错误信息
func (rc *relayContext) readUpstreamError(response *http.Response) string {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Warnf("failed to read response body: %v", err)
		return "failed to read error body"
	}
	errMsg := string(body)
	log.Warnf("upstream server error: %d - %s", response.StatusCode, errMsg)
	return errMsg
}

// isStreamRequest 判断是否为流式请求
func (rc *relayContext) isStreamRequest() bool {
	return rc.internalRequest.Stream != nil && *rc.internalRequest.Stream
}

// handleStreamResponse 处理流式响应
func (rc *relayContext) handleStreamResponse(ctx context.Context, response *http.Response) error {
	// 设置 SSE 响应头
	rc.c.Header("Content-Type", "text/event-stream")
	rc.c.Header("Cache-Control", "no-cache")
	rc.c.Header("Connection", "keep-alive")
	rc.c.Header("X-Accel-Buffering", "no")

	firstToken := true
	for ev, err := range sse.Read(response.Body, nil) {
		// 检查客户端是否断开
		select {
		case <-ctx.Done():
			log.Infof("client disconnected, stopping stream")
			return nil
		default:
		}

		if err != nil {
			log.Warnf("failed to read event: %v", err)
			break
		}

		// 转换流式数据
		data, err := rc.transformStreamData(ctx, ev.Data)
		if err != nil || len(data) == 0 {
			continue
		}
		// 记录首个 Token 时间
		if firstToken {
			rc.metrics.SetFirstTokenTime(time.Now())
			firstToken = false
		}

		rc.c.Writer.Write(data)
		rc.c.Writer.Flush()
	}

	log.Infof("stream end")
	return nil
}

// transformStreamData 转换流式数据
func (rc *relayContext) transformStreamData(ctx context.Context, data string) ([]byte, error) {
	// 上游格式 → 内部格式
	internalStream, err := rc.outAdapter.TransformStream(ctx, []byte(data))
	if err != nil {
		log.Warnf("failed to transform stream: %v", err)
		return nil, err
	}
	if internalStream == nil {
		return nil, nil
	}

	// 内部格式 → 入站格式
	inStream, err := rc.inAdapter.TransformStream(ctx, internalStream)
	if err != nil {
		log.Warnf("failed to transform stream: %v", err)
		return nil, err
	}

	return inStream, nil
}

// handleResponse 处理非流式响应
func (rc *relayContext) handleResponse(ctx context.Context, response *http.Response) error {
	// 上游格式 → 内部格式
	internalResponse, err := rc.outAdapter.TransformResponse(ctx, response)
	if err != nil {
		log.Warnf("failed to transform response: %v", err)
		return fmt.Errorf("failed to transform outbound response: %w", err)
	}

	// 内部格式 → 入站格式
	inResponse, err := rc.inAdapter.TransformResponse(ctx, internalResponse)
	if err != nil {
		log.Warnf("failed to transform response: %v", err)
		return fmt.Errorf("failed to transform inbound response: %w", err)
	}

	rc.c.Data(http.StatusOK, "application/json", inResponse)
	return nil
}

// collectResponse 收集响应信息
func (rc *relayContext) collectResponse() {
	internalResponse, err := rc.inAdapter.GetInternalResponse(rc.c.Request.Context())
	if err != nil || internalResponse == nil {
		return
	}

	// 设置响应内容
	rc.metrics.SetInternalResponse(internalResponse)
}
