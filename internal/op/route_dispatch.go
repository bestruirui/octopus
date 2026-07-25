package op

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/log"
	llm "github.com/looplj/axonhub/llm"
)

// dispatchRequest 分派 LLM 请求体（OpenAI chat completion 格式）
type dispatchRequest struct {
	Model       string              `json:"model"`
	Messages    []dispatchMessage   `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type dispatchMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// dispatchResponse 分派 LLM 响应体
type dispatchResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// DispatchCallInfo 记录 dispatch LLM 调用的完整信息，供调用方写日志
type DispatchCallInfo struct {
	ChannelID    int
	ChannelName  string
	ModelName    string
	RequestJSON  string // 完整请求体 JSON（含 system + user prompt）
	ResponseText string // LLM 返回的原始文本（group ID）
	ResponseJSON string // 完整响应体 JSON（供日志展示）
	Duration     time.Duration
	StartTime    time.Time
}

// RouteDispatch 调用分派分组的 LLM 分析请求，返回应处理的工作分组
// 遍历所有可用 channel 直到成功，而非仅尝试一个
// 返回：目标分组、分派分组名、分派结果描述、调用信息、错误
func RouteDispatch(route *model.Route, requestModel string, requestContent string, ctx context.Context) (model.Group, string, string, DispatchCallInfo, error) {
	var emptyInfo DispatchCallInfo

	if route.DispatchGroupID == nil || *route.DispatchGroupID <= 0 {
		return model.Group{}, "", "", emptyInfo, fmt.Errorf("dispatch group not configured")
	}

	// 1. 获取分派分组
	dispatchGroup, err := GroupGetEnabled(*route.DispatchGroupID, ctx)
	if err != nil {
		return model.Group{}, "", "", emptyInfo, fmt.Errorf("dispatch group not found: %w", err)
	}
	if len(dispatchGroup.Items) == 0 {
		return model.Group{}, "", "", emptyInfo, fmt.Errorf("dispatch group has no enabled items")
	}

	// 获取分派分组名称
	dispatchGroupName := dispatchGroup.Name

	// 2. 构建 prompt（system + user 两个消息，包含用户消息内容）
	systemMsg, userMsg := buildDispatchPrompt(route.WorkGroups, requestModel, requestContent)

	// 3. 收集所有可用 channel+key 组合，逐个尝试
	candidates := collectDispatchCandidates(dispatchGroup)
	if len(candidates) == 0 {
		return model.Group{}, dispatchGroupName, "", emptyInfo, fmt.Errorf("no usable channel in dispatch group")
	}

	var lastErr error
	var lastCallInfo DispatchCallInfo
	for _, c := range candidates {
		groupID, callInfo, err := callDispatchLLM(c.channel, c.key, c.model, systemMsg, userMsg, ctx)
		lastCallInfo = callInfo
		if err != nil {
			log.Warnf("dispatch LLM try channel=%s model=%s failed: %v", c.channel.Name, c.model, err)
			lastErr = err
			continue
		}

		// 4. 查找目标工作分组
		for _, wg := range route.WorkGroups {
			if wg.GroupID == groupID {
				targetGroup, err := GroupGetEnabled(wg.GroupID, ctx)
				if err == nil && len(targetGroup.Items) > 0 {
					dispatchDesc := fmt.Sprintf("llm: %s(%d)", wg.Category, wg.GroupID)
					log.Infof("dispatch LLM selected work group: %s (group_id=%d) via channel=%s", wg.Category, wg.GroupID, c.channel.Name)
					return targetGroup, dispatchGroupName, dispatchDesc, callInfo, nil
				}
				break
			}
		}
		lastErr = fmt.Errorf("dispatch LLM returned unknown group_id: %d", groupID)
		log.Warnf("dispatch LLM returned unknown group_id=%d from channel=%s", groupID, c.channel.Name)
	}

	return model.Group{}, dispatchGroupName, "", lastCallInfo, fmt.Errorf("dispatch LLM failed after %d attempts: %w", len(candidates), lastErr)
}

type dispatchCandidate struct {
	channel *model.Channel
	key     *model.ChannelKey
	model   string // group_item.model_name 优先于 channel.Model
}

// collectDispatchCandidates 收集所有可用的 channel+key 组合
// 优先使用 group_item 的 model_name，而非 channel.Model
func collectDispatchCandidates(group model.Group) []dispatchCandidate {
	var candidates []dispatchCandidate
	for _, item := range group.Items {
		ch, ok := channelCache.Get(item.ChannelID)
		if !ok || !ch.Enabled {
			continue
		}
		// 优先使用 group_item 的 model_name
		itemModel := strings.TrimSpace(item.ModelName)
		if itemModel == "" {
			itemModel = getModelName(&ch)
		}
		if itemModel == "" {
			continue
		}
		chCopy := ch
		for _, k := range chCopy.Keys {
			if k.Enabled && k.ChannelKey != "" {
				candidates = append(candidates, dispatchCandidate{channel: &chCopy, key: &k, model: itemModel})
			}
		}
	}
	return candidates
}

// getModelName 从 channel 获取可用的模型名（取 Model 列表第一个，或 CustomModel）
func getModelName(ch *model.Channel) string {
	if ch.Model != "" {
		for _, name := range strings.Split(ch.Model, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				return name
			}
		}
	}
	if ch.CustomModel != "" {
		for _, name := range strings.Split(ch.CustomModel, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				return name
			}
		}
	}
	return ""
}

// buildDispatchPrompt 构建分派 prompt，包含所有工作分组元数据和用户消息内容
// 使用 system + user 两个消息，提高 LLM 响应可靠性
func buildDispatchPrompt(workGroups []model.RouteGroup, requestModel string, requestContent string) (systemMsg string, userMsg string) {
	systemMsg = "You are a task dispatcher. You MUST return ONLY a single number (the group ID). No explanation, no text, just the number."

	var sb strings.Builder
	sb.WriteString("Available work groups:\n")
	for _, wg := range workGroups {
		sb.WriteString(fmt.Sprintf("- ID: %d, Name: %s", wg.GroupID, wg.Category))
		if wg.Description != "" {
			sb.WriteString(fmt.Sprintf(", Description: %s", wg.Description))
		}
		if wg.Keywords != "" {
			sb.WriteString(fmt.Sprintf(", Keywords: %s", wg.Keywords))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("\nRequest model: %s\n", requestModel))
	if requestContent != "" {
		// 截取前500字符避免 prompt 过长
		content := requestContent
		if len(content) > 500 {
			content = content[:500]
		}
		sb.WriteString(fmt.Sprintf("User message: %s\n", content))
	}
	sb.WriteString("\nWhich group should handle this request? Return ONLY the group ID number:")
	userMsg = sb.String()
	return
}

// callDispatchLLM 向分派分组的 channel 发送 chat completion 请求，解析返回的分组 ID
// 仅负责调用和返回数据，日志由调用方通过 op.RelayLogAdd 写入
func callDispatchLLM(channel *model.Channel, key *model.ChannelKey, modelName string, systemMsg string, userMsg string, ctx context.Context) (int, DispatchCallInfo, error) {
	startTime := time.Now()

	emptyInfo := DispatchCallInfo{StartTime: startTime}

	if channel.Type != llm.APIFormatOpenAIChatCompletion {
		return 0, emptyInfo, fmt.Errorf("dispatch channel type %s not supported, only openai/chat_completions is supported", channel.Type)
	}
	if len(channel.BaseUrls) == 0 {
		return 0, emptyInfo, fmt.Errorf("dispatch channel has no base url")
	}

	baseURL := channel.BaseUrls[0].URL
	baseURL = strings.TrimRight(baseURL, "/")

	if modelName == "" {
		modelName = getModelName(channel)
	}
	if modelName == "" {
		return 0, emptyInfo, fmt.Errorf("dispatch channel has no usable model name")
	}

	reqBody := dispatchRequest{
		Model: modelName,
		Messages: []dispatchMessage{
			{Role: "system", Content: systemMsg},
			{Role: "user", Content: userMsg},
		},
		Temperature: 0,
		MaxTokens:   10,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, emptyInfo, fmt.Errorf("failed to marshal dispatch request: %w", err)
	}

	httpClient := &http.Client{}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST", baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, emptyInfo, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key.ChannelKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, emptyInfo, fmt.Errorf("dispatch LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, emptyInfo, fmt.Errorf("failed to read dispatch response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, emptyInfo, fmt.Errorf("dispatch LLM returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var dispatchResp dispatchResponse
	if err := json.Unmarshal(respBody, &dispatchResp); err != nil {
		return 0, emptyInfo, fmt.Errorf("failed to unmarshal dispatch response: %w", err)
	}

	if len(dispatchResp.Choices) == 0 {
		return 0, emptyInfo, fmt.Errorf("dispatch LLM returned no choices")
	}

	content := strings.TrimSpace(dispatchResp.Choices[0].Message.Content)
	log.Infof("dispatch LLM raw response: %s", content)

	callInfo := DispatchCallInfo{
		ChannelID:    channel.ID,
		ChannelName:  channel.Name,
		ModelName:    modelName,
		RequestJSON:  string(bodyBytes),
		ResponseText: content,
		ResponseJSON: string(respBody),
		Duration:     time.Since(startTime),
		StartTime:    startTime,
	}

	content = extractNumber(content)
	var groupID int
	if _, err := fmt.Sscanf(content, "%d", &groupID); err != nil {
		return 0, callInfo, fmt.Errorf("failed to parse group ID from dispatch response: %q", content)
	}
	if groupID <= 0 {
		return 0, callInfo, fmt.Errorf("invalid group ID from dispatch response: %d", groupID)
	}

	return groupID, callInfo, nil
}

// extractNumber 从字符串中提取第一个连续数字
func extractNumber(s string) string {
	s = strings.TrimSpace(s)
	start := -1
	for i, ch := range s {
		if ch >= '0' && ch <= '9' {
			if start == -1 {
				start = i
			}
		} else if start != -1 {
			return s[start:i]
		}
	}
	if start != -1 {
		return s[start:]
	}
	return s
}
