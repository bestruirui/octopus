package relay

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"text/template"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/charmbracelet/log"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// overrideContext 是参数覆盖与自定义 Header 的条件及值模板渲染时可见的请求上下文。
// 字段名即模板变量名: {{eq .Model "gpt-4o"}}, {{index .RequestHeader "x-trace-id"}} 等。
type overrideContext struct {
	RequestModel    string            // 客户端请求的模型名, 即分组名。
	Model           string            // 模型映射后发给上游的模型名。
	ReasoningEffort string            // 入站转换器解析出的思考等级。
	RequestHeader   map[string]string // 客户端请求头; canonical 与小写双键, 敏感头剔除。
	Metadata        map[string]any    // 请求体 metadata 字段的内容, 无则nil。
}

// buildOverrideContext 从本轮转发已知的信息组装渲染上下文; 每轮渠道与模型都可能不同, 故逐轮构建。
func buildOverrideContext(requestModel, modelName, reasoningEffort string, headers http.Header, body []byte) *overrideContext {
	oc := &overrideContext{
		RequestModel:    requestModel,
		Model:           modelName,
		ReasoningEffort: reasoningEffort,
		RequestHeader:   map[string]string{},
	}
	for key, values := range headers {
		if httpclient.IsSensitiveHeader(key) || len(values) == 0 {
			continue
		}
		// canonical 与小写双键: 模板里写 "X-Trace-Id" 与 "x-trace-id" 都能取到。
		oc.RequestHeader[http.CanonicalHeaderKey(key)] = values[0]
		oc.RequestHeader[strings.ToLower(key)] = values[0]
	}
	if metadata := gjson.GetBytes(body, "metadata"); metadata.IsObject() {
		oc.Metadata, _ = metadata.Value().(map[string]any)
	}
	return oc
}

// evaluateOverrideCondition 渲染条件模板, 去空白后等于 "true" 才为真; 空条件恒真。
// 渲染失败视为不满足并告警, 让单条配置的语法错误不至于中断整个转发; oc 为 nil 说明当前调用
// 没有请求上下文(如模型探测), 带条件的一律不生效。
func evaluateOverrideCondition(condition string, oc *overrideContext) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true
	}
	if oc == nil {
		return false
	}
	rendered, err := renderOverrideTemplate(condition, oc)
	if err != nil {
		log.Warnf("render override condition %q: %v", condition, err)
		return false
	}
	return strings.TrimSpace(rendered) == "true"
}

// renderOverrideValue 渲染值模板; 不含 {{ 视为静态值, 渲染失败回退原值并告警, 与条件同一容错口径。
func renderOverrideValue(value string, oc *overrideContext) string {
	if !strings.Contains(value, "{{") {
		return value
	}
	rendered, err := renderOverrideTemplate(value, oc)
	if err != nil {
		log.Warnf("render override value %q: %v", value, err)
		return value
	}
	return rendered
}

func renderOverrideTemplate(text string, oc *overrideContext) (string, error) {
	tmpl, err := template.New("override").Funcs(model.OverrideTemplateFuncs).Parse(text)
	if err != nil {
		return "", err
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, oc); err != nil {
		return "", err
	}
	return rendered.String(), nil
}

// applyOverrideOperations 按数组顺序执行参数覆盖操作; model 与 stream 由转发流程决定, 静默跳过。
// 单条操作的失败不影响其余操作: 覆盖是尽力而为的配置修补, 不应让整轮转发失败。
func applyOverrideOperations(body []byte, ops []model.OverrideOperation, oc *overrideContext) []byte {
	for _, op := range ops {
		if op.Path == "" || op.Path == "model" || op.Path == "stream" {
			continue
		}
		if !evaluateOverrideCondition(op.Condition, oc) {
			continue
		}
		switch op.Op {
		case model.OverrideOpSet:
			if op.Value == "" {
				continue
			}
			body = setOverrideValue(body, op.Path, renderOverrideValue(op.Value, oc))
		case model.OverrideOpSetIfAbsent:
			if op.Value == "" || gjson.GetBytes(body, op.Path).Exists() {
				continue
			}
			body = setOverrideValue(body, op.Path, renderOverrideValue(op.Value, oc))
		case model.OverrideOpDelete:
			if next, err := sjson.DeleteBytes(body, op.Path); err == nil {
				body = next
			}
		}
	}
	return body
}

// setOverrideValue 把渲染后的值写入路径: 结果是合法 JSON 时按结构化值写入, 否则按字符串写入;
// 这样 "0.7" 会写成数字 0.7, "premium" 写成字符串 "premium"。两次写入都失败只保留原 body。
func setOverrideValue(body []byte, path, value string) []byte {
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(value), &raw); err == nil {
		if next, err := sjson.SetRawBytes(body, path, raw); err == nil {
			return next
		}
	}
	next, err := sjson.SetBytes(body, path, value)
	if err != nil {
		log.Warnf("set override path %q: %v", path, err)
		return body
	}
	return next
}
