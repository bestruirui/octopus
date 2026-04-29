package semantic_cache

import (
	"fmt"
	"strings"

	transmodel "github.com/lingyuins/octopus/internal/transformer/model"
)

func BuildNamespace(apiKeyID int, endpointFamily, requestedModel string) string {
	return fmt.Sprintf("%d:%s:%s", apiKeyID, strings.TrimSpace(endpointFamily), strings.TrimSpace(requestedModel))
}

func ExtractNormalizedText(req *transmodel.InternalLLMRequest) (string, bool) {
	if req == nil || len(req.Messages) == 0 {
		return "", false
	}

	lines := make([]string, 0, len(req.Messages))
	for _, msg := range req.Messages {
		text := normalizeSemanticCacheText(extractMessageText(msg.Content))
		if text == "" {
			continue
		}

		role := normalizeSemanticCacheText(msg.Role)
		if role == "" {
			role = "message"
		}
		lines = append(lines, role+": "+text)
	}

	joined := strings.TrimSpace(strings.Join(lines, "\n"))
	if joined == "" {
		return "", false
	}

	return joined, true
}

func extractMessageText(content transmodel.MessageContent) string {
	textParts := make([]string, 0, 1+len(content.MultipleContent))
	if content.Content != nil {
		textParts = append(textParts, *content.Content)
	}
	for _, part := range content.MultipleContent {
		if !strings.EqualFold(strings.TrimSpace(part.Type), "text") || part.Text == nil {
			continue
		}
		textParts = append(textParts, *part.Text)
	}
	return strings.Join(textParts, " ")
}

func normalizeSemanticCacheText(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}
