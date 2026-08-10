package handlers

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/circuit-breaker").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(listCircuitBreaker),
		).
		AddRoute(
			router.NewRoute("/manual", http.MethodPost).
				Use(middleware.RequireJSON()).
				Handle(setCircuitBreakerManual),
		).
		AddRoute(
			router.NewRoute("/reset", http.MethodPost).
				Use(middleware.RequireJSON()).
				Handle(resetCircuitBreaker),
		)
}

// listCircuitBreaker 返回所有渠道+模型的熔断状态（仅包含仍存在的渠道）。
func listCircuitBreaker(c *gin.Context) {
	ctx := c.Request.Context()
	records := balancer.List(ctx)

	// 过滤掉已删除渠道的残留记录，并补充渠道名
	channels, err := op.ChannelList(ctx)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	channelNameByID := make(map[int]string, len(channels))
	for _, ch := range channels {
		channelNameByID[ch.ID] = ch.Name
	}
	result := make([]model.CircuitBreaker, 0, len(records))
	for _, r := range records {
		name, ok := channelNameByID[r.ChannelID]
		if !ok {
			continue
		}
		if r.ChannelName == "" || r.ChannelName != name {
			r.ChannelName = name
		}
		result = append(result, r)
	}
	resp.Success(c, result)
}

// setCircuitBreakerManual 手动启用/禁用某个渠道+模型的熔断状态。
func setCircuitBreakerManual(c *gin.Context) {
	var req model.CircuitBreakerManualRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	channel, err := op.ChannelGet(req.ChannelID, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusNotFound, "channel not found")
		return
	}
	balancer.SetManualDisabled(req.ChannelID, req.ModelName, channel.Name, req.Disabled)
	resp.Success(c, nil)
}

// resetCircuitBreaker 取消熔断：清除熔断状态与手动禁用标记。
func resetCircuitBreaker(c *gin.Context) {
	var req model.CircuitBreakerResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	balancer.Reset(req.ChannelID, req.ModelName)
	resp.Success(c, nil)
}
