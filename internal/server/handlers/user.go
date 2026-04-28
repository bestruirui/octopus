package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
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
			router.NewRoute("/create", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermUsersWrite)).
				Handle(createUser),
		).
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
		).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Use(middleware.RequirePermission(auth.PermUsersRead)).
				Handle(listUsers),
		).
		AddRoute(
			router.NewRoute("/update-role", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermUsersWrite)).
				Handle(updateUserRole),
		).
		AddRoute(
			router.NewRoute("/delete/:id", http.MethodDelete).
				Use(middleware.RequirePermission(auth.PermUsersWrite)).
				Handle(deleteUser),
		)
}

func createUser(c *gin.Context) {
	var req model.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.UserCreate(req, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.Success(c, nil)
}

func listUsers(c *gin.Context) {
	users, err := op.UserList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, users)
}

func updateUserRole(c *gin.Context) {
	var req struct {
		ID   uint   `json:"id" binding:"required"`
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.UserUpdateRole(req.ID, req.Role, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.Success(c, nil)
}

func deleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}
	currentUserID := uint(c.GetInt("user_id"))
	if err := op.UserDelete(uint(id), currentUserID, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.Success(c, nil)
}

func login(c *gin.Context) {
	var user model.UserLogin
	if err := c.ShouldBindJSON(&user); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	loginKey := c.GetString("login_rate_limit_key")
	userObj, err := op.UserVerify(user.Username, user.Password)
	if err != nil {
		if errors.Is(err, op.ErrUserNotInitialized) {
			resp.Error(c, http.StatusConflict, err.Error())
			return
		}
		middleware.RecordLoginFailure(loginKey, time.Now())
		resp.Error(c, http.StatusUnauthorized, resp.ErrUnauthorized)
		return
	}
	middleware.ClearLoginFailures(loginKey)
	token, expire, err := auth.GenerateJWTToken(user.Expire, userObj.ID, userObj.Role)
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
	currentUserID := uint(c.GetInt("user_id"))
	if err := op.UserChangePassword(currentUserID, user.OldPassword, user.NewPassword); err != nil {
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
	currentUserID := uint(c.GetInt("user_id"))
	if err := op.UserChangeUsername(currentUserID, user.NewUsername); err != nil {
		if strings.Contains(err.Error(), "same as the old username") || strings.Contains(err.Error(), "username already exists") || strings.Contains(err.Error(), "username is required") {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		resp.InternalError(c)
		return
	}
	resp.Success(c, "username changed successfully")
}

func status(c *gin.Context) {
	if !op.UserReady() {
		resp.Error(c, http.StatusConflict, op.ErrUserNotInitialized.Error())
		return
	}
	resp.Success(c, "ok")
}
