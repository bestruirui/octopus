package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay/breaker"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/circuit-breaker").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/group/:groupId/states", http.MethodGet).
				Handle(getGroupCircuitBreakerStates),
		).
		AddRoute(
			router.NewRoute("/channel/:channelId/reset", http.MethodPost).
				Handle(resetChannelCircuitBreaker),
		).
		AddRoute(
			router.NewRoute("/item/reset", http.MethodPost).
				Use(middleware.RequireJSON()).
				Handle(resetCircuitBreakerItem),
		)
}

func getGroupCircuitBreakerStates(c *gin.Context) {
	groupID, ok := parseParamInt(c, "groupId")
	if !ok {
		return
	}
	group, err := op.GroupGet(groupID, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusNotFound, err.Error())
		return
	}

	m := breaker.GetManager()
	items := make([]model.GroupCircuitBreakerItemState, 0, len(group.Items))
	for _, item := range group.Items {
		channelName := ""
		if ch, chErr := op.ChannelGet(item.ChannelID, c.Request.Context()); chErr == nil {
			channelName = ch.Name
		}
		key := breaker.BuildKey(item.ChannelID, item.ModelName)
		snap := m.Snapshot(key)
		items = append(items, model.GroupCircuitBreakerItemState{
			GroupID:             group.ID,
			GroupName:           group.Name,
			ChannelID:           item.ChannelID,
			ChannelName:         channelName,
			ModelName:           item.ModelName,
			BreakerKey:          key,
			State:               snap.State,
			ConsecutiveFailures: snap.ConsecutiveFailures,
			TripCount:           snap.TripCount,
			LastFailureAt:       formatTime(snap.LastFailureAt),
			LastFailureReason:   snap.LastFailureReason,
			LastTripAt:          formatTime(snap.LastTripAt),
			OpenUntil:           formatTime(snap.OpenUntil),
			OpenRemainingSecond: openRemainingSeconds(snap.OpenUntil),
			ProbeInFlight:       snap.ProbeInFlight,
		})
	}

	resp.Success(c, model.GroupCircuitBreakerStatesResponse{
		GroupID:   group.ID,
		GroupName: group.Name,
		Items:     items,
	})
}

func resetChannelCircuitBreaker(c *gin.Context) {
	channelID, ok := parseParamInt(c, "channelId")
	if !ok {
		return
	}
	affected, keys := breaker.GetManager().ResetChannel(channelID)
	resp.Success(c, model.CircuitBreakerResetResponse{
		ChannelID:        channelID,
		AffectedBreakers: affected,
		BreakerKeys:      keys,
	})
}

func resetCircuitBreakerItem(c *gin.Context) {
	var req struct {
		ChannelID int    `json:"channel_id" binding:"required"`
		ModelName string `json:"model_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	key := breaker.BuildKey(req.ChannelID, req.ModelName)
	affected := 0
	if breaker.GetManager().ResetKey(key) {
		affected = 1
	}
	resp.Success(c, gin.H{
		"breaker_key":       key,
		"affected_breakers": affected,
	})
}

func parseParamInt(c *gin.Context, name string) (int, bool) {
	raw := c.Param(name)
	v, err := strconv.Atoi(raw)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return 0, false
	}
	return v, true
}

func formatTime(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339Nano)
}

func openRemainingSeconds(openUntil time.Time) int {
	if openUntil.IsZero() {
		return 0
	}
	now := time.Now()
	if !openUntil.After(now) {
		return 0
	}
	return int(openUntil.Sub(now).Seconds() + 0.999)
}
