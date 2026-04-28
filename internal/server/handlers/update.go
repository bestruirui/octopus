package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/conf"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
	"github.com/lingyuins/octopus/internal/update"
)

func init() {
	router.NewGroupRouter("/api/v1/update").
		Use(middleware.Auth()).
		Use(middleware.RequirePermission(auth.PermSettingsRead)).
		AddRoute(
			router.NewRoute("", http.MethodGet).
				Handle(latest),
		).
		AddRoute(
			router.NewRoute("/now-version", http.MethodGet).
				Handle(getNowVersion),
		).
		AddRoute(
			router.NewRoute("", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermSettingsWrite)).
				Handle(updateFunc),
		)
}

func latest(c *gin.Context) {
	latestInfo, err := update.GetLatestInfo()
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, *latestInfo)
}

func getNowVersion(c *gin.Context) {
	resp.Success(c, conf.Version)
}

func updateFunc(c *gin.Context) {
	err := update.UpdateCore()
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, "update success")
}
