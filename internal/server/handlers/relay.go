package handlers

import (
	"net/http"

	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/relay"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/router"
	"github.com/lingyuins/octopus/internal/transformer/inbound"
	"github.com/gin-gonic/gin"
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
			router.NewRoute("/messages", http.MethodPost).
				Handle(message),
		).
		AddRoute(
			router.NewRoute("/embeddings", http.MethodPost).
				Handle(embedding),
		)
}

func chat(c *gin.Context) {
	relay.Handler(model.EndpointTypeChat, inbound.InboundTypeOpenAIChat, c)
}
func response(c *gin.Context) {
	relay.Handler(model.EndpointTypeResponses, inbound.InboundTypeOpenAIResponse, c)
}
func message(c *gin.Context) {
	relay.Handler(model.EndpointTypeMessages, inbound.InboundTypeAnthropic, c)
}
func embedding(c *gin.Context) {
	relay.Handler(model.EndpointTypeEmbeddings, inbound.InboundTypeOpenAIEmbedding, c)
}
