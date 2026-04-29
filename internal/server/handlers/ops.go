package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/api/v1/ops").
		Use(middleware.Auth()).
		Use(middleware.RequirePermission(auth.PermSettingsRead)).
		AddRoute(
			router.NewRoute("/cache", http.MethodGet).
				Handle(getOpsCache),
		).
		AddRoute(
			router.NewRoute("/quota", http.MethodGet).
				Handle(getOpsQuota),
		).
		AddRoute(
			router.NewRoute("/health", http.MethodGet).
				Handle(getOpsHealth),
		).
		AddRoute(
			router.NewRoute("/system", http.MethodGet).
				Handle(getOpsSystem),
		)
}

func getOpsCache(c *gin.Context) {
	data, err := op.OpsCacheStatusGet(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}

func getOpsQuota(c *gin.Context) {
	data, err := op.OpsQuotaSummaryGet(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}

func getOpsHealth(c *gin.Context) {
	data, err := op.OpsHealthStatusGet(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}

func getOpsSystem(c *gin.Context) {
	data, err := op.OpsSystemSummaryGet(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}
