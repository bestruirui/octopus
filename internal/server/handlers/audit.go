package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/api/v1/audit").
		Use(middleware.Auth()).
		Use(middleware.RequirePermission(auth.PermLogsRead)).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(listAuditLogs),
		).
		AddRoute(
			router.NewRoute("/detail", http.MethodGet).
				Handle(getAuditLogDetail),
		)
}

func listAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, err := op.AuditLogList(c.Request.Context(), page, pageSize)
	if err != nil {
		resp.InternalError(c)
		return
	}

	resp.Success(c, logs)
}

func getAuditLogDetail(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		resp.Error(c, http.StatusBadRequest, "id is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	auditLog, err := op.AuditLogGetByID(c.Request.Context(), id)
	if err != nil {
		resp.InternalError(c)
		return
	}
	if auditLog == nil {
		resp.Error(c, http.StatusNotFound, "audit log not found")
		return
	}

	resp.Success(c, auditLog)
}
