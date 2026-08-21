package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

var activityMaxRequestCount int64 // 最近 54 周每日请求量的最大值。
var activityMaxCalculatedAt time.Time // 最大值上次计算时间。
var activityMaxMu sync.Mutex // 保护最大值及计算时间的并发更新。

type statsDailyResponse struct {
	MaxRequestCount int64             `json:"max_request_count"` // 最近 54 周每日请求量的最大值。
	Items           []model.StatsDaily `json:"items"` // 每日原始统计数据。
}

func init() {
	router.NewGroupRouter("/api/v1/stats").
		Use(middleware.Auth()).
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
			router.NewRoute("/apikey", http.MethodGet).
				Handle(getStatsAPIKey),
		).
		AddRoute(
			router.NewRoute("/group", http.MethodGet).
				Handle(getStatsGroup),
		)
}

func getStatsToday(c *gin.Context) {
	resp.Success(c, op.StatsTodayGet())
}

func getStatsDaily(c *gin.Context) {
	statsDaily, err := op.StatsGetDaily(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now()
	activityMaxMu.Lock()
	if activityMaxCalculatedAt.IsZero() || now.Sub(activityMaxCalculatedAt) >= 24*time.Hour {
		cutoff := now.AddDate(0, 0, -(int(now.Weekday()) + 53*7)).Format("20060102")
		maxRequestCount := int64(0)
		for _, daily := range statsDaily {
			if daily.Date < cutoff {
				continue
			}
			requestCount := daily.RequestSuccess + daily.RequestFailed
			if requestCount > maxRequestCount {
				maxRequestCount = requestCount
			}
		}
		activityMaxRequestCount = maxRequestCount
		activityMaxCalculatedAt = now
	}
	maxRequestCount := activityMaxRequestCount
	activityMaxMu.Unlock()

	resp.Success(c, statsDailyResponse{
		MaxRequestCount: maxRequestCount,
		Items:           statsDaily,
	})
}

func getStatsHourly(c *gin.Context) {
	resp.Success(c, op.StatsHourlyGet())
}

func getStatsTotal(c *gin.Context) {
	resp.Success(c, op.StatsTotalGet())
}

func getStatsAPIKey(c *gin.Context) {
	resp.Success(c, op.StatsAPIKeyList())
}

func getStatsGroup(c *gin.Context) {
	resp.Success(c, op.StatsGroupList())
}
