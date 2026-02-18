package openai

import "github.com/bestruirui/octopus/internal/transformer/model"

// convertChatMessagesToResponsesInput is the dedicated conversion-layer entry for:
// OpenAI Chat-style internal messages -> OpenAI Responses input.
func convertChatMessagesToResponsesInput(msgs []model.Message, transformOptions model.TransformOptions) ResponsesInput {
	return convertInputFromMessages(msgs, transformOptions)
}
