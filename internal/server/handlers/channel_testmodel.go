package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

const (
	// testModelTimeout 单次模型测试的上游等待上限, 超过即判定超时, 避免界面请求无限挂起。
	testModelTimeout = 120 * time.Second
	// testModelErrorLimit 回给界面的上游错误摘要长度上限(字符), 与探测接口截断上游正文的口径保持一致。
	testModelErrorLimit = 512
)

func init() {
	router.NewGroupRouter("/api/v1/channel").
		Use(middleware.Auth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/test-model", http.MethodPost).
				Handle(testChannelModel),
		)
}

// channelTestModelRequest 用编辑表单当前的未保存配置与指定凭据, 对单个模型发起一次真实测试。
// 与 fetch-model 同理直接收整份渠道配置; 请求不带提示词, 由后端读取全局 test_prompt 设置。
type channelTestModelRequest struct {
	Channel  model.ChannelConfig `json:"channel"`  // 当前未保存的渠道配置, 提供地址, 路径, 代理与 Header。
	Key      string              `json:"key"`      // 选定凭据的明文; 只用当前这一把, 不做替换。
	Model    string              `json:"model"`    // 待测试的模型名称。
	Protocol model.Protocol      `json:"protocol"` // 单个协议位: chat 2 / response 4 / message 8。
}

// testChannelModel 真实请求上游完成一次非流式模型测试, 返回回复文本, 耗时与实际协议。
// 成功回复原样返回由界面滚动展示; 提示词取全局设置且读取失败如实报错; 上游错误脱敏限长后返回, 绝不透出凭据明文。
func testChannelModel(c *gin.Context) {
	var request channelTestModelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}

	key := strings.TrimSpace(request.Key)
	modelName := strings.TrimSpace(request.Model)
	if key == "" || modelName == "" {
		resp.Error(c, http.StatusBadRequest, "key and model are required")
		return
	}
	// 只接受单个协议位: 测试一次只走一条协议, 由前端按 Response > Message > Chat 选定后传入。
	switch request.Protocol {
	case model.ProtocolOpenAIChatCompletion, model.ProtocolOpenAIResponse, model.ProtocolAnthropicMessage:
	default:
		resp.Error(c, http.StatusBadRequest, "protocol must be one of chat, response or message")
		return
	}

	// 收的是尚未落库的提交配置, 不经 normalizeChannelConfig, 故在此自行去空白; 只有地址是硬需求。
	channel := model.Channel{ChannelConfig: request.Channel}
	channel.BaseURL = strings.TrimSpace(channel.BaseURL)
	channel.OpenAIChatCompletionPath = strings.TrimSpace(channel.OpenAIChatCompletionPath)
	channel.OpenAIResponsePath = strings.TrimSpace(channel.OpenAIResponsePath)
	channel.AnthropicMessagePath = strings.TrimSpace(channel.AnthropicMessagePath)
	channel.ChannelProxy = strings.TrimSpace(channel.ChannelProxy)
	channel.ParamOverride = strings.TrimSpace(channel.ParamOverride)
	if channel.BaseURL == "" {
		resp.Error(c, http.StatusBadRequest, "channel base url is required")
		return
	}

	// 提示词取全局设置: 读取失败如实报错; 设置为空时按用户原样传入, 不做替换(默认值只在设置缺失初始化时生效)。
	prompt, err := op.SettingGetString(model.SettingKeyTestPrompt)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, fmt.Sprintf("failed to read test prompt setting: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), testModelTimeout)
	defer cancel()

	result, err := relay.TestModel(ctx, channel, key, modelName, prompt, request.Protocol)
	if err != nil {
		// 客户端已断开时无需再写响应, 取消已随上下文传入上游, 此处直接结束。
		if c.Request.Context().Err() != nil {
			c.Abort()
			return
		}
		status := http.StatusBadGateway
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		}
		resp.Error(c, status, sanitizeTestModelError(err, key))
		return
	}

	// 成功回复原样返回由界面滚动展示, 不做截断, 以免长回答被误认为上游异常。
	resp.Success(c, result)
}

// sanitizeTestModelError 生成可安全回给界面的上游错误摘要: 抹掉请求凭据, 去掉控制字符并限制长度。
// 上游鉴权失败时可能回显凭据或整页 HTML, 直接透出既会泄漏密钥也会污染提示。
func sanitizeTestModelError(err error, key string) string {
	message := err.Error()
	if key != "" {
		message = strings.ReplaceAll(message, key, "[redacted]")
	}
	message = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case r < 0x20 || r == 0x7f:
			return -1
		default:
			return r
		}
	}, message)
	message = strings.Join(strings.Fields(message), " ")
	message = truncateTestModelText(message, testModelErrorLimit)
	if message == "" {
		return "upstream request failed"
	}
	return message
}

// truncateTestModelText 按字符数截断文本并追加省略号, 仅用于错误摘要, 成功回复不截断。
func truncateTestModelText(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}
