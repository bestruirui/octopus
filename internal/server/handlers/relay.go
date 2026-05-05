package handlers

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/relay"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/bestruirui/octopus/internal/transformer/inbound"
	"github.com/gin-gonic/gin"
)

const (
	responsesEndpointContextKey = relay.ResponsesEndpointContextKey
	responsesEndpointCompact    = relay.ResponsesEndpointCompact
)

func init() {
	router.NewGroupRouter("/v1").
		Use(middleware.APIKeyAuth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/chat/completions", http.MethodPost).
				Handle(chat),
		).
		AddRoute(
			router.NewRoute("/responses", http.MethodPost).
				Handle(response),
		).
		AddRoute(
			router.NewRoute("/responses/compact", http.MethodPost).
				Handle(responseCompact),
		).
		AddRoute(
			router.NewRoute("/messages", http.MethodPost).
				Handle(message),
		).
		AddRoute(
			router.NewRoute("/embeddings", http.MethodPost).
				Handle(embedding),
		)
}

func chat(c *gin.Context) {
	relay.Handler(inbound.InboundTypeOpenAIChat, c)
}
func response(c *gin.Context) {
	relay.Handler(inbound.InboundTypeOpenAIResponse, c)
}

// responseCompact handles /v1/responses/compact by marking compact endpoint
// context before delegating to the shared relay pipeline.
func responseCompact(c *gin.Context) {
	c.Set(responsesEndpointContextKey, responsesEndpointCompact)
	relay.Handler(inbound.InboundTypeOpenAIResponse, c)
}
func message(c *gin.Context) {
	relay.Handler(inbound.InboundTypeAnthropic, c)
}
func embedding(c *gin.Context) {
	relay.Handler(inbound.InboundTypeOpenAIEmbedding, c)
}
