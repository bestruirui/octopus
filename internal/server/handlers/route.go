package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/helper"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/api/v1/route").
		Use(middleware.Auth()).
		Use(middleware.RequirePermission(auth.PermGroupsRead)).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/ai-generate", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermGroupsWrite)).
				Handle(generateAIRoute),
		).
		AddRoute(
			router.NewRoute("/ai-generate/status/:id", http.MethodGet).
				Handle(getGenerateAIRouteStatus),
		).
		AddRoute(
			router.NewRoute("/ai-generate/progress/:id", http.MethodGet).
				Handle(getGenerateAIRouteProgress),
		).
		AddRoute(
			router.NewRoute("/ai-generate/result/:id", http.MethodGet).
				Handle(getGenerateAIRouteResult),
		).
		AddRoute(
			router.NewRoute("/ai-generate/stream/:id", http.MethodGet).
				Handle(streamGenerateAIRouteProgress),
		)
}

func generateAIRoute(c *gin.Context) {
	var req model.GenerateAIRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := req.Validate(); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	progress, err := helper.StartGenerateAIRoute(req)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp.Success(c, progress)
}

func getGenerateAIRouteProgress(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		resp.Error(c, http.StatusBadRequest, "missing progress id")
		return
	}

	progress, ok := helper.GetGenerateAIRouteProgress(id)
	if !ok {
		resp.Error(c, http.StatusNotFound, "ai route progress not found")
		return
	}

	resp.Success(c, progress)
}

func getGenerateAIRouteStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		resp.Error(c, http.StatusBadRequest, "missing progress id")
		return
	}

	progress, ok := helper.GetGenerateAIRouteProgress(id)
	if !ok {
		resp.Error(c, http.StatusNotFound, "ai route progress not found")
		return
	}

	resp.Success(c, gin.H{
		"id":               progress.ID,
		"scope":            progress.Scope,
		"group_id":         progress.GroupID,
		"status":           progress.Status,
		"current_step":     progress.CurrentStep,
		"progress_percent": progress.ProgressPercent,
		"done":             progress.Done,
		"result_ready":     progress.ResultReady,
		"message":          progress.Message,
		"error_reason":     progress.ErrorReason,
		"updated_at":       progress.UpdatedAt,
		"heartbeat_at":     progress.HeartbeatAt,
		"finished_at":      progress.FinishedAt,
		"event_sequence":   progress.EventSequence,
	})
}

func getGenerateAIRouteResult(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		resp.Error(c, http.StatusBadRequest, "missing progress id")
		return
	}

	progress, ok := helper.GetGenerateAIRouteProgress(id)
	if !ok {
		resp.Error(c, http.StatusNotFound, "ai route progress not found")
		return
	}

	resp.Success(c, gin.H{
		"id":           progress.ID,
		"status":       progress.Status,
		"done":         progress.Done,
		"ready":        progress.ResultReady,
		"error_reason": progress.ErrorReason,
		"result":       progress.Result,
	})
}

func streamGenerateAIRouteProgress(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		resp.Error(c, http.StatusBadRequest, "missing progress id")
		return
	}

	progress, ok := helper.GetGenerateAIRouteProgress(id)
	if !ok {
		resp.Error(c, http.StatusNotFound, "ai route progress not found")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	progressChan, unsubscribe := helper.SubscribeGenerateAIRouteProgress(id)
	defer unsubscribe()

	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	if err := writeAIRouteProgressEvent(c, "progress", *progress); err != nil {
		return
	}
	if progress.Done {
		return
	}

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeatTicker.C:
			if _, err := c.Writer.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			c.Writer.Flush()
		case nextProgress, ok := <-progressChan:
			if !ok {
				return
			}

			eventName := "progress"
			switch nextProgress.Status {
			case model.AIRouteTaskStatusCompleted:
				eventName = "completed"
			case model.AIRouteTaskStatusFailed:
				eventName = "failed"
			case model.AIRouteTaskStatusTimeout:
				eventName = "timeout"
			}

			if err := writeAIRouteProgressEvent(c, eventName, nextProgress); err != nil {
				return
			}
			if nextProgress.Done {
				return
			}
		}
	}
}

func writeAIRouteProgressEvent(c *gin.Context, eventName string, progress model.GenerateAIRouteProgress) error {
	data, err := json.Marshal(progress)
	if err != nil {
		return err
	}

	if _, err := c.Writer.Write([]byte(fmt.Sprintf("id: %d\n", progress.EventSequence))); err != nil {
		return err
	}
	if eventName != "" {
		if _, err := c.Writer.Write([]byte(fmt.Sprintf("event: %s\n", eventName))); err != nil {
			return err
		}
	}
	if _, err := c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data))); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}
