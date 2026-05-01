package openai

import (
	"encoding/json"
	"strings"

	"github.com/lingyuins/octopus/internal/transformer/model"
)

func isDeepSeekCompatRequest(baseURL string, request *model.InternalLLMRequest) bool {
	lowerBaseURL := strings.ToLower(strings.TrimSpace(baseURL))
	if lowerBaseURL != "" && strings.Contains(lowerBaseURL, "deepseek") {
		return true
	}
	if request == nil {
		return false
	}

	if strings.EqualFold(
		strings.TrimSpace(request.TransformerMetadata[model.TransformerMetadataGroupEndpointType]),
		"deepseek",
	) {
		return true
	}

	lowerModelName := strings.ToLower(strings.TrimSpace(request.Model))
	return strings.Contains(lowerModelName, "deepseek")
}

func normalizeDeepSeekReasoningCompat(request *model.InternalLLMRequest, baseURL string) {
	if request == nil || !isDeepSeekCompatRequest(baseURL, request) {
		return
	}

	thinkingType, hasThinkingType := extractDeepSeekThinkingType(request.ExtraBody)
	normalizedEffort := normalizeDeepSeekReasoningEffort(request.ReasoningEffort)

	if hasThinkingType && thinkingType == "disabled" {
		request.ReasoningEffort = ""
	} else {
		request.ReasoningEffort = normalizedEffort
	}

	switch {
	case hasThinkingType && (thinkingType == "enabled" || thinkingType == "disabled"):
		request.ExtraBody = mergeDeepSeekThinkingExtraBody(request.ExtraBody, thinkingType)
	case request.ReasoningEffort == "":
		request.ExtraBody = mergeDeepSeekThinkingExtraBody(request.ExtraBody, "disabled")
	}
}

func normalizeDeepSeekReasoningEffort(effort string) string {
	switch strings.ToLower(strings.TrimSpace(effort)) {
	case "", "none":
		return ""
	case "low", "medium", "high":
		return "high"
	case "xhigh", "max":
		return "max"
	default:
		return strings.TrimSpace(effort)
	}
}

func extractDeepSeekThinkingType(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", false
	}

	thinkingValue, ok := payload["thinking"]
	if !ok {
		return "", false
	}

	thinking, ok := thinkingValue.(map[string]any)
	if !ok {
		return "", false
	}

	typeValue, ok := thinking["type"]
	if !ok {
		return "", false
	}

	typeString, ok := typeValue.(string)
	if !ok {
		return "", false
	}

	normalized := strings.ToLower(strings.TrimSpace(typeString))
	return normalized, normalized != ""
}

func mergeDeepSeekThinkingExtraBody(raw json.RawMessage, thinkingType string) json.RawMessage {
	payload := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}

	payload["thinking"] = map[string]any{
		"type": thinkingType,
	}

	merged, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return merged
}
