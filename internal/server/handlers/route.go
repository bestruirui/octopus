package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/helper"
	"github.com/lingyuins/octopus/internal/model"
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
		).
		AddRoute(
			router.NewRoute("/ai-generate/progress/:id", http.MethodGet).
				Handle(getGenerateAIRouteProgress),
		)
}

func generateAIRoute(c *gin.Context) {
	var req model.GenerateAIRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	progress, err := helper.StartGenerateAIRoute(req)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp.Success(c, progress)
}

func getGenerateAIRouteProgress(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		resp.Error(c, http.StatusBadRequest, "missing progress id")
		return
	}

	progress, ok := helper.GetGenerateAIRouteProgress(id)
	if !ok {
		resp.Error(c, http.StatusNotFound, "ai route progress not found")
		return
	}

	resp.Success(c, progress)
}
