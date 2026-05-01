package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
)

type apiKeyStatsResponse struct {
	model.StatsAPIKey
	Name string `json:"name"`
}

type channelStatsResponse struct {
	model.StatsChannel
	ChannelName string `json:"channel_name"`
	Enabled     bool   `json:"enabled"`
}

func init() {
	router.NewGroupRouter("/api/v1/stats").
		Use(middleware.Auth()).
		Use(middleware.RequirePermission(auth.PermStatsRead)).
		AddRoute(
			router.NewRoute("/today", http.MethodGet).
				Handle(getStatsToday),
		).
		AddRoute(
			router.NewRoute("/daily", http.MethodGet).
				Handle(getStatsDaily),
		).
		AddRoute(
			router.NewRoute("/hourly", http.MethodGet).
				Handle(getStatsHourly),
		).
		AddRoute(
			router.NewRoute("/total", http.MethodGet).
				Handle(getStatsTotal),
		).
		AddRoute(
			router.NewRoute("/channel", http.MethodGet).
				Handle(getStatsChannel),
		).
		AddRoute(
			router.NewRoute("/apikey", http.MethodGet).
				Handle(getStatsAPIKey),
		)
}

func getStatsToday(c *gin.Context) {
	resp.Success(c, op.StatsTodayGet())
}

func getStatsDaily(c *gin.Context) {
	statsDaily, err := op.StatsGetDaily(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, statsDaily)
}

func getStatsHourly(c *gin.Context) {
	resp.Success(c, op.StatsHourlyGet())
}

func getStatsTotal(c *gin.Context) {
	resp.Success(c, op.StatsTotalGet())
}

func getStatsChannel(c *gin.Context) {
	stats := op.StatsChannelList()
	statsByChannelID := make(map[int]model.StatsChannel, len(stats))
	for _, item := range stats {
		statsByChannelID[item.ChannelID] = item
	}

	channels, err := op.ChannelList(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}

	result := make([]channelStatsResponse, 0, len(channels))
	for _, channel := range channels {
		channelStats, ok := statsByChannelID[channel.ID]
		if !ok {
			channelStats = model.StatsChannel{ChannelID: channel.ID}
		} else {
			delete(statsByChannelID, channel.ID)
		}

		result = append(result, channelStatsResponse{
			StatsChannel: channelStats,
			ChannelName:  channel.Name,
			Enabled:      channel.Enabled,
		})
	}

	for channelID, item := range statsByChannelID {
		result = append(result, channelStatsResponse{
			StatsChannel: item,
			ChannelName:  fmt.Sprintf("Channel #%d", channelID),
			Enabled:      false,
		})
	}

	resp.Success(c, result)
}

func getStatsAPIKey(c *gin.Context) {
	stats := op.StatsAPIKeyList()

	apiKeys, err := op.APIKeyList(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}

	apiKeyNames := make(map[int]string, len(apiKeys))
	for _, apiKey := range apiKeys {
		apiKeyNames[apiKey.ID] = apiKey.Name
	}

	result := make([]apiKeyStatsResponse, 0, len(stats))
	for _, item := range stats {
		name, ok := apiKeyNames[item.APIKeyID]
		if !ok {
			name = fmt.Sprintf("Key #%d", item.APIKeyID)
		}
		result = append(result, apiKeyStatsResponse{
			StatsAPIKey: item,
			Name:        name,
		})
	}

	resp.Success(c, result)
}
