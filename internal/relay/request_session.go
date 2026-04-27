package relay

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/gin-gonic/gin"
	transmodel "github.com/lingyuins/octopus/internal/transformer/model"
)

func populateRelayRequestSessionFields(c *gin.Context, req *transmodel.InternalLLMRequest, body []byte) {
	if req == nil {
		return
	}

	req.RawRequest = append([]byte(nil), body...)
	req.ConversationID = strings.TrimSpace(c.GetHeader("X-Conversation-ID"))
	req.ResumeFromEventID = parseRelayEventSequence(c.GetHeader("Last-Event-ID"))

	if req.ConversationID == "" {
		req.ConversationID = strings.TrimSpace(c.Query("conversation_id"))
	}
	if req.ResumeFromEventID == 0 {
		req.ResumeFromEventID = parseRelayEventSequence(c.Query("last_event_id"))
	}
	if req.ResumeFromEventID == 0 {
		req.ResumeFromEventID = parseRelayEventSequence(c.Query("resume_from_sequence"))
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return
	}

	if req.ConversationID == "" {
		req.ConversationID = parseRelayRawStringField(raw, "conversation_id")
	}
	if req.ResumeFromEventID == 0 {
		req.ResumeFromEventID = parseRelayRawIntField(raw, "last_event_id")
	}
	if req.ResumeFromEventID == 0 {
		req.ResumeFromEventID = parseRelayRawIntField(raw, "resume_from_sequence")
	}
}

func parseRelayRawStringField(raw map[string]json.RawMessage, field string) string {
	value, ok := raw[field]
	if !ok || len(value) == 0 {
		return ""
	}

	var s string
	if err := json.Unmarshal(value, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return ""
}

func parseRelayRawIntField(raw map[string]json.RawMessage, field string) int64 {
	value, ok := raw[field]
	if !ok || len(value) == 0 {
		return 0
	}

	var n int64
	if err := json.Unmarshal(value, &n); err == nil && n > 0 {
		return n
	}

	var s string
	if err := json.Unmarshal(value, &s); err == nil {
		return parseRelayEventSequence(s)
	}
	return 0
}

func parseRelayEventSequence(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}

	n, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func shouldUseRelayStreamSession(req *transmodel.InternalLLMRequest) bool {
	if req == nil || req.Stream == nil || !*req.Stream {
		return false
	}
	return strings.TrimSpace(req.ConversationID) != ""
}

func buildRelayStreamSessionHash(endpointType string, inboundType int, apiKeyID int, rawRequest []byte) uint64 {
	normalizedRequest := normalizeRelayRequestHashBody(rawRequest)
	hasher := xxhash.New()
	_, _ = hasher.WriteString(strings.TrimSpace(endpointType))
	_, _ = hasher.WriteString("\n")
	_, _ = hasher.WriteString(strconv.Itoa(inboundType))
	_, _ = hasher.WriteString("\n")
	_, _ = hasher.WriteString(strconv.Itoa(apiKeyID))
	_, _ = hasher.WriteString("\n")
	_, _ = hasher.Write(normalizedRequest)
	return hasher.Sum64()
}

func normalizeRelayRequestHashBody(rawRequest []byte) []byte {
	if len(rawRequest) == 0 {
		return nil
	}

	var payload any
	if err := json.Unmarshal(rawRequest, &payload); err != nil {
		return rawRequest
	}

	root, ok := payload.(map[string]any)
	if !ok {
		return rawRequest
	}

	delete(root, "conversation_id")
	delete(root, "last_event_id")
	delete(root, "resume_from_sequence")

	normalized, err := json.Marshal(root)
	if err != nil {
		return rawRequest
	}
	return normalized
}
