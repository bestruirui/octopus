package middleware

import (
	"net/http"

	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/gin-gonic/gin"
)

// RequirePermission returns a middleware that checks if the authenticated user has the required permission.
func RequirePermission(perm auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("user_role")
		if role == "" {
			role = "admin" // backward-compat: existing users without role get admin
		}
		if !auth.HasPermission(role, perm) {
			resp.Error(c, http.StatusForbidden, "permission denied")
			c.Abort()
			return
		}
		c.Next()
	}
}
