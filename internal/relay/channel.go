package relay

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm/auth"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/anthropic"
	"github.com/looplj/axonhub/llm/transformer/openai"
	"github.com/looplj/axonhub/llm/transformer/openai/responses"
	"github.com/tidwall/sjson"
)

// buildOutbound 在渠道授权支持的协议内选出本轮上游协议, 构造对应的出站转换器, 并返回选中的协议和能否同协议透传。
// 地址由渠道的协议路径字段与地址拼接, 凭据取自目标绑定的渠道凭据。
// want 是客户端请求使用的协议, 由调用方按入站格式定出; 选中的协议随请求状态推给界面, 故一并返回。
func buildOutbound(channel model.Channel, grant model.ChannelGrant, channelKey model.ChannelKey, want model.Protocol) (transformer.Outbound, model.Protocol, bool, error) {
	protocol, passthrough := want, grant.Protocols&want != 0
	if !passthrough {
		protocol = 0
		switch {
		case grant.Protocols&model.ProtocolAnthropicMessage != 0:
			protocol = model.ProtocolAnthropicMessage
		case grant.Protocols&model.ProtocolOpenAIResponse != 0:
			protocol = model.ProtocolOpenAIResponse
		case grant.Protocols&model.ProtocolOpenAIChatCompletion != 0:
			protocol = model.ProtocolOpenAIChatCompletion
		}
	}

	key := auth.NewStaticKeyProvider(channelKey.Key)
	switch protocol {
	case model.ProtocolOpenAIChatCompletion:
		outbound, err := openai.NewOutboundTransformerWithConfig(&openai.Config{PlatformType: openai.PlatformOpenAI, BaseURL: channel.BaseURL, EndpointPath: channel.OpenAIChatCompletionPath, APIKeyProvider: key})
		return outbound, protocol, passthrough, err
	case model.ProtocolOpenAIResponse:
		outbound, err := responses.NewOutboundTransformerWithConfig(&responses.Config{BaseURL: channel.BaseURL, EndpointPath: channel.OpenAIResponsePath, APIKeyProvider: key})
		return outbound, protocol, passthrough, err
	case model.ProtocolAnthropicMessage:
		outbound, err := anthropic.NewOutboundTransformerWithConfig(&anthropic.Config{Type: anthropic.PlatformDirect, BaseURL: channel.BaseURL, EndpointPath: channel.AnthropicMessagePath, APIKeyProvider: key})
		return outbound, protocol, passthrough, err
	default:
		return nil, 0, false, fmt.Errorf("channel grant %d supports no known protocol: %d", grant.ID, grant.Protocols)
	}
}

// clientHeaderPlaceholder 匹配自定义 Header 值中引用的客户端请求头
var clientHeaderPlaceholder = regexp.MustCompile(`\{client_header:[^}]+\}`)

// applyChannelConfig 按渠道配置覆盖上游请求的参数并追加自定义 Header; model 与 stream 由转发流程决定, 不允许覆盖。
// oc 为条件与值模板提供本轮请求上下文, 为 nil 时所有带条件的配置一律不生效。
// 参数覆盖支持两种格式, 以首个非空白字符区分: {...} 为旧的平铺对象(顶层键), [...] 为操作数组(支持条件)。
func applyChannelConfig(channel model.Channel, oc *overrideContext, request *httpclient.Request) error {
	if trimmed := strings.TrimSpace(channel.ParamOverride); trimmed != "" {
		var body []byte
		switch trimmed[0] {
		case '[':
			var ops []model.OverrideOperation
			if err := json.Unmarshal([]byte(channel.ParamOverride), &ops); err != nil {
				return fmt.Errorf("invalid channel parameter override: %w", err)
			}
			body = applyOverrideOperations(request.Body, ops, oc)
		default:
			var overrides map[string]json.RawMessage
			if err := json.Unmarshal([]byte(channel.ParamOverride), &overrides); err != nil {
				return fmt.Errorf("invalid channel parameter override: %w", err)
			}
			body = request.Body
			// 覆盖键可能自带点号或冒号, 转义后再作为 sjson 路径使用, 避免被解析成嵌套路径。
			escape := strings.NewReplacer("\\", "\\\\", ".", "\\.", ":", "\\:")
			for key, value := range overrides {
				if key == "model" || key == "stream" {
					continue
				}
				next, err := sjson.SetRawBytes(body, ":"+escape.Replace(key), value)
				if err != nil {
					return fmt.Errorf("apply channel parameter %q: %w", key, err)
				}
				body = next
			}
		}
		request.Body = body
		if len(request.JSONBody) > 0 {
			request.JSONBody = slices.Clone(body)
		}
	}

	// 自定义 Header 按配置行序执行: set 覆盖, delete 删除, rename 改名(保留全部值), copy 复制。
	// 转换器已写入的认证等敏感 Header 不允许被任何操作改动; 值支持模板与 {client_header:xxx} 占位。
	for _, header := range channel.CustomHeader {
		if !evaluateOverrideCondition(header.Condition, oc) {
			continue
		}
		if header.HeaderKey == "" {
			continue
		}
		op := header.Op
		if op == "" {
			op = model.OverrideOpSet
		}
		target := strings.TrimSpace(header.HeaderValue)
		switch op {
		case model.OverrideOpSet:
			if guardedHeader(request, header.HeaderKey) {
				continue
			}
			request.Headers.Set(header.HeaderKey, renderHeaderValue(header.HeaderValue, request, oc))
		case model.OverrideOpDelete:
			if guardedHeader(request, header.HeaderKey) {
				continue
			}
			request.Headers.Del(header.HeaderKey)
		case model.OverrideOpRename, model.OverrideOpCopy:
			// rename/copy 面向已有头, 目标名缺失时跳过; 源头不存在则无事可做。
			if target == "" || guardedHeader(request, header.HeaderKey) || guardedHeader(request, target) {
				continue
			}
			values := request.Headers.Values(header.HeaderKey)
			if len(values) == 0 {
				continue
			}
			if op == model.OverrideOpRename {
				request.Headers.Del(header.HeaderKey)
			}
			for _, value := range values {
				request.Headers.Add(target, value)
			}
		}
	}
	return nil
}

// guardedHeader 判断敏感 Header 是否已被转换器写入: 已写入的不允许被自定义配置改动或删除。
func guardedHeader(request *httpclient.Request, key string) bool {
	return key != "" && request.Headers.Get(key) != "" && httpclient.IsSensitiveHeader(key)
}

// renderHeaderValue 渲染 Header 值: 先按 Go template 渲染 {{...}}, 再把 {client_header:xxx} 片段替换为客户端请求头。
func renderHeaderValue(value string, request *httpclient.Request, oc *overrideContext) string {
	rendered := renderOverrideValue(value, oc)
	return clientHeaderPlaceholder.ReplaceAllStringFunc(rendered, func(placeholder string) string {
		return request.Headers.Get(placeholder[len("{client_header:") : len(placeholder)-1])
	})
}
