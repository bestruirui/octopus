package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/api/v1/route").
		Use(middleware.Auth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/ai-generate", http.MethodPost).
				Handle(generateAIRoute),
		)
}

func generateAIRoute(c *gin.Context) {
	var req model.GenerateAIRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := op.GenerateAIRoute(c.Request.Context(), req)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp.Success(c, result)
}
