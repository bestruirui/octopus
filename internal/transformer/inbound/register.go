package inbound

import (
	"github.com/bestruirui/octopus/internal/transformer/inbound/anthropic"
	"github.com/bestruirui/octopus/internal/transformer/inbound/openai"
	"github.com/bestruirui/octopus/internal/transformer/model"
)

type InboundType int

const (
	InboundTypeOpenAIChat InboundType = iota
	InboundTypeOpenAIResponse
	InboundTypeAnthropic
	InboundTypeGemini
	InboundTypeOpenAIEmbedding

	// Compatibility alias for legacy naming
	InboundTypeOpenAI = InboundTypeOpenAIChat
)

var inboundFactories = map[InboundType]func() model.Inbound{
	InboundTypeOpenAIChat:      func() model.Inbound { return &openai.ChatInbound{} },
	InboundTypeOpenAIResponse:  func() model.Inbound { return &openai.ResponseInbound{} },
	InboundTypeOpenAIEmbedding: func() model.Inbound { return &openai.EmbeddingInbound{} },
	InboundTypeAnthropic:       func() model.Inbound { return &anthropic.MessagesInbound{} },
}

func Get(inboundType InboundType) model.Inbound {
	if factory, ok := inboundFactories[inboundType]; ok {
		return factory()
	}
	return nil
}
