package openai

import "strings"

func normalizeOpenAICompatReasoningEffort(effort string) string {
	normalized := strings.ToLower(strings.TrimSpace(effort))

	switch normalized {
	case "", "none":
		return ""
	case "low", "medium", "high":
		return normalized
	default:
		return strings.TrimSpace(effort)
	}
}
