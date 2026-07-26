package handlers

import (
	"errors"
	"net/http"

	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	tailscaletunnel "github.com/bestruirui/octopus/internal/tailscale"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/tailscale").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/status", http.MethodGet).
				Handle(getTailscaleStatus),
		).
		AddRoute(
			router.NewRoute("/start", http.MethodPost).
				Handle(startTailscaleFunnel),
		).
		AddRoute(
			router.NewRoute("/stop", http.MethodPost).
				Handle(stopTailscaleFunnel),
		)
}

func getTailscaleStatus(c *gin.Context) {
	resp.Success(c, tailscaletunnel.GetStatus(c.Request.Context()))
}

func startTailscaleFunnel(c *gin.Context) {
	status, err := tailscaletunnel.Start(c.Request.Context())
	if err != nil {
		resp.Error(c, tailscaleErrorStatus(err), err.Error())
		return
	}
	resp.Success(c, status)
}

func stopTailscaleFunnel(c *gin.Context) {
	status, err := tailscaletunnel.Stop(c.Request.Context())
	if err != nil {
		resp.Error(c, tailscaleErrorStatus(err), err.Error())
		return
	}
	resp.Success(c, status)
}

func tailscaleErrorStatus(err error) int {
	switch {
	case errors.Is(err, tailscaletunnel.ErrUnsupported),
		errors.Is(err, tailscaletunnel.ErrNotInstalled),
		errors.Is(err, tailscaletunnel.ErrNeedsLogin),
		errors.Is(err, tailscaletunnel.ErrDefaultCredential):
		return http.StatusBadRequest
	case errors.Is(err, tailscaletunnel.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
