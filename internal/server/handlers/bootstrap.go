package handlers

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/bootstrap").
		AddRoute(
			router.NewRoute("/status", http.MethodGet).
				Handle(getBootstrapStatus),
		)
}

func getBootstrapStatus(c *gin.Context) {
	initialized, message, err := op.UserBootstrapStatus()
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, resp.ErrInternalServer)
		return
	}
	resp.Success(c, gin.H{
		"initialized": initialized,
		"message":     message,
	})
}
