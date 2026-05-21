package handlers

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/conf"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	// TOTP 初始化端点 - 需要认证
	router.NewGroupRouter("/api/v1/user/totp").
		Use(middleware.Auth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/init", http.MethodPost).
				Handle(initTOTP),
		).AddRoute(
		router.NewRoute("/verify-setup", http.MethodPost).
			Handle(verifyAndEnableTOTP),
	).AddRoute(
		router.NewRoute("/disable", http.MethodPost).
			Handle(disableTOTP),
	).AddRoute(
		router.NewRoute("/status", http.MethodGet).
			Handle(getTOTPStatus),
	)
}

// TOTPInitResponse TOTP 初始化响应
type TOTPInitResponse struct {
	Secret string `json:"secret"`
	Uri    string `json:"uri"`
}

// initTOTP 初始化 TOTP（生成密钥和 URI）
func initTOTP(c *gin.Context) {
	// 如果 TOTP 已启用，不允许重新初始化
	if op.TOTPIsEnabled() {
		resp.Error(c, http.StatusBadRequest, "2FA is already enabled")
		return
	}

	secret, err := op.GenerateTOTPSecret()
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, resp.ErrInternalServer)
		return
	}

	user := op.UserGet()
	uri := op.GenerateTOTPUri(secret, user.Username, conf.APP_NAME)

	// 保存密钥到数据库（但还未启用）
	if err := op.TOTPSetSecret(secret); err != nil {
		resp.Error(c, http.StatusInternalServerError, resp.ErrInternalServer)
		return
	}

	resp.Success(c, TOTPInitResponse{
		Secret: secret,
		Uri:    uri,
	})
}

// VerifyAndEnableTOTPRequest 验证并启用 TOTP 请求
type VerifyAndEnableTOTPRequest struct {
	Code string `json:"code"`
}

// verifyAndEnableTOTP 验证验证码并启用 TOTP
func verifyAndEnableTOTP(c *gin.Context) {
	var req VerifyAndEnableTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}

	if op.TOTPIsEnabled() {
		resp.Error(c, http.StatusBadRequest, "2FA is already enabled")
		return
	}

	secret := op.TOTPGetSecret()
	if secret == "" {
		resp.Error(c, http.StatusBadRequest, "TOTP not initialized, please init first")
		return
	}

	if !op.ValidateTOTP(secret, req.Code) {
		resp.Error(c, http.StatusBadRequest, "Invalid verification code")
		return
	}

	if err := op.TOTPEnable(); err != nil {
		resp.Error(c, http.StatusInternalServerError, resp.ErrInternalServer)
		return
	}

	resp.Success(c, "2FA enabled successfully")
}

// DisableTOTPRequest 禁用 TOTP 请求
type DisableTOTPRequest struct {
	Code string `json:"code"`
}

// disableTOTP 禁用 TOTP
func disableTOTP(c *gin.Context) {
	var req DisableTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}

	if !op.TOTPIsEnabled() {
		resp.Error(c, http.StatusBadRequest, "2FA is not enabled")
		return
	}

	secret := op.TOTPGetSecret()
	if !op.ValidateTOTP(secret, req.Code) {
		resp.Error(c, http.StatusBadRequest, "Invalid verification code")
		return
	}

	if err := op.TOTPDisable(); err != nil {
		resp.Error(c, http.StatusInternalServerError, resp.ErrInternalServer)
		return
	}

	resp.Success(c, "2FA disabled successfully")
}

// getTOTPStatus 获取 TOTP 状态
func getTOTPStatus(c *gin.Context) {
	enabled := op.TOTPIsEnabled()
	resp.Success(c, gin.H{
		"enabled": enabled,
	})
}
