package condition

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// ConditionRule represents a single routing condition.
type ConditionRule struct {
	Key   string `json:"key"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

// RequestContext provides request metadata for condition evaluation.
type RequestContext struct {
	Model       string
	APIKeyID    int
	APIKeyName  string
	Headers     map[string]string
	ContentType string
	Hour        int // 0-23 UTC
}

// Evaluate returns true if all rules pass (AND logic). An empty condition always passes.
func Evaluate(conditionJSON string, ctx RequestContext) (bool, error) {
	if strings.TrimSpace(conditionJSON) == "" {
		return true, nil
	}
	var rules []ConditionRule
	if err := json.Unmarshal([]byte(conditionJSON), &rules); err != nil {
		return false, err
	}
	if len(rules) == 0 {
		return true, nil
	}
	for _, rule := range rules {
		if !evaluateRule(rule, ctx) {
			return false, nil
		}
	}
	return true, nil
}

func evaluateRule(rule ConditionRule, ctx RequestContext) bool {
	actual := resolveValue(rule.Key, ctx)
	switch rule.Op {
	case "equals":
		return strings.EqualFold(actual, rule.Value)
	case "not_equals":
		return !strings.EqualFold(actual, rule.Value)
	case "contains":
		return strings.Contains(strings.ToLower(actual), strings.ToLower(rule.Value))
	case "not_contains":
		return !strings.Contains(strings.ToLower(actual), strings.ToLower(rule.Value))
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(actual), strings.ToLower(rule.Value))
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(actual), strings.ToLower(rule.Value))
	case "regex":
		re, err := regexp.Compile(rule.Value)
		if err != nil {
			return false
		}
		return re.MatchString(actual)
	case "in_list":
		return inList(actual, rule.Value)
	case "gt":
		return compareInt(actual, rule.Value) > 0
	case "lt":
		return compareInt(actual, rule.Value) < 0
	default:
		return false
	}
}

func resolveValue(key string, ctx RequestContext) string {
	switch {
	case key == "model":
		return ctx.Model
	case key == "api_key_id":
		return strconv.Itoa(ctx.APIKeyID)
	case key == "api_key_name":
		return ctx.APIKeyName
	case key == "content_type":
		return ctx.ContentType
	case key == "hour":
		return strconv.Itoa(ctx.Hour)
	case strings.HasPrefix(key, "header_"):
		headerName := strings.TrimPrefix(key, "header_")
		if v, ok := ctx.Headers[strings.ToLower(headerName)]; ok {
			return v
		}
		return ""
	default:
		return ""
	}
}

func inList(value, csv string) bool {
	for _, item := range strings.Split(csv, ",") {
		if strings.EqualFold(strings.TrimSpace(item), value) {
			return true
		}
	}
	return false
}

func compareInt(a, b string) int {
	av, _ := strconv.Atoi(a)
	bv, _ := strconv.Atoi(b)
	if av > bv {
		return 1
	}
	if av < bv {
		return -1
	}
	return 0
}
