package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/dop251/goja"
)

// 余额查询脚本 URL 的模板变量。
const (
	// balancePlaceholderBaseURL 是网关根地址(渠道 base_url 去掉尾部 API 版本路径,如 /v1)。
	balancePlaceholderBaseURL = "{{baseUrl}}"
	// balancePlaceholderAPIKey 是渠道 Key。
	balancePlaceholderAPIKey = "{{apiKey}}"
)

// balanceScript 是脚本求值后的结构:request 描述查询请求,extractor 解析响应。
type balanceScript struct {
	URL        string
	Method     string
	Headers    map[string]string
	Extractor  goja.Callable
	ExtractorVM *goja.Runtime
}

// deepSeekBalanceScript 是 DeepSeek 预设类型的查询脚本:
// 调用官方 /user/balance 接口,聚合 balance_infos 中的各币种余额。
// DeepSeek 官方没有总额度概念,只返回余额,因此不返回 total。
const deepSeekBalanceScript = `({
  request: {
    url: "{{baseUrl}}/user/balance",
    method: "GET",
    headers: {
      "Accept": "application/json",
      "Authorization": "Bearer {{apiKey}}"
    }
  },
  extractor: function (response) {
    if (response.is_available === false) {
      return { isValid: false, invalidMessage: response.message || "余额不可用" };
    }
    var infos = response.balance_infos || [];
    if (infos.length === 0) {
      return { isValid: false, invalidMessage: "无余额信息" };
    }
    var remaining = 0, units = [];
    infos.forEach(function (info) {
      remaining += Number(info.total_balance) || 0;
      if (info.currency) units.push(info.currency);
    });
    return {
      isValid: true,
      remaining: remaining,
      unit: units[0] || ""
    };
  }
})`

// FetchBalance 执行渠道配置的余额查询脚本,返回余额快照。
// 查询失败时返回带 Error 字段的 Balance,由调用方决定是否落库展示。
func FetchBalance(ctx context.Context, request model.Channel) (*model.Balance, error) {
	if request.BalanceQuery == nil || !request.BalanceQuery.Enabled {
		return nil, fmt.Errorf("balance query not configured")
	}

	// 单次查询超时,默认 10 秒。
	timeout := request.BalanceQuery.Timeout
	if timeout <= 0 {
		timeout = 10
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 预设类型使用内置脚本,与自定义脚本走同一条执行链路。
	scriptText := request.BalanceQuery.Script
	switch request.BalanceQuery.Type {
	case model.BalanceQueryTypeDeepSeek:
		scriptText = deepSeekBalanceScript
	}
	if strings.TrimSpace(scriptText) == "" {
		return nil, fmt.Errorf("balance query script is required")
	}

	script, err := compileBalanceScript(scriptText, &request)
	if err != nil {
		return nil, err
	}

	body, err := doBalanceRequest(ctx, &request, script)
	if err != nil {
		return nil, err
	}

	balance, err := runExtractor(script, body)
	if err != nil {
		return nil, err
	}
	balance.UpdatedAt = time.Now().Unix()
	return balance, nil
}

// compileBalanceScript 用 goja 执行脚本得到 request 配置与 extractor 函数。
// 脚本形如 ({request: {url, method?, headers?}, extractor: function (response) {...}})。
func compileBalanceScript(scriptText string, channel *model.Channel) (*balanceScript, error) {
	vm := goja.New()
	// 注入环境变量,提取函数内可直接使用 baseUrl / apiKey。
	gatewayBase := strings.TrimSuffix(strings.TrimRight(channel.GetBaseUrl(), "/"), "/v1")
	if err := vm.Set("baseUrl", gatewayBase); err != nil {
		return nil, fmt.Errorf("failed to set baseUrl env: %w", err)
	}
	if err := vm.Set("apiKey", channel.GetChannelKey().ChannelKey); err != nil {
		return nil, fmt.Errorf("failed to set apiKey env: %w", err)
	}
	value, err := vm.RunString(scriptText)
	if err != nil {
		return nil, fmt.Errorf("failed to compile balance script: %w", err)
	}
	obj := value.ToObject(vm)
	if obj == nil {
		return nil, fmt.Errorf("balance script must evaluate to an object")
	}

	requestObj := obj.Get("request").ToObject(vm)
	if requestObj == nil {
		return nil, fmt.Errorf("balance script missing request config")
	}
	url := requestObj.Get("url").String()
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("balance script request url is required")
	}
	method := requestObj.Get("method").String()
	if strings.TrimSpace(method) == "" {
		method = http.MethodGet
	}
	headers := make(map[string]string)
	if headersObj := requestObj.Get("headers").ToObject(vm); headersObj != nil {
		for _, key := range headersObj.Keys() {
			if key != "" {
				headers[key] = headersObj.Get(key).String()
			}
		}
	}

	extractor, ok := goja.AssertFunction(obj.Get("extractor"))
	if !ok {
		return nil, fmt.Errorf("balance script extractor is not a function")
	}

	return &balanceScript{
		URL:         resolveTemplateVars(url, channel),
		Method:      strings.ToUpper(method),
		Headers:     headers,
		Extractor:   extractor,
		ExtractorVM: vm,
	}, nil
}

// doBalanceRequest 按脚本的 request 配置发出 HTTP 请求并返回响应体。
func doBalanceRequest(ctx context.Context, channel *model.Channel, script *balanceScript) ([]byte, error) {
	client, err := ChannelHttpClient(channel)
	if err != nil {
		return nil, fmt.Errorf("failed to get http client: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, script.Method, script.URL, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range script.Headers {
		req.Header.Set(key, resolveTemplateVars(value, channel))
	}
	applyCustomHeaders(req, *channel)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("balance request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read balance response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("balance request returned status %d: %s", resp.StatusCode, truncateBody(body))
	}
	return body, nil
}

// resolveTemplateVars 展开脚本 URL/Header 中的模板变量。
// {{baseUrl}} 是网关根地址(渠道 base_url 去掉尾部 API 版本路径,如 /v1);{{apiKey}} 是渠道 Key。
func resolveTemplateVars(text string, channel *model.Channel) string {
	gatewayBase := strings.TrimSuffix(strings.TrimRight(channel.GetBaseUrl(), "/"), "/v1")
	text = strings.ReplaceAll(text, balancePlaceholderBaseURL, gatewayBase)
	text = strings.ReplaceAll(text, balancePlaceholderAPIKey, channel.GetChannelKey().ChannelKey)
	return text
}

// runExtractor 调用脚本的 extractor 函数解析响应体。
// 提取函数签名为 function (response) {...},response 是响应 JSON 解析后的对象,
// 成功时通常不返回 isValid,失败时返回 {isValid: false, invalidMessage: "..."}。
func runExtractor(script *balanceScript, responseBody []byte) (*model.Balance, error) {
	var responseJSON any
	if err := json.Unmarshal(responseBody, &responseJSON); err != nil {
		return nil, fmt.Errorf("balance response is not valid json: %s", truncateBody(responseBody))
	}

	vm := script.ExtractorVM
	resultValue, err := script.Extractor(goja.Undefined(), vm.ToValue(responseJSON))
	if err != nil {
		return nil, fmt.Errorf("extractor execution failed: %w", err)
	}
	result := resultValue.ToObject(vm)
	if result == nil {
		return nil, fmt.Errorf("extractor returned no result")
	}

	out := &model.Balance{}
	isValid := true // 提取函数约定:成功时通常不返回 isValid。
	if v := result.Get("isValid"); v != nil {
		isValid = v.ToBoolean()
	}
	if !isValid {
		message := ""
		if v := result.Get("invalidMessage"); v != nil {
			message = v.String()
		}
		if message == "" || message == "undefined" {
			message = "balance query failed"
		}
		return nil, fmt.Errorf("%s", message)
	}
	if v := result.Get("planName"); v != nil {
		out.PlanName = v.String()
	}
	if v := result.Get("extra"); v != nil {
		out.Extra = v.String()
	}
	if v := result.Get("remaining"); v != nil {
		out.Remaining = v.ToFloat()
	}
	if v := result.Get("used"); v != nil {
		out.Used = v.ToFloat()
	}
	if v := result.Get("total"); v != nil {
		out.Total = v.ToFloat()
	}
	if v := result.Get("unit"); v != nil {
		out.Unit = v.String()
	}
	return out, nil
}

// truncateBody 截断响应体用于错误提示,避免超长错误消息。
func truncateBody(body []byte) string {
	const maxLen = 256
	if len(body) <= maxLen {
		return string(body)
	}
	return string(body[:maxLen]) + "..."
}
