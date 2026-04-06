package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/auth"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	publicUserRoutes := router.NewGroupRouter("/api/v1/user").
		Use(middleware.RequireJSON())

	publicUserRoutes.AddRoute(
		router.NewRoute("/login", http.MethodPost).
			Use(middleware.LoginRateLimit()).
			Handle(login),
	)

	router.NewGroupRouter("/api/v1/user").
		Use(middleware.Auth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/change-password", http.MethodPost).
				Handle(changePassword),
		).
		AddRoute(
			router.NewRoute("/change-username", http.MethodPost).
				Handle(changeUsername),
		).
		AddRoute(
			router.NewRoute("/status", http.MethodGet).
				Handle(status),
		)
}

func login(c *gin.Context) {
	var user model.UserLogin
	if err := c.ShouldBindJSON(&user); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	loginKey := c.GetString("login_rate_limit_key")
	if err := op.UserVerify(user.Username, user.Password); err != nil {
		if errors.Is(err, op.ErrUserNotInitialized) {
			resp.Error(c, http.StatusConflict, err.Error())
			return
		}
		middleware.RecordLoginFailure(loginKey, time.Now())
		resp.Error(c, http.StatusUnauthorized, resp.ErrUnauthorized)
		return
	}
	middleware.ClearLoginFailures(loginKey)
	token, expire, err := auth.GenerateJWTToken(user.Expire)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, resp.ErrInternalServer)
		return
	}
	resp.Success(c, model.UserLoginResponse{Token: token, ExpireAt: expire})
}

func changePassword(c *gin.Context) {
	var user model.UserChangePassword
	if err := c.ShouldBindJSON(&user); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.UserChangePassword(user.OldPassword, user.NewPassword); err != nil {
		if strings.Contains(err.Error(), "incorrect old password") {
			resp.Error(c, http.StatusUnauthorized, resp.ErrUnauthorized)
			return
		}
		resp.Error(c, http.StatusInternalServerError, resp.ErrDatabase)
		return
	}
	resp.Success(c, "password changed successfully")
}

func changeUsername(c *gin.Context) {
	var user model.UserChangeUsername
	if err := c.ShouldBindJSON(&user); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.UserChangeUsername(user.NewUsername); err != nil {
		if strings.Contains(err.Error(), "same as the old username") {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, "username changed successfully")
}

func status(c *gin.Context) {
	resp.Success(c, "ok")
}
