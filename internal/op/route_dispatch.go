package op

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
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
	Role      string `json:"role"`
	Content   string `json:"content"`
	Reasoning string `json:"reasoning"`
}

// dispatchResponse 分派 LLM 响应体
type dispatchResponse struct {
	Choices []struct {
		Message dispatchMessage `json:"message"`
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
	Attempts     []model.ChannelAttempt // 所有通道尝试记录（含跳过/失败）
}

// RouteDispatch 调用分派分组的 LLM 分析请求，返回应处理的工作分组
// 简单遍历分派分组的通道，逐个尝试直到成功
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

	dispatchGroupName := dispatchGroup.Name

	// 2. 构建 prompt
	systemMsg, userMsg := buildDispatchPrompt(route.WorkGroups, requestModel, requestContent)

	// 3. 按优先级排序通道（升序：数字越小优先级越高），逐个尝试
	candidates := make([]model.GroupItem, len(dispatchGroup.Items))
	copy(candidates, dispatchGroup.Items)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority < candidates[j].Priority
	})

	var lastErr error
	var lastCallInfo DispatchCallInfo
	var allAttempts []model.ChannelAttempt
	for _, item := range candidates {
		channel, err := ChannelGet(item.ChannelID, ctx)
		if err != nil {
			log.Warnf("dispatch: channel %d not found: %v", item.ChannelID, err)
			allAttempts = append(allAttempts, model.ChannelAttempt{
				ChannelID:   item.ChannelID,
				ChannelName: fmt.Sprintf("channel_%d", item.ChannelID),
				Status:      model.AttemptSkipped,
				Msg:         fmt.Sprintf("channel not found: %v", err),
			})
			lastErr = err
			continue
		}
		if !channel.Enabled {
			log.Warnf("dispatch: channel %s (id=%d) disabled, skip", channel.Name, channel.ID)
			allAttempts = append(allAttempts, model.ChannelAttempt{
				ChannelID:   channel.ID,
				ChannelName: channel.Name,
				Status:      model.AttemptSkipped,
				Msg:         "channel disabled",
			})
			continue
		}
		usedKey := channel.GetChannelKey()
		if usedKey.ChannelKey == "" {
			log.Warnf("dispatch: channel %s (id=%d) has no available key, skip", channel.Name, channel.ID)
			allAttempts = append(allAttempts, model.ChannelAttempt{
				ChannelID:   channel.ID,
				ChannelName: channel.Name,
				Status:      model.AttemptSkipped,
				Msg:         "no available key",
			})
			continue
		}

		itemModel := strings.TrimSpace(item.ModelName)
		if itemModel == "" {
			itemModel = getModelName(channel)
		}
		if itemModel == "" {
			log.Warnf("dispatch: channel %s (id=%d) has no usable model name, skip", channel.Name, channel.ID)
			allAttempts = append(allAttempts, model.ChannelAttempt{
				ChannelID:   channel.ID,
				ChannelName: channel.Name,
				Status:      model.AttemptSkipped,
				Msg:         "no usable model name",
			})
			continue
		}

		groupID, callInfo, err := callDispatchLLM(channel, &usedKey, itemModel, systemMsg, userMsg, ctx)
		lastCallInfo = callInfo
		if err != nil {
			log.Warnf("dispatch LLM try channel=%s model=%s failed: %v", channel.Name, itemModel, err)
			allAttempts = append(allAttempts, model.ChannelAttempt{
				ChannelID:   channel.ID,
				ChannelName: channel.Name,
				ModelName:   itemModel,
				Status:      model.AttemptFailed,
				Duration:    int(callInfo.Duration.Milliseconds()),
				Msg:         err.Error(),
			})
			lastErr = err
			continue
		}
		// 成功调用
		allAttempts = append(allAttempts, model.ChannelAttempt{
			ChannelID:   channel.ID,
			ChannelName: channel.Name,
			ModelName:   itemModel,
			Status:      model.AttemptSuccess,
			Duration:    int(callInfo.Duration.Milliseconds()),
		})

		// 4. 查找目标工作分组
		for _, wg := range route.WorkGroups {
			if wg.GroupID == groupID {
				targetGroup, err := GroupGetEnabled(wg.GroupID, ctx)
				if err == nil && len(targetGroup.Items) > 0 {
					dispatchDesc := fmt.Sprintf("llm: %s(%d)", wg.Category, wg.GroupID)
					log.Infof("dispatch LLM selected work group: %s (group_id=%d) via channel=%s", wg.Category, wg.GroupID, channel.Name)
					callInfo.Attempts = allAttempts
				return targetGroup, dispatchGroupName, dispatchDesc, callInfo, nil
				}
				break
			}
		}
		lastErr = fmt.Errorf("dispatch LLM returned unknown group_id: %d", groupID)
		log.Warnf("dispatch LLM returned unknown group_id=%d from channel=%s", groupID, channel.Name)
	}

	return model.Group{}, dispatchGroupName, "", DispatchCallInfo{
		ChannelID:   lastCallInfo.ChannelID,
		ChannelName: lastCallInfo.ChannelName,
		Attempts:    allAttempts,
		StartTime:   lastCallInfo.StartTime,
	}, fmt.Errorf("dispatch LLM failed: %w", lastErr)
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
		MaxTokens:   512,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, emptyInfo, fmt.Errorf("failed to marshal dispatch request: %w", err)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}

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
	if content == "" {
		content = strings.TrimSpace(dispatchResp.Choices[0].Message.Reasoning)
		log.Infof("dispatch LLM content empty, fallback to reasoning: %s", content)
	}
	// Strip <think>...</think> tags if present (reasoning models wrap output)
	content = stripThinkingTags(content)
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

// stripThinkingTags 移除 <think>...</think> 标签
func stripThinkingTags(s string) string {
	for {
		start := strings.Index(s, "<think>")
		if start == -1 {
			return s
		}
		end := strings.Index(s[start:], "</think>")
		if end == -1 {
			return s
		}
		s = strings.TrimSpace(s[:start] + s[start+end+7:])
	}
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
