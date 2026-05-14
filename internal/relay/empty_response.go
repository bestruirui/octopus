package relay

import transformerModel "github.com/bestruirui/octopus/internal/transformer/model"

// hasStreamResponseContent 判断统一流式响应中是否已经包含对客户端有意义的内容。
// 注意：仅 role、空 choices、usage、finish_reason 等元数据不算有效回复。
func hasStreamResponseContent(resp *transformerModel.InternalLLMResponse) bool {
	if resp == nil {
		return false
	}

	if len(resp.EmbeddingData) > 0 {
		return true
	}

	for _, choice := range resp.Choices {
		if hasMessageContent(choice.Message) || hasMessageContent(choice.Delta) {
			return true
		}
	}

	return false
}

// hasMessageContent 判断消息体是否包含可交付给客户端的有效内容。
func hasMessageContent(msg *transformerModel.Message) bool {
	if msg == nil {
		return false
	}

	if msg.Content.Content != nil && *msg.Content.Content != "" {
		return true
	}
	if len(msg.Content.MultipleContent) > 0 {
		return true
	}
	if len(msg.ToolCalls) > 0 {
		return true
	}
	if len(msg.Images) > 0 {
		return true
	}
	if msg.GetReasoningContent() != "" {
		return true
	}
	if msg.Refusal != "" {
		return true
	}
	if msg.Audio != nil && (msg.Audio.Data != "" || msg.Audio.ID != "" || msg.Audio.Transcript != "") {
		return true
	}

	return false
}
