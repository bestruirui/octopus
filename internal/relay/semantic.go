package relay

import (
	"strings"
	"unicode/utf8"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/looplj/axonhub/llm"
)

// 轻量语义/复杂度路由：规则版，不引入 embedding 模型。
// 开启后：复杂请求把 Weight 更高的渠道提前（假设用户给强模型更高 weight）。
//
// 设计原则：不新增 GroupMode，避免与轮询/加权/故障迁移重复造轮子。
// 只 soft-bias 候选顺序；Failover 的 Priority 主序由 ReorderByHealthScore 尊重。

// Complexity 请求复杂度
type Complexity int

const (
	ComplexitySimple Complexity = iota
	ComplexityMedium
	ComplexityComplex
)

// ClassifyComplexity 基于消息长度 + 关键词启发式分类
func ClassifyComplexity(req *llm.Request) Complexity {
	if req == nil {
		return ComplexityMedium
	}
	text := extractUserText(req)
	if text == "" {
		return ComplexityMedium
	}
	runes := utf8.RuneCountInString(text)
	lower := strings.ToLower(text)

	complexHints := []string{
		"写一个", "实现", "重构", "架构", "设计系统", "完整代码", "单元测试",
		"implement", "refactor", "architecture", "design a", "full code",
		"unit test", "debug", "fix bug", "算法", "algorithm", "leetcode",
		"优化性能", "optimize", "并发", "distributed", "分布式",
		"详细分析", "in depth", "step by step",
	}
	simpleHints := []string{
		"等于几", "是什么", "what is", "who is", "几点", "天气",
		"翻译", "translate", "hello", "你好", "hi", "1+1", "2+2",
		"首都", "capital of", "多少钱", "how much",
	}

	complexHits, simpleHits := 0, 0
	for _, h := range complexHints {
		if strings.Contains(lower, h) || strings.Contains(text, h) {
			complexHits++
		}
	}
	for _, h := range simpleHints {
		if strings.Contains(lower, h) || strings.Contains(text, h) {
			simpleHits++
		}
	}

	if runes > 800 || complexHits >= 2 {
		return ComplexityComplex
	}
	if runes > 200 || complexHits >= 1 {
		return ComplexityMedium
	}
	if runes < 40 || simpleHits >= 1 {
		return ComplexitySimple
	}
	return ComplexityMedium
}

func extractUserText(req *llm.Request) string {
	var b strings.Builder
	for _, m := range req.Messages {
		role := strings.ToLower(m.Role)
		if role == "user" || role == "human" {
			b.WriteString(messageContentToString(m))
			b.WriteByte(' ')
		}
	}
	return strings.TrimSpace(b.String())
}

func messageContentToString(m llm.Message) string {
	if m.Content.Content != nil {
		return *m.Content.Content
	}
	if len(m.Content.MultipleContent) > 0 {
		var b strings.Builder
		for _, p := range m.Content.MultipleContent {
			if p.Text != nil {
				b.WriteString(*p.Text)
				b.WriteByte(' ')
			}
		}
		return b.String()
	}
	return ""
}

// SemanticBiasCandidates 按复杂度 soft-bias 候选。
// complex: Weight 降序；simple/medium: 不改。
// 与 GroupModeWeighted 不重复：Weighted 是全局概率；这里按本次请求内容临时 bias。
func SemanticBiasCandidates(items []model.GroupItem, cx Complexity) []model.GroupItem {
	enabled, err := op.SettingGetBool(model.SettingKeySemanticRouteEnabled)
	if err != nil || !enabled || len(items) <= 1 {
		return items
	}
	if cx != ComplexityComplex {
		return items
	}
	out := make([]model.GroupItem, len(items))
	copy(out, items)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			wi, wj := out[i].Weight, out[j].Weight
			if wi <= 0 {
				wi = 1
			}
			if wj <= 0 {
				wj = 1
			}
			if wj > wi {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
