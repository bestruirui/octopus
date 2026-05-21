package handlers

import (
	"net/http"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/auth"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/user").
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/login", http.MethodPost).
				Handle(login),
		).
		AddRoute(
			router.NewRoute("/login/verify-2fa", http.MethodPost).
				Handle(verify2FA),
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
	if err := op.UserVerify(user.Username, user.Password); err != nil {
		resp.Error(c, http.StatusUnauthorized, resp.ErrUnauthorized)
		return
	}

	// 检查是否启用 2FA
	if op.TOTPIsEnabled() {
		// 如果 2FA 已启用但未提供验证码，返回 2fa_required
		if user.TOTPCode == "" {
			// 生成一个临时 token（1 分钟有效期），用于第二步验证
			tempToken, expire, err := auth.GenerateJWTToken(1)
			if err != nil {
				resp.Error(c, http.StatusInternalServerError, resp.ErrInternalServer)
				return
			}
			resp.Success(c, gin.H{
				"2fa_required": true,
				"temp_token":   tempToken,
				"expire_at":    expire,
			})
			return
		}

		// 如果提供了 TOTP 验证码，直接验证
		secret := op.TOTPGetSecret()
		if !op.ValidateTOTP(secret, user.TOTPCode) {
			resp.Error(c, http.StatusUnauthorized, "Invalid 2FA verification code")
			return
		}
	}

	token, expire, err := auth.GenerateJWTToken(user.Expire)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, resp.ErrInternalServer)
		return
	}
	resp.Success(c, model.UserLoginResponse{Token: token, ExpireAt: expire})
}

// Verify2FARequest 第二步 2FA 验证请求
type Verify2FARequest struct {
	TempToken string `json:"temp_token"`
	TOTPCode  string `json:"totp_code"`
	Expire    int    `json:"expire"`
}

// Verify2FAResponse 第二步 2FA 验证响应
type Verify2FAResponse struct {
	Token    string `json:"token"`
	ExpireAt string `json:"expire_at"`
}

// verify2FA 验证 2FA 验证码并返回完整 JWT
func verify2FA(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}

	// 验证临时 token
	if !auth.VerifyJWTToken(strings.TrimSpace(req.TempToken)) {
		resp.Error(c, http.StatusUnauthorized, "Invalid or expired temp token")
		return
	}

	// 验证 TOTP 码
	if !op.TOTPIsEnabled() {
		resp.Error(c, http.StatusBadRequest, "2FA is not enabled")
		return
	}

	secret := op.TOTPGetSecret()
	if !op.ValidateTOTP(secret, req.TOTPCode) {
		resp.Error(c, http.StatusUnauthorized, "Invalid 2FA verification code")
		return
	}

	token, expire, err := auth.GenerateJWTToken(req.Expire)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, resp.ErrInternalServer)
		return
	}
	resp.Success(c, Verify2FAResponse{Token: token, ExpireAt: expire})
}

func changePassword(c *gin.Context) {
	var user model.UserChangePassword
	if err := c.ShouldBindJSON(&user); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.UserChangePassword(user.OldPassword, user.NewPassword); err != nil {
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
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, "username changed successfully")
}

func status(c *gin.Context) {
	resp.Success(c, "ok")
}
