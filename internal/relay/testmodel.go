package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

// ModelTestResult 是一次真实模型测试对界面可见的结果: 回复文本, 端到端耗时与实际执行的上游协议。
type ModelTestResult struct {
	Content   string         `json:"content"`    // 上游回复文本; 上游给出推理内容时一并包含。
	LatencyMS int64          `json:"latency_ms"` // 从发起到读完完整响应的端到端耗时(毫秒)。
	Protocol  model.Protocol `json:"protocol"`   // 实际执行的上游协议位。
}

// ErrEmptyTestResponse 表示上游以成功状态返回但没有任何可见文本; 调用方不得据此判定测试成功。
var ErrEmptyTestResponse = errors.New("upstream returned an empty response")

// TestModel 以未落库的渠道配置与凭据向真实上游发起一次非流式模型测试, 返回回复文本与实际协议。
// 复用转发链路的出站转换, 代理, 自定义路径/Header 与参数覆盖: 三条协议共用这一条实现, 不另写平行分支。
// 入站固定按 OpenAI Chat 组织提示词, 出站按 protocol 选定的协议转换, 因而三种协议都经过真实的跨协议转换。
// protocol 必须是单个协议位, 由调用方校验; 不做重试与回落, 失败原样返回交由调用方安全处理。
func TestModel(ctx context.Context, channel model.Channel, key, modelName, prompt string, protocol model.Protocol) (*ModelTestResult, error) {
	// 借单协议授权复用协议到转换器的映射: 只测指定协议, 不做任何回落。
	outbound, selected, _, err := buildOutbound(channel, model.ChannelGrant{Protocols: protocol}, model.ChannelKey{ChannelKeyConfig: model.ChannelKeyConfig{Key: key}}, protocol)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(map[string]any{
		"model":    modelName,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   false,
	})
	if err != nil {
		return nil, fmt.Errorf("build model test request: %w", err)
	}
	raw := &httpclient.Request{
		Method:  http.MethodPost,
		URL:     "/v1/chat/completions",
		Headers: http.Header{"Content-Type": []string{"application/json"}},
		Body:    body,
	}

	startedAt := time.Now()
	result, err := sendConverted(ctx, llm.APIFormatOpenAIChatCompletion, raw, channel, outbound, false)
	latency := time.Since(startedAt).Milliseconds()
	if err != nil {
		return nil, err
	}
	if result == nil || len(result.body) == 0 {
		return nil, ErrEmptyTestResponse
	}

	// 客户端协议固定为 Chat, 故响应正文即统一 Chat 结构; 解析后可同时取到正文与推理内容。
	var parsed llm.Response
	if err := json.Unmarshal(result.body, &parsed); err != nil {
		return nil, fmt.Errorf("decode model test response: %w", err)
	}
	content := testResponseText(&parsed)
	if content == "" {
		return nil, ErrEmptyTestResponse
	}
	return &ModelTestResult{Content: content, LatencyMS: latency, Protocol: selected}, nil
}

// testResponseText 从统一响应中取出可见文本: 回复正文为主, 上游给出推理内容时一并展示。
// 两者皆空返回空串, 由调用方判定失败, 避免把没有内容的上游响应当作测试成功。
func testResponseText(response *llm.Response) string {
	for _, choice := range response.Choices {
		message := choice.Message
		if message == nil {
			message = choice.Delta
		}
		if message == nil {
			continue
		}
		content := testMessageContentText(message.Content)
		reasoning := testMessageReasoningText(message)
		switch {
		case content != "" && reasoning != "":
			return reasoning + "\n\n" + content
		case content != "":
			return content
		case reasoning != "":
			return reasoning
		}
	}
	return ""
}

// testMessageContentText 拼接消息正文: 单段文本直接取用, 多段内容只保留其中的文本片段。
func testMessageContentText(content llm.MessageContent) string {
	if content.Content != nil {
		return strings.TrimSpace(*content.Content)
	}
	parts := make([]string, 0, len(content.MultipleContent))
	for _, part := range content.MultipleContent {
		if part.Type == "text" && part.Text != nil && *part.Text != "" {
			parts = append(parts, *part.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// testMessageReasoningText 取出推理内容, 兼容 reasoning_content 与 reasoning 两种字段。
func testMessageReasoningText(message *llm.Message) string {
	if message.ReasoningContent != nil && strings.TrimSpace(*message.ReasoningContent) != "" {
		return strings.TrimSpace(*message.ReasoningContent)
	}
	if message.Reasoning != nil {
		return strings.TrimSpace(*message.Reasoning)
	}
	return ""
}
