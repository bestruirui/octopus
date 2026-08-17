package handlers

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/relay"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/v1").
		Use(middleware.APIKeyAuth()).
		AddRoute(
			router.NewRoute("/chat/completions", http.MethodPost).
			Handle(relay.HandleChatCompletions),
		).
		AddRoute(
			router.NewRoute("/responses", http.MethodPost).
			Handle(relay.HandleResponses),
		).
		AddRoute(
			router.NewRoute("/messages", http.MethodPost).
			Handle(relay.HandleMessages),
		).
		AddRoute(
			router.NewRoute("/images/generations", http.MethodPost).
				Handle(relay.HandleImageGenerations),
		).
		AddRoute(
			router.NewRoute("/images/edits", http.MethodPost).
				Handle(relay.HandleImageEdits),
		).
		AddRoute(
			router.NewRoute("/images/variations", http.MethodPost).
				Handle(relay.HandleImageVariations),
		)
}
