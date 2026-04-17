package op

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/transformer/outbound"
	"github.com/lingyuins/octopus/internal/utils/log"
	"github.com/lingyuins/octopus/internal/utils/xstrings"
	"golang.org/x/net/proxy"
)

const (
	aiRouteHTTPTimeout     = 60 * time.Second
	aiRouteResponseMaxSize = 2 << 20
)

const aiRouteSystemPrompt = `你是一个模型路由分析器。你的任务是根据给定的模型列表，识别哪些模型本质上是同一类模型，并为它们生成统一的路由映射。
要求：
1. 只输出 JSON，不要输出任何解释、Markdown、代码块标记。
2. 将语义相同或同系列的模型归一到一个 requested_model。
3. requested_model 应尽量使用简洁、稳定、常见的名称。
4. items 中每个元素表示一个可用上游：
   - channel_id: 整数，必须来自输入列表
   - upstream_model: 原始模型名，必须来自输入列表中相同 channel_id 下的 model
   - priority: 数字，越小优先级越高
   - weight: 数字，默认 100
5. 如果模型明显属于 embedding、rerank、tts、image、moderation、audio、speech 等非文本对话模型，则不要输出。
6. 如果一个模型名无法判断，不要强行归类。
7. 输出格式必须严格符合：
{
  "routes": [
    {
      "requested_model": "string",
      "items": [
        {
          "channel_id": 1,
          "upstream_model": "string",
          "priority": 1,
          "weight": 100
        }
      ]
    }
  ]
}`

type aiRouteChatCompletionRequest struct {
	Model       string                      `json:"model"`
	Messages    []aiRouteChatCompletionItem `json:"messages"`
	Temperature float64                     `json:"temperature"`
}

type aiRouteChatCompletionItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiRouteChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content any `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func GenerateAIRoute(ctx context.Context, req model.GenerateAIRouteRequest) (*model.GenerateAIRouteResult, error) {
	modelInputs, inputModelSet, err := collectAIRouteModelInputs(ctx)
	if err != nil {
		return nil, err
	}
	if len(modelInputs) == 0 {
		return nil, fmt.Errorf("没有可分析的模型")
	}

	if req.Scope == model.AIRouteScopeTable {
		return generateAIRouteTable(ctx, modelInputs, inputModelSet)
	}

	return generateAIRouteForGroup(ctx, req.GroupID, modelInputs, inputModelSet)
}

func generateAIRouteForGroup(
	ctx context.Context,
	groupID int,
	modelInputs []model.AIRouteModelInput,
	inputModelSet map[int]map[string]struct{},
) (*model.GenerateAIRouteResult, error) {
	if groupID <= 0 {
		value, err := SettingGetInt(model.SettingKeyAIRouteGroupID)
		if err != nil {
			return nil, fmt.Errorf("请先在设置中选择AI路由目标分组")
		}
		groupID = value
	}
	if groupID <= 0 {
		return nil, fmt.Errorf("请先在设置中选择AI路由目标分组")
	}

	group, err := GroupGet(groupID, ctx)
	if err != nil {
		return nil, fmt.Errorf("目标分组不存在")
	}

	routes, err := generateAIRoutesFromModelList(ctx, modelInputs, group.Name)
	if err != nil {
		return nil, err
	}

	selectedRoute, err := selectAIRouteForGroup(*group, routes)
	if err != nil {
		return nil, err
	}

	validatedItems, err := validateAIRouteItems(selectedRoute, inputModelSet)
	if err != nil {
		return nil, err
	}
	if len(validatedItems) == 0 {
		return nil, fmt.Errorf("AI 返回结果为空，未写入任何路由")
	}

	addedCount, err := syncGroupItemsWithAIRoute(ctx, group.ID, validatedItems)
	if err != nil {
		return nil, err
	}

	log.Infof("ai route generated successfully: group_id=%d routes=%d validated_items=%d added_items=%d",
		group.ID, len(routes), len(validatedItems), addedCount)

	return &model.GenerateAIRouteResult{
		Scope:      model.AIRouteScopeGroup,
		GroupID:    group.ID,
		GroupCount: 1,
		RouteCount: len(routes),
		ItemCount:  addedCount,
	}, nil
}

func generateAIRouteTable(
	ctx context.Context,
	modelInputs []model.AIRouteModelInput,
	inputModelSet map[int]map[string]struct{},
) (*model.GenerateAIRouteResult, error) {
	routes, err := generateAIRoutesFromModelList(ctx, modelInputs, "")
	if err != nil {
		return nil, err
	}

	existingGroups, err := GroupList(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取现有分组失败: %w", err)
	}

	groupByName := make(map[string]model.Group, len(existingGroups))
	for _, group := range existingGroups {
		name := strings.ToLower(strings.TrimSpace(group.Name))
		if name == "" {
			continue
		}
		groupByName[name] = group
	}

	affectedGroups := 0
	addedItems := 0
	for _, route := range routes {
		validatedItems, err := validateAIRouteItems(route, inputModelSet)
		if err != nil {
			log.Warnf("ai route skipped invalid route: requested_model=%q err=%v", route.RequestedModel, err)
			continue
		}

		groupName := strings.TrimSpace(route.RequestedModel)
		if groupName == "" {
			continue
		}

		groupKey := strings.ToLower(groupName)
		if existing, ok := groupByName[groupKey]; ok {
			addedCount, err := syncGroupItemsWithAIRoute(ctx, existing.ID, validatedItems)
			if err != nil {
				log.Warnf("ai route failed to sync existing group: group=%q err=%v", existing.Name, err)
				continue
			}
			addedItems += addedCount
			affectedGroups++
			continue
		}

		createdGroup, addedCount, err := createAIRouteGroup(ctx, groupName, validatedItems)
		if err != nil {
			log.Warnf("ai route failed to create group: group=%q err=%v", groupName, err)
			continue
		}

		groupByName[groupKey] = *createdGroup
		addedItems += addedCount
		affectedGroups++
	}

	if affectedGroups == 0 {
		return nil, fmt.Errorf("AI 返回结果为空，未写入任何路由")
	}

	log.Infof("ai route table generated successfully: routes=%d groups=%d added_items=%d",
		len(routes), affectedGroups, addedItems)

	return &model.GenerateAIRouteResult{
		Scope:      model.AIRouteScopeTable,
		GroupCount: affectedGroups,
		RouteCount: len(routes),
		ItemCount:  addedItems,
	}, nil
}

func collectAIRouteModelInputs(ctx context.Context) ([]model.AIRouteModelInput, map[int]map[string]struct{}, error) {
	channels, err := ChannelList(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("收集模型列表失败: %w", err)
	}

	seen := make(map[string]struct{})
	result := make([]model.AIRouteModelInput, 0)
	modelSet := make(map[int]map[string]struct{})

	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}

		for _, modelName := range xstrings.SplitTrimCompact(",", channel.Model, channel.CustomModel) {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" {
				continue
			}

			key := fmt.Sprintf("%d\x00%s", channel.ID, strings.ToLower(modelName))
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			if _, ok := modelSet[channel.ID]; !ok {
				modelSet[channel.ID] = make(map[string]struct{})
			}
			modelSet[channel.ID][strings.ToLower(modelName)] = struct{}{}

			result = append(result, model.AIRouteModelInput{
				ChannelID:   channel.ID,
				ChannelName: channel.Name,
				Provider:    aiRouteProviderName(channel.Type),
				Model:       modelName,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].ChannelID != result[j].ChannelID {
			return result[i].ChannelID < result[j].ChannelID
		}
		return result[i].Model < result[j].Model
	})

	return result, modelSet, nil
}

func generateAIRoutesFromModelList(ctx context.Context, modelInputs []model.AIRouteModelInput, targetGroupName string) ([]model.AIRouteEntry, error) {
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

	payload, err := json.Marshal(modelInputs)
	if err != nil {
		return nil, fmt.Errorf("构造模型列表失败: %w", err)
	}

	userPrompt := fmt.Sprintf("请分析以下模型列表，并生成完整路由表。\n模型列表：\n%s", string(payload))
	if strings.TrimSpace(targetGroupName) != "" {
		userPrompt = fmt.Sprintf(
			"请分析以下模型列表，并生成路由表。\n本次目标分组名称为 %q，请优先输出 requested_model 为 %q 的路由；如果无法确定，可返回空 routes。\n模型列表：\n%s",
			targetGroupName,
			targetGroupName,
			string(payload),
		)
	}

	requestBody := aiRouteChatCompletionRequest{
		Model: strings.TrimSpace(modelName),
		Messages: []aiRouteChatCompletionItem{
			{Role: "system", Content: aiRouteSystemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("构造AI请求失败: %w", err)
	}

	httpClient, err := getAIRouteHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("初始化AI请求客户端失败: %w", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, aiRouteHTTPTimeout)
	defer cancel()

	endpoint, err := joinAIRouteChatCompletionsURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("AI路由模型配置不完整")
	}

	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建AI请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AI 分析失败: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, aiRouteResponseMaxSize))
	if err != nil {
		return nil, fmt.Errorf("读取AI响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI 分析失败: upstream status %d", resp.StatusCode)
	}

	var completionResp aiRouteChatCompletionResponse
	if err := json.Unmarshal(rawBody, &completionResp); err != nil {
		return nil, fmt.Errorf("AI返回结果不是合法JSON")
	}
	if len(completionResp.Choices) == 0 {
		return nil, fmt.Errorf("AI返回结果为空")
	}

	content, err := normalizeAIMessageContent(completionResp.Choices[0].Message.Content)
	if err != nil {
		return nil, err
	}

	jsonContent, err := extractJSON(content)
	if err != nil {
		return nil, err
	}

	var routeResp model.AIRouteResponse
	if err := json.Unmarshal([]byte(jsonContent), &routeResp); err != nil {
		return nil, fmt.Errorf("AI返回结果不是合法JSON")
	}

	normalizedRoutes := normalizeAIRouteEntries(routeResp.Routes)
	if len(normalizedRoutes) == 0 {
		return nil, fmt.Errorf("AI返回结果为空")
	}

	return normalizedRoutes, nil
}

func selectAIRouteForGroup(group model.Group, routes []model.AIRouteEntry) (model.AIRouteEntry, error) {
	for _, route := range routes {
		if strings.EqualFold(strings.TrimSpace(route.RequestedModel), strings.TrimSpace(group.Name)) {
			return route, nil
		}
	}
	if len(routes) == 1 {
		return routes[0], nil
	}
	return model.AIRouteEntry{}, fmt.Errorf("AI 返回结果未包含目标分组对应路由")
}

func validateAIRouteItems(route model.AIRouteEntry, inputModelSet map[int]map[string]struct{}) ([]model.GroupItem, error) {
	if strings.TrimSpace(route.RequestedModel) == "" {
		return nil, fmt.Errorf("AI返回结果缺少 requested_model")
	}
	if len(route.Items) == 0 {
		return nil, fmt.Errorf("AI返回结果为空")
	}

	seen := make(map[string]struct{})
	groupItems := make([]model.GroupItem, 0, len(route.Items))
	nextPriority := 1

	for _, item := range route.Items {
		if item.ChannelID <= 0 {
			return nil, fmt.Errorf("AI返回了不存在的channel_id: %d", item.ChannelID)
		}

		channelModels, ok := inputModelSet[item.ChannelID]
		if !ok {
			return nil, fmt.Errorf("AI返回了不存在的channel_id: %d", item.ChannelID)
		}

		upstreamModel := strings.TrimSpace(item.UpstreamModel)
		if upstreamModel == "" {
			return nil, fmt.Errorf("AI返回结果缺少 upstream_model")
		}
		if _, ok := channelModels[strings.ToLower(upstreamModel)]; !ok {
			return nil, fmt.Errorf("AI返回了不存在的upstream_model: channel_id=%d model=%q", item.ChannelID, upstreamModel)
		}

		key := fmt.Sprintf("%d\x00%s", item.ChannelID, strings.ToLower(upstreamModel))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		priority := item.Priority
		if priority <= 0 {
			priority = nextPriority
		}
		weight := item.Weight
		if weight <= 0 {
			weight = 100
		}

		groupItems = append(groupItems, model.GroupItem{
			ChannelID: item.ChannelID,
			ModelName: upstreamModel,
			Priority:  priority,
			Weight:    weight,
		})
		nextPriority++
	}

	if len(groupItems) == 0 {
		return nil, fmt.Errorf("AI返回结果为空")
	}

	sort.SliceStable(groupItems, func(i, j int) bool {
		if groupItems[i].Priority != groupItems[j].Priority {
			return groupItems[i].Priority < groupItems[j].Priority
		}
		if groupItems[i].ChannelID != groupItems[j].ChannelID {
			return groupItems[i].ChannelID < groupItems[j].ChannelID
		}
		return groupItems[i].ModelName < groupItems[j].ModelName
	})

	for i := range groupItems {
		groupItems[i].Priority = i + 1
	}

	return groupItems, nil
}

func normalizeAIRouteEntries(routes []model.AIRouteEntry) []model.AIRouteEntry {
	merged := make(map[string]*model.AIRouteEntry, len(routes))
	order := make([]string, 0, len(routes))

	for _, route := range routes {
		requestedModel := strings.TrimSpace(route.RequestedModel)
		if requestedModel == "" {
			continue
		}

		key := strings.ToLower(requestedModel)
		entry, ok := merged[key]
		if !ok {
			entry = &model.AIRouteEntry{
				RequestedModel: requestedModel,
				Items:          make([]model.AIRouteItemSpec, 0, len(route.Items)),
			}
			merged[key] = entry
			order = append(order, key)
		}

		for _, item := range route.Items {
			upstreamModel := strings.TrimSpace(item.UpstreamModel)
			if item.ChannelID <= 0 || upstreamModel == "" {
				continue
			}
			entry.Items = append(entry.Items, model.AIRouteItemSpec{
				ChannelID:     item.ChannelID,
				UpstreamModel: upstreamModel,
				Priority:      item.Priority,
				Weight:        item.Weight,
			})
		}
	}

	result := make([]model.AIRouteEntry, 0, len(order))
	for _, key := range order {
		entry := merged[key]
		if entry == nil {
			continue
		}
		entry.Items = dedupeAIRouteItems(entry.Items)
		if len(entry.Items) == 0 {
			continue
		}
		result = append(result, *entry)
	}

	return result
}

func dedupeAIRouteItems(items []model.AIRouteItemSpec) []model.AIRouteItemSpec {
	seen := make(map[string]struct{}, len(items))
	result := make([]model.AIRouteItemSpec, 0, len(items))

	for _, item := range items {
		upstreamModel := strings.TrimSpace(item.UpstreamModel)
		if item.ChannelID <= 0 || upstreamModel == "" {
			continue
		}

		key := fmt.Sprintf("%d\x00%s", item.ChannelID, strings.ToLower(upstreamModel))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		item.UpstreamModel = upstreamModel
		result = append(result, item)
	}

	return result
}

func syncGroupItemsWithAIRoute(ctx context.Context, groupID int, items []model.GroupItem) (int, error) {
	group, err := GroupGet(groupID, ctx)
	if err != nil {
		return 0, fmt.Errorf("目标分组不存在")
	}

	existing := make(map[string]struct{}, len(group.Items))
	nextPriority := 1
	for _, item := range group.Items {
		existing[fmt.Sprintf("%d\x00%s", item.ChannelID, strings.ToLower(strings.TrimSpace(item.ModelName)))] = struct{}{}
		if item.Priority >= nextPriority {
			nextPriority = item.Priority + 1
		}
	}

	itemsToAdd := make([]model.GroupItemAddRequest, 0, len(items))
	for _, item := range items {
		key := fmt.Sprintf("%d\x00%s", item.ChannelID, strings.ToLower(strings.TrimSpace(item.ModelName)))
		if _, ok := existing[key]; ok {
			continue
		}
		existing[key] = struct{}{}

		itemsToAdd = append(itemsToAdd, model.GroupItemAddRequest{
			ChannelID: item.ChannelID,
			ModelName: item.ModelName,
			Priority:  nextPriority,
			Weight:    item.Weight,
		})
		nextPriority++
	}

	if len(itemsToAdd) == 0 {
		return 0, nil
	}

	if _, err := GroupUpdate(&model.GroupUpdateRequest{
		ID:         groupID,
		ItemsToAdd: itemsToAdd,
	}, ctx); err != nil {
		return 0, fmt.Errorf("写入路由表失败")
	}
	return len(itemsToAdd), nil
}

func createAIRouteGroup(ctx context.Context, groupName string, items []model.GroupItem) (*model.Group, int, error) {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return nil, 0, fmt.Errorf("AI返回结果缺少 requested_model")
	}

	tx := db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	group := model.Group{
		Name:              groupName,
		EndpointType:      model.EndpointTypeAll,
		Mode:              model.GroupModeRoundRobin,
		MatchRegex:        "",
		FirstTokenTimeOut: 0,
		SessionKeepTime:   0,
	}
	if err := tx.Create(&group).Error; err != nil {
		tx.Rollback()
		return nil, 0, fmt.Errorf("创建分组失败: %w", err)
	}

	groupItems := make([]model.GroupItem, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	priority := 1
	for _, item := range items {
		modelName := strings.TrimSpace(item.ModelName)
		if item.ChannelID <= 0 || modelName == "" {
			continue
		}

		key := fmt.Sprintf("%d\x00%s", item.ChannelID, strings.ToLower(modelName))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		weight := item.Weight
		if weight <= 0 {
			weight = 100
		}

		groupItems = append(groupItems, model.GroupItem{
			GroupID:   group.ID,
			ChannelID: item.ChannelID,
			ModelName: modelName,
			Priority:  priority,
			Weight:    weight,
		})
		priority++
	}

	if len(groupItems) == 0 {
		tx.Rollback()
		return nil, 0, fmt.Errorf("AI返回结果为空")
	}

	if err := tx.Create(&groupItems).Error; err != nil {
		tx.Rollback()
		return nil, 0, fmt.Errorf("创建分组项失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, 0, fmt.Errorf("提交AI路由分组失败: %w", err)
	}

	if err := groupRefreshCacheByID(group.ID, ctx); err != nil {
		return nil, 0, err
	}

	createdGroup, err := GroupGet(group.ID, ctx)
	if err != nil {
		return nil, 0, err
	}
	return createdGroup, len(groupItems), nil
}

func getAIRouteHTTPClient() (*http.Client, error) {
	proxyURL, _ := SettingGetString(model.SettingKeyProxyURL)

	baseClient, err := newAIRouteHTTPClient(strings.TrimSpace(proxyURL))
	if err != nil {
		return nil, err
	}

	cloned := *baseClient
	cloned.Timeout = aiRouteHTTPTimeout
	return &cloned, nil
}

func newAIRouteHTTPClient(proxyURLStr string) (*http.Client, error) {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default transport is not *http.Transport")
	}

	cloned := transport.Clone()
	if proxyURLStr == "" {
		cloned.Proxy = nil
		return &http.Client{Transport: cloned}, nil
	}

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy url: %w", err)
	}

	switch proxyURL.Scheme {
	case "http", "https":
		cloned.Proxy = http.ProxyURL(proxyURL)
	case "socks", "socks5":
		socksDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("invalid socks proxy: %w", err)
		}
		cloned.Proxy = nil
		cloned.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return socksDialer.Dial(network, addr)
		}
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", proxyURL.Scheme)
	}

	return &http.Client{Transport: cloned}, nil
}

func joinAIRouteChatCompletionsURL(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid base url")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if strings.HasSuffix(parsed.Path, "/chat/completions") {
		return parsed.String(), nil
	}
	parsed.Path += "/chat/completions"
	return parsed.String(), nil
}

func normalizeAIMessageContent(content any) (string, error) {
	switch value := content.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("AI返回结果为空")
		}
		return value, nil
	case []any:
		var builder strings.Builder
		for _, item := range value {
			record, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := record["text"].(string); ok {
				builder.WriteString(text)
			}
		}
		result := strings.TrimSpace(builder.String())
		if result == "" {
			return "", fmt.Errorf("AI返回结果为空")
		}
		return result, nil
	default:
		return "", fmt.Errorf("AI返回结果为空")
	}
}

func extractJSON(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("AI返回结果为空")
	}

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < 0 || end <= start {
		return "", fmt.Errorf("AI返回结果不是合法JSON")
	}

	return content[start : end+1], nil
}

func aiRouteProviderName(provider outbound.OutboundType) string {
	switch provider {
	case outbound.OutboundTypeOpenAIChat:
		return "openai_chat"
	case outbound.OutboundTypeOpenAIResponse:
		return "openai_response"
	case outbound.OutboundTypeAnthropic:
		return "anthropic"
	case outbound.OutboundTypeGemini:
		return "gemini"
	case outbound.OutboundTypeVolcengine:
		return "volcengine"
	case outbound.OutboundTypeOpenAIEmbedding:
		return "openai_embedding"
	default:
		return fmt.Sprintf("provider_%d", provider)
	}
}
