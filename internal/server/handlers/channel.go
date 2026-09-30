package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/price"
	"github.com/bestruirui/octopus/internal/rhttp"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/channel").
		Use(middleware.Auth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/detail/:id", http.MethodGet).
				Handle(getChannelDetail),
		).
		AddRoute(
			router.NewRoute("/stats", http.MethodGet).
				Handle(listChannelStats),
		).
		AddRoute(
			router.NewRoute("/grants", http.MethodGet).
				Handle(listChannelGrant),
		).
		AddRoute(
			router.NewRoute("/create", http.MethodPost).
				Handle(createChannel),
		).
		AddRoute(
			router.NewRoute("/update", http.MethodPost).
				Handle(updateChannel),
		).
		AddRoute(
			router.NewRoute("/enable", http.MethodPost).
				Handle(enableChannel),
		).
		AddRoute(
			router.NewRoute("/delete/:id", http.MethodDelete).
				Handle(deleteChannel),
		).
		AddRoute(
			router.NewRoute("/fetch-model", http.MethodPost).
				Handle(fetchModel),
		)
}

// getChannelDetail 返回单个渠道的完整配置, 供编辑表单打开时读取。
// 与列表分开: 整份配置带着路径, 代理与凭据明文, 只有正在编辑的那一个渠道用得上。
func getChannelDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidParam)
		return
	}
	detail, err := op.ChannelDetailGet(id)
	if err != nil {
		resp.Error(c, http.StatusNotFound, err.Error())
		return
	}
	resp.Success(c, detail)
}

// listChannelStats 返回全部渠道及其模型的累计统计, 也是渠道列表页的数据来源。
// 不带整份配置: 统计每次转发都在变, 界面按更短的间隔刷新它, 而路径, 代理与凭据明文只在编辑时用得上。
func listChannelStats(c *gin.Context) {
	resp.Success(c, op.ChannelStatsList())
}

// listChannelGrant 返回全部渠道授权候选, 供分组页选取成员。
func listChannelGrant(c *gin.Context) {
	resp.Success(c, op.ChannelGrantCandidates())
}

func createChannel(c *gin.Context) {
	var req model.ChannelDetail
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	channel, err := op.ChannelCreate(&req, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := addChannelModelPrices(channel.Models, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, channel)
}

func updateChannel(c *gin.Context) {
	var req model.ChannelDetail
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if req.ID == 0 {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidParam)
		return
	}
	channel, err := op.ChannelUpdate(&req, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := addChannelModelPrices(channel.Models, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := op.LLMCleanupGhosts(c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, channel)
}

func enableChannel(c *gin.Context) {
	var request struct {
		ID      int  `json:"id"`
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.ChannelEnabled(request.ID, request.Enabled, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

func deleteChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidParam)
		return
	}
	if err := op.ChannelDel(id, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := op.LLMCleanupGhosts(c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

// addChannelModelPrices 为渠道模型匹配校准价格，并批量写入尚不存在的价格记录。
func addChannelModelPrices(modelNames []string, ctx context.Context) error {
	seen := make(map[string]struct{}, len(modelNames))
	llmInfos := make([]model.LLMInfo, 0, len(modelNames))
	for _, modelName := range modelNames {
		modelName = strings.ToLower(modelName)
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		llmInfo := model.LLMInfo{Name: modelName}
		if modelPrice := price.GetLLMPrice(modelName); modelPrice != nil {
			llmInfo.LLMPrice = *modelPrice
		}
		llmInfos = append(llmInfos, llmInfo)
	}
	return op.LLMBatchCreate(llmInfos, ctx)
}

// fetchModel 按提交的渠道配置与凭据拉取上游模型列表, 并按过滤表达式筛选后返回。
// 同时探测 OpenAI 与 Anthropic 两侧, 谁返回了哪些模型, 就给对应协议位打勾: 协议支持由探测结果决定, 无需用户声明。
// OpenAI 侧记为 Responses 而不是 Chat: Chat Completions 已被官方标记弃用, 新渠道应默认走 Responses,
// 仍需 Chat 的渠道由用户在界面上手动勾选。两侧的 /models 地址与认证形态不同, 故必须分别探测:
// 单协议上游只有一侧会成功, "哪侧成功" 本身就是协议支持的证据。
// 只有两侧都失败才算失败; 一侧失败属正常情况, 单协议上游本就只有一侧讲得通, 按成功那侧的结果返回。
func fetchModel(c *gin.Context) {
	var request model.ChannelFetchModelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	ctx := c.Request.Context()
	// 探测收的是尚未落库的提交配置, 不经 normalizeChannelConfig, 故在此自行去空白;
	// 其中只有地址是硬需求: 渠道尚未命名时也可试拉, 故名称不在此校验。
	target := request.Channel
	target.BaseURL = strings.TrimSpace(target.BaseURL)
	target.ChannelProxy = strings.TrimSpace(target.ChannelProxy)
	target.MatchRegex = strings.TrimSpace(target.MatchRegex)
	if target.BaseURL == "" {
		resp.Error(c, http.StatusBadRequest, "channel base url is required")
		return
	}

	var httpClient *http.Client
	var err error
	switch {
	case !target.Proxy:
		httpClient, err = rhttp.Direct()
	case target.ChannelProxy == "":
		httpClient, err = rhttp.Proxy()
	default:
		httpClient, err = rhttp.New(target.ChannelProxy)
		// 渠道专用代理的客户端不再共享, 探测完就得关掉空闲连接; 探测收的是未落库的输入, 留着也无从复用。
		if httpClient != nil {
			defer httpClient.CloseIdleConnections()
		}
	}
	if err != nil {
		resp.Error(c, http.StatusBadGateway, err.Error())
		return
	}

	var openaiModels, anthropicModels []string
	var openaiErr, anthropicErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		openaiModels, openaiErr = fetchOpenAIModels(httpClient, ctx, target, request.Key, modelsURL(target.BaseURL, target.OpenAIResponsePath))
	}()
	go func() {
		defer wg.Done()
		anthropicModels, anthropicErr = fetchAnthropicModels(httpClient, ctx, target, request.Key, modelsURL(target.BaseURL, target.AnthropicMessagePath))
	}()
	wg.Wait()

	if openaiErr != nil && anthropicErr != nil {
		// 上游鉴权失败或地址不通属于调用方配置问题, 按 502 返回并带上上游原文, 便于在界面上直接看到原因。
		resp.Error(c, http.StatusBadGateway, fmt.Sprintf("openai: %v; anthropic: %v", openaiErr, anthropicErr))
		return
	}

	// /v1/models 只是目录接口, 列得出模型不代表对话端点真的实现了该协议;
	// 不少聚合网关 (new-api 系) 的 /v1/models 全量放行, 而 /v1/responses 返回 500 not implemented。
	// 故两侧列表拉取成功后, 逐个取列表中的模型对对话端点发最小真实请求, 任一模型拿到 2xx 即打协议位;
	// 用多个模型验证是因为列表里可能混有 rerank / embedding 等非对话模型, 首个模型失败不代表端点不支持。
	// 某一端点全部失败时该侧保留模型列表但不打对应协议位, 由用户在界面上参照报错自行勾选。
	var endpointProtocols model.Protocol
	if openaiErr == nil && len(openaiModels) > 0 {
		if anyModelSucceeds(httpClient, ctx, target, request.Key, target.BaseURL+target.OpenAIChatCompletionPath, openAIStyleChat, openaiModels) {
			endpointProtocols |= model.ProtocolOpenAIChatCompletion
		}
		if anyModelSucceeds(httpClient, ctx, target, request.Key, target.BaseURL+target.OpenAIResponsePath, openAIStyleResponses, openaiModels) {
			endpointProtocols |= model.ProtocolOpenAIResponse
		}
	}
	if anthropicErr == nil && len(anthropicModels) > 0 {
		if anyModelSucceeds(httpClient, ctx, target, request.Key, target.BaseURL+target.AnthropicMessagePath, "anthropic", anthropicModels) {
			endpointProtocols |= model.ProtocolAnthropicMessage
		}
	}

	var re, reGlobal *regexp2.Regexp
	if target.MatchRegex != "" {
		if re, err = regexp2.Compile(target.MatchRegex, regexp2.ECMAScript); err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	// 全局过滤由设置页维护, 与渠道过滤同取 AND: 模型须同时通过两枚正则才保留, 留空的一侧不生效。
	// 设置缺失按不过滤处理: 启动初始化会补齐默认值, 缺行只可能出现在旧库尚未刷新的瞬间。
	globalFilter, _ := op.SettingGetString(model.SettingKeyModelFilter)
	if globalFilter != "" {
		if reGlobal, err = regexp2.Compile(globalFilter, regexp2.ECMAScript); err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
	}

	// 模型名须同时通过渠道与全局两枚过滤正则, 编译与匹配错误统一按请求错误返回。
	matches := func(name string) (bool, error) {
		if re != nil {
			matched, err := re.MatchString(name)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil
			}
		}
		if reGlobal != nil {
			matched, err := reGlobal.MatchString(name)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil
			}
		}
		return true, nil
	}

	// 两侧结果按名称合并成一份有序集合: 同名模型在两侧都出现时, 协议位取并集。
	// 保持首次出现的顺序, 界面上模型的排列才与上游返回的一致;
	// 先并入 OpenAI 再并入 Anthropic, 顺序写死而不用 map 遍历, 否则界面上的模型排列会随每次刷新变化。
	// 协议位不再仅由列表接口的存在推断: 上述对话端点验证通过才会打对应的位,
	// 避免 /v1/models 通而 /v1/responses 实际 500 的网关被误标为支持 Responses。
	protocolsByModel := make(map[string]model.Protocol, len(openaiModels)+len(anthropicModels))
	order := make([]string, 0, len(openaiModels)+len(anthropicModels))
	for _, name := range openaiModels {
		matched, err := matches(name)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		if !matched {
			continue
		}
		if _, ok := protocolsByModel[name]; !ok {
			order = append(order, name)
		}
		protocolsByModel[name] |= endpointProtocols & (model.ProtocolOpenAIChatCompletion | model.ProtocolOpenAIResponse)
	}
	for _, name := range anthropicModels {
		matched, err := matches(name)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		if !matched {
			continue
		}
		if _, ok := protocolsByModel[name]; !ok {
			order = append(order, name)
		}
		protocolsByModel[name] |= endpointProtocols & model.ProtocolAnthropicMessage
	}

	models := make([]model.ChannelFetchModel, 0, len(order))
	for _, name := range order {
		models = append(models, model.ChannelFetchModel{Name: name, Protocols: protocolsByModel[name]})
	}
	resp.Success(c, models)
}

// modelsURL 取协议请求路径的父级目录, 与地址拼成同级的 /models 地址。
// 例如 /v1/chat/completions 与 /v1/messages 都得到 /v1/models, /chat/completions 得到 /models。
func modelsURL(baseURL, protocolPath string) string {
	parent := path.Dir(strings.TrimRight(protocolPath, "/"))
	// Anthropic 的 /v1/messages 只有一层, 父级即 /v1; Chat 的 /v1/chat/completions 需要再上一层。
	if strings.HasSuffix(parent, "/chat") {
		parent = path.Dir(parent)
	}
	if parent == "." || parent == "/" {
		parent = ""
	}
	return strings.TrimRight(baseURL, "/") + parent + "/models"
}

// refer: https://platform.openai.com/docs/api-reference/models/list
func fetchOpenAIModels(httpClient *http.Client, ctx context.Context, target model.ChannelConfig, key, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	for _, header := range target.CustomHeader {
		if header.HeaderKey != "" {
			req.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}

	response, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	result, err := decodeModelList[model.OpenAIModelList](response)
	if err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

// 对话端点探测的请求形态: OpenAI 的两个端点请求体不同, Chat 用 messages, Responses 用 input。
const (
	openAIStyleChat      = "openai-chat"
	openAIStyleResponses = "openai-responses"
)

// anyModelSucceeds 逐个模型尝试对话端点, 任一模型返回 2xx 即认为端点可用。
// 列表里混入非对话模型 (rerank / embedding) 时跳过它继续试; 连续 3 个都失败
// 基本可断定端点未实现或凭据无权访问, 不必扫完全列表。
func anyModelSucceeds(httpClient *http.Client, ctx context.Context, target model.ChannelConfig, key, url, style string, modelNames []string) bool {
	attempts := 0
	for _, name := range modelNames {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		if attempts >= 3 {
			return false
		}
		attempts++
		if probeDialogEndpoint(httpClient, ctx, target, key, url, style, name) {
			return true
		}
	}
	return false
}

// probeDialogEndpoint 对对话端点发一次最小真实请求, 验证它确实被上游实现。
// 大量聚合网关的 /v1/models 全量放行而部分对话端点并未实现, 列表接口的存在不能作为协议支持的证据。
// style 区分三种请求形态: OpenAI Chat 发 messages 体, OpenAI Responses 发 input 体, Anthropic 发 Messages 体;
// 返回 2xx 即视为端点可用; 4xx/5xx 说明路径不存在或协议未实现。
func probeDialogEndpoint(httpClient *http.Client, ctx context.Context, target model.ChannelConfig, key, url, style, modelName string) bool {
	body := map[string]any{"model": modelName}
	switch style {
	case openAIStyleChat:
		body["max_tokens"] = 1
		body["messages"] = []map[string]string{{"role": "user", "content": "ping"}}
	case openAIStyleResponses:
		body["input"] = "ping"
		body["max_output_tokens"] = 1
	default: // anthropic
		body["max_tokens"] = 1
		body["messages"] = []map[string]string{{"role": "user", "content": "ping"}}
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	if style == "anthropic" {
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	for _, header := range target.CustomHeader {
		if header.HeaderKey != "" {
			req.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 512))
	return response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
}

// refer: https://platform.claude.com/docs
func fetchAnthropicModels(httpClient *http.Client, ctx context.Context, target model.ChannelConfig, key, url string) ([]string, error) {
	var allModels []string
	var afterID string
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Api-Key", key)
		req.Header.Set("Anthropic-Version", "2023-06-01")
		for _, header := range target.CustomHeader {
			if header.HeaderKey != "" {
				req.Header.Set(header.HeaderKey, header.HeaderValue)
			}
		}
		if afterID != "" {
			q := req.URL.Query()
			q.Set("after_id", afterID)
			req.URL.RawQuery = q.Encode()
		}

		response, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		// 分页时每轮都会新建响应, 必须当轮读完即关; 用 defer 会攒到整个函数返回才释放。
		result, err := decodeModelList[model.AnthropicModelList](response)
		if err != nil {
			return nil, err
		}
		for _, m := range result.Data {
			allModels = append(allModels, m.ID)
		}
		if !result.HasMore {
			break
		}
		afterID = result.LastID
	}
	return allModels, nil
}

// decodeModelList 关闭响应并把响应体解成模型列表; 非 2xx 时按上游错误返回。
// 两侧解析流程一致, 只有目标结构不同, 故用类型参数收敛; 分页调用要求当轮读完即关, 关闭点放在此处最稳。
func decodeModelList[T any](response *http.Response) (T, error) {
	defer response.Body.Close()
	var result T
	// 上游报错时响应体常是能被正常解码的 JSON, 若不先拦下, 模型列表会解成空列表并当作成功;
	// 响应体截断到 512 字节: 部分上游在鉴权失败时返回整页 HTML, 全文带到界面上无用。
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, err := io.ReadAll(io.LimitReader(response.Body, 512))
		if err != nil {
			return result, fmt.Errorf("upstream %s", response.Status)
		}
		return result, fmt.Errorf("upstream %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return result, err
	}
	return result, nil
}
