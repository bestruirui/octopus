package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/conf"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/auth"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("auth")
		if err != nil || token == "" {
			resp.Error(c, http.StatusUnauthorized, resp.ErrUnauthorized)
			c.Abort()
			return
		}
		if !auth.VerifyJWTToken(token) {
			c.SetCookie("auth", "", -1, "/", "", false, false)
			resp.Error(c, http.StatusUnauthorized, resp.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Next()
	}
}

func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var apiKey string

		if key := c.Request.Header.Get("x-api-key"); key != "" {
			apiKey = key
		} else if authorization := c.Request.Header.Get("Authorization"); authorization != "" {
			apiKey = strings.TrimPrefix(authorization, "Bearer ")
		}

		if apiKey == "" {
			resp.Error(c, http.StatusUnauthorized, resp.ErrUnauthorized)
			c.Abort()
			return
		}

		if !strings.HasPrefix(apiKey, "sk-"+conf.APP_NAME+"-") {
			resp.Error(c, http.StatusUnauthorized, resp.ErrUnauthorized)
			c.Abort()
			return
		}
		apiKeyObj, err := op.APIKeyGetByAPIKey(apiKey, c.Request.Context())
		if err != nil {
			resp.Error(c, http.StatusUnauthorized, resp.ErrUnauthorized)
			c.Abort()
			return
		}
		if !apiKeyObj.Enabled {
			resp.Error(c, http.StatusUnauthorized, "API key is disabled")
			c.Abort()
			return
		}
		if apiKeyObj.ExpireAt > 0 && apiKeyObj.ExpireAt < time.Now().Unix() {
			resp.Error(c, http.StatusUnauthorized, "API key has expired")
			c.Abort()
			return
		}
		if apiKeyObj.MaxRPM > 0 {
			allowed, retryAfter := op.RateLimitCheck(apiKeyObj.ID, apiKeyObj.MaxRPM)
			if !allowed {
				c.Header("Retry-After", strconv.Itoa(retryAfter))
				resp.Error(c, http.StatusTooManyRequests, "API key has exceeded the rate limit")
				c.Abort()
				return
			}
		}
		statsAPIKey := op.StatsAPIKeyGet(apiKeyObj.ID)
		if apiKeyObj.MaxCost > 0 && apiKeyObj.MaxCost < statsAPIKey.StatsMetrics.OutputCost+statsAPIKey.StatsMetrics.InputCost {
			resp.Error(c, http.StatusUnauthorized, "API key has reached the max cost")
			c.Abort()
			return
		}
		c.Set("supported_models", apiKeyObj.SupportedModels)
		c.Set("api_key_id", apiKeyObj.ID)
		c.Next()
	}
}
