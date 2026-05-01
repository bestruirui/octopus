package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/helper"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
	"github.com/lingyuins/octopus/internal/task"
)

func init() {
	router.NewGroupRouter("/api/v1/channel").
		Use(middleware.Auth()).
		Use(middleware.RequirePermission(auth.PermChannelsRead)).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(listChannel),
		).
		AddRoute(
			router.NewRoute("/create", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermChannelsWrite)).
				Handle(createChannel),
		).
		AddRoute(
			router.NewRoute("/update", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermChannelsWrite)).
				Handle(updateChannel),
		).
		AddRoute(
			router.NewRoute("/enable", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermChannelsWrite)).
				Handle(enableChannel),
		).
		AddRoute(
			router.NewRoute("/delete/:id", http.MethodDelete).
				Use(middleware.RequirePermission(auth.PermChannelsWrite)).
				Handle(deleteChannel),
		).
		AddRoute(
			router.NewRoute("/fetch-model", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermChannelsWrite)).
				Handle(fetchModel),
		).
		AddRoute(
			router.NewRoute("/test", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermChannelsWrite)).
				Handle(testChannel),
		)
	router.NewGroupRouter("/api/v1/channel").
		Use(middleware.Auth()).
		Use(middleware.RequirePermission(auth.PermChannelsRead)).
		AddRoute(
			router.NewRoute("/sync", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermChannelsWrite)).
				Handle(syncChannel),
		).
		AddRoute(
			router.NewRoute("/last-sync-time", http.MethodGet).
				Handle(getLastSyncTime),
		)
}

func listChannel(c *gin.Context) {
	channels, err := op.ChannelList(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	canViewRawKeys := auth.HasPermission(c.GetString("user_role"), auth.PermChannelsWrite)
	for i, channel := range channels {
		if !canViewRawKeys {
			channels[i].Keys = maskChannelKeys(channel.Keys)
		}
		stats := op.StatsChannelGet(channel.ID)
		channels[i].Stats = &stats
	}
	resp.Success(c, channels)
}

func createChannel(c *gin.Context) {
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.ChannelCreate(&channel, c.Request.Context()); err != nil {
		if status, msg, ok := classifyChannelMutationError(err); ok {
			resp.Error(c, status, msg)
			return
		}
		resp.InternalError(c)
		return
	}
	stats := op.StatsChannelGet(channel.ID)
	channel.Stats = &stats
	go func(channel *model.Channel) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		modelStr := channel.Model + "," + channel.CustomModel
		modelArray := strings.Split(modelStr, ",")
		helper.LLMPriceAddToDB(modelArray, ctx)
		helper.ChannelBaseUrlDelayUpdate(channel, ctx)
		helper.ChannelAutoGroup(channel, ctx)
	}(&channel)
	resp.Success(c, channel)
}

func updateChannel(c *gin.Context) {
	var req model.ChannelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	channel, err := op.ChannelUpdate(&req, c.Request.Context())
	if err != nil {
		if status, msg, ok := classifyChannelMutationError(err); ok {
			resp.Error(c, status, msg)
			return
		}
		resp.InternalError(c)
		return
	}
	stats := op.StatsChannelGet(channel.ID)
	channel.Stats = &stats
	go func(channel *model.Channel) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		modelStr := channel.Model + "," + channel.CustomModel
		modelArray := strings.Split(modelStr, ",")
		helper.LLMPriceAddToDB(modelArray, ctx)
		helper.ChannelBaseUrlDelayUpdate(channel, ctx)
		helper.ChannelAutoGroup(channel, ctx)
	}(channel)
	resp.Success(c, channel)
}

func enableChannel(c *gin.Context) {
	var request struct {
		ID      int  `json:"id"`
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.ChannelEnabled(request.ID, request.Enabled, c.Request.Context()); err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, nil)
}

func deleteChannel(c *gin.Context) {
	id := c.Param("id")
	idNum, err := strconv.Atoi(id)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidParam)
		return
	}
	if err := op.ChannelDel(idNum, c.Request.Context()); err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, nil)
}
func fetchModel(c *gin.Context) {
	var request model.Channel
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	models, err := helper.FetchModels(c.Request.Context(), request)
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, models)
}

func testChannel(c *gin.Context) {
	var request model.Channel
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	summary, err := helper.TestChannel(c.Request.Context(), request)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.Success(c, summary)
}

func syncChannel(c *gin.Context) {
	task.SyncModelsTask()
	resp.Success(c, nil)
}

func getLastSyncTime(c *gin.Context) {
	time := task.GetLastSyncModelsTime()
	resp.Success(c, time)
}

func classifyChannelMutationError(err error) (int, string, bool) {
	if err == nil {
		return 0, "", false
	}

	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "channel not found"):
		return http.StatusNotFound, "channel not found", true
	case strings.Contains(msg, "request rewrite profile is required when enabled"),
		strings.Contains(msg, "unsupported request rewrite profile"),
		strings.Contains(msg, "unsupported tool role strategy"),
		strings.Contains(msg, "unsupported system message strategy"),
		strings.Contains(msg, "request rewrite profile") && strings.Contains(msg, "is not supported for channel type"):
		return http.StatusBadRequest, err.Error(), true
	case strings.Contains(msg, "request_rewrite") &&
		(strings.Contains(msg, "no such column") ||
			strings.Contains(msg, "has no column named") ||
			strings.Contains(msg, "unknown column")):
		return http.StatusServiceUnavailable, "database schema is outdated", true
	case strings.Contains(msg, "unique constraint failed: channels.name"),
		strings.Contains(msg, "duplicate entry") && strings.Contains(msg, "channels.name"):
		return http.StatusConflict, "channel name already exists", true
	default:
		return 0, "", false
	}
}

func maskChannelKeys(keys []model.ChannelKey) []model.ChannelKey {
	if len(keys) == 0 {
		return nil
	}

	masked := make([]model.ChannelKey, len(keys))
	for i, key := range keys {
		key.ChannelKey = maskChannelKeyValue(key.ChannelKey)
		masked[i] = key
	}
	return masked
}

func maskChannelKeyValue(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 8 {
		return strings.Repeat("*", len(trimmed))
	}
	return trimmed[:4] + strings.Repeat("*", len(trimmed)-8) + trimmed[len(trimmed)-4:]
}
