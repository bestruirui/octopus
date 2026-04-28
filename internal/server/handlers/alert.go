package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/api/v1/alert").
		Use(middleware.Auth()).
		Use(middleware.RequirePermission(auth.PermSettingsRead)).
		Use(middleware.RequireJSON()).
		AddRoute(router.NewRoute("/rule/list", http.MethodGet).Handle(listAlertRules)).
		AddRoute(router.NewRoute("/rule/create", http.MethodPost).Use(middleware.RequirePermission(auth.PermSettingsWrite)).Handle(createAlertRule)).
		AddRoute(router.NewRoute("/rule/update", http.MethodPost).Use(middleware.RequirePermission(auth.PermSettingsWrite)).Handle(updateAlertRule)).
		AddRoute(router.NewRoute("/rule/delete/:id", http.MethodDelete).Use(middleware.RequirePermission(auth.PermSettingsWrite)).Handle(deleteAlertRule)).
		AddRoute(router.NewRoute("/notif/list", http.MethodGet).Handle(listNotifChannels)).
		AddRoute(router.NewRoute("/notif/create", http.MethodPost).Use(middleware.RequirePermission(auth.PermSettingsWrite)).Handle(createNotifChannel)).
		AddRoute(router.NewRoute("/notif/update", http.MethodPost).Use(middleware.RequirePermission(auth.PermSettingsWrite)).Handle(updateNotifChannel)).
		AddRoute(router.NewRoute("/notif/delete/:id", http.MethodDelete).Use(middleware.RequirePermission(auth.PermSettingsWrite)).Handle(deleteNotifChannel)).
		AddRoute(router.NewRoute("/history", http.MethodGet).Handle(listAlertHistory))
}

func listAlertRules(c *gin.Context) {
	rules, err := op.AlertRuleList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, rules)
}

func createAlertRule(c *gin.Context) {
	var rule model.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.AlertRuleCreate(c.Request.Context(), &rule); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, rule)
}

func updateAlertRule(c *gin.Context) {
	var rule model.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.AlertRuleUpdate(c.Request.Context(), &rule); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

func deleteAlertRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := op.AlertRuleDelete(c.Request.Context(), id); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

func listNotifChannels(c *gin.Context) {
	channels, err := op.AlertNotifChannelList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, channels)
}

func createNotifChannel(c *gin.Context) {
	var ch model.AlertNotifChannel
	if err := c.ShouldBindJSON(&ch); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.AlertNotifChannelCreate(c.Request.Context(), &ch); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, ch)
}

func updateNotifChannel(c *gin.Context) {
	var ch model.AlertNotifChannel
	if err := c.ShouldBindJSON(&ch); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.AlertNotifChannelUpdate(c.Request.Context(), &ch); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

func deleteNotifChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := op.AlertNotifChannelDelete(c.Request.Context(), id); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

func listAlertHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	history, err := op.AlertHistoryList(c.Request.Context(), limit)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, history)
}
