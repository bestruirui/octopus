package handlers

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/price"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/bestruirui/octopus/internal/task"
	"github.com/bestruirui/octopus/internal/utils/xstrings"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/channel").
		Use(middleware.Auth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(listChannel),
		).
		AddRoute(
			router.NewRoute("/create", http.MethodPost).
				Handle(createChannel),
		).
		AddRoute(
			router.NewRoute("/update", http.MethodPost).
				Handle(updateChannel),
		).
		AddRoute(
			router.NewRoute("/enable", http.MethodPost).
				Handle(enableChannel),
		).
		AddRoute(
			router.NewRoute("/delete/:id", http.MethodDelete).
				Handle(deleteChannel),
		).
		AddRoute(
			router.NewRoute("/fetch-model", http.MethodPost).
				Handle(fetchModel),
		)
	router.NewGroupRouter("/api/v1/channel").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/sync", http.MethodPost).
				Handle(syncChannel),
		).
		AddRoute(
			router.NewRoute("/last-sync-time", http.MethodGet).
				Handle(getLastSyncTime),
		)
}

func listChannel(c *gin.Context) {
	channels := op.ChannelList()
	for i, channel := range channels {
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
	channel.Model = normalizeChannelModelList(channel.Model)
	channel.CustomModel = normalizeChannelModelList(channel.CustomModel)
	channel.Model = excludeCustomChannelModels(channel.Model, channel.CustomModel)
	if err := op.ChannelCreate(&channel, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := addChannelModelPrices(xstrings.SplitCompact(",", channel.Model, channel.CustomModel), c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	stats := op.StatsChannelGet(channel.ID)
	channel.Stats = &stats
	resp.Success(c, channel)
}

func updateChannel(c *gin.Context) {
	var req model.ChannelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if req.Model != nil {
		normalizedModels := normalizeChannelModelList(*req.Model)
		req.Model = &normalizedModels
	}
	if req.CustomModel != nil {
		normalizedCustomModels := normalizeChannelModelList(*req.CustomModel)
		req.CustomModel = &normalizedCustomModels
	}
	channel, err := op.ChannelUpdate(&req, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	newModelNames := xstrings.SplitCompact(",", channel.Model, channel.CustomModel)
	if err := addChannelModelPrices(newModelNames, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := op.LLMCleanupGhosts(c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	stats := op.StatsChannelGet(channel.ID)
	channel.Stats = &stats
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
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

func deleteChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidParam)
		return
	}
	if err := op.ChannelDel(id, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := op.LLMCleanupGhosts(c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

// normalizeChannelModelList 规范化外部提交的渠道模型列表。
func normalizeChannelModelList(models string) string {
	modelNames := xstrings.SplitTrimCompact(",", models)
	seen := make(map[string]struct{}, len(modelNames))
	normalizedModelNames := modelNames[:0]
	for _, modelName := range modelNames {
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		normalizedModelNames = append(normalizedModelNames, modelName)
	}
	return strings.Join(normalizedModelNames, ",")
}

// excludeCustomChannelModels 从已规范化的自动模型列表中排除手动模型。
func excludeCustomChannelModels(models, customModels string) string {
	modelNames := xstrings.SplitCompact(",", models)
	customModelNames := xstrings.SplitCompact(",", customModels)
	modelNames = slices.DeleteFunc(modelNames, func(modelName string) bool {
		return slices.Contains(customModelNames, modelName)
	})
	return strings.Join(modelNames, ",")
}

// addChannelModelPrices 为渠道模型匹配校准价格，并批量写入尚不存在的价格记录。
func addChannelModelPrices(modelNames []string, ctx context.Context) error {
	seen := make(map[string]struct{}, len(modelNames))
	llmInfos := make([]model.LLMInfo, 0, len(modelNames))
	for _, modelName := range modelNames {
		modelName = strings.ToLower(modelName)
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		llmInfo := model.LLMInfo{Name: modelName}
		if modelPrice := price.GetLLMPrice(modelName); modelPrice != nil {
			llmInfo.LLMPrice = *modelPrice
		}
		llmInfos = append(llmInfos, llmInfo)
	}
	return op.LLMBatchCreate(llmInfos, ctx)
}

func fetchModel(c *gin.Context) {
	var request model.Channel
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	models, err := helper.FetchModels(c.Request.Context(), request)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := addChannelModelPrices(models, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, models)
}

func syncChannel(c *gin.Context) {
	if err := task.SyncModelsTask(); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

func getLastSyncTime(c *gin.Context) {
	resp.Success(c, task.GetLastSyncModelsTime())
}
