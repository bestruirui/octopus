package handlers

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/relay"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/v1/images").
		Use(middleware.APIKeyAuth()).
		AddRoute(
			router.NewRoute("/generations", http.MethodPost).
				Handle(generations),
		).
		AddRoute(
			router.NewRoute("/edits", http.MethodPost).
				Handle(edits),
		).
		AddRoute(
			router.NewRoute("/variations", http.MethodPost).
				Handle(variations),
		)
}

func generations(c *gin.Context) {
	relay.ImagesHandler("/images/generations", c)
}

func edits(c *gin.Context) {
	relay.ImagesHandler("/images/edits", c)
}

func variations(c *gin.Context) {
	relay.ImagesHandler("/images/variations", c)
}