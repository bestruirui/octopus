package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/api/v1/analytics").
		Use(middleware.Auth()).
		Use(middleware.RequirePermission(auth.PermStatsRead)).
		AddRoute(
			router.NewRoute("/overview", http.MethodGet).
				Handle(getAnalyticsOverview),
		).
		AddRoute(
			router.NewRoute("/utilization", http.MethodGet).
				Handle(getAnalyticsUtilization),
		).
		AddRoute(
			router.NewRoute("/evaluation", http.MethodGet).
				Handle(getAnalyticsEvaluation),
		).
		AddRoute(
			router.NewRoute("/group-health", http.MethodGet).
				Handle(getAnalyticsGroupHealth),
		).
		AddRoute(
			router.NewRoute("/provider-breakdown", http.MethodGet).
				Handle(getAnalyticsProviderBreakdown),
		).
		AddRoute(
			router.NewRoute("/model-breakdown", http.MethodGet).
				Handle(getAnalyticsModelBreakdown),
		).
		AddRoute(
			router.NewRoute("/apikey-breakdown", http.MethodGet).
				Handle(getAnalyticsAPIKeyBreakdown),
		)
}

func getAnalyticsOverview(c *gin.Context) {
	analyticsRange, ok := parseAnalyticsRange(c)
	if !ok {
		return
	}

	data, err := op.AnalyticsOverviewGet(c.Request.Context(), analyticsRange)
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}

func getAnalyticsUtilization(c *gin.Context) {
	analyticsRange, ok := parseAnalyticsRange(c)
	if !ok {
		return
	}

	data, err := op.AnalyticsUtilizationGet(c.Request.Context(), analyticsRange)
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}

func getAnalyticsEvaluation(c *gin.Context) {
	data, err := op.AnalyticsEvaluationGet(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}

func getAnalyticsGroupHealth(c *gin.Context) {
	data, err := op.AnalyticsGroupHealthGet(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}

func getAnalyticsProviderBreakdown(c *gin.Context) {
	analyticsRange, ok := parseAnalyticsRange(c)
	if !ok {
		return
	}

	data, err := op.AnalyticsProviderBreakdownGet(c.Request.Context(), analyticsRange)
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}

func getAnalyticsModelBreakdown(c *gin.Context) {
	analyticsRange, ok := parseAnalyticsRange(c)
	if !ok {
		return
	}

	data, err := op.AnalyticsModelBreakdownGet(c.Request.Context(), analyticsRange)
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}

func getAnalyticsAPIKeyBreakdown(c *gin.Context) {
	analyticsRange, ok := parseAnalyticsRange(c)
	if !ok {
		return
	}

	data, err := op.AnalyticsAPIKeyBreakdownGet(c.Request.Context(), analyticsRange)
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}

func parseAnalyticsRange(c *gin.Context) (model.AnalyticsRange, bool) {
	analyticsRange, err := model.ParseAnalyticsRange(c.Query("range"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return "", false
	}
	return analyticsRange, true
}
