package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/gin-gonic/gin"
)

type apiKeyWindow struct {
	started time.Time
	count   int
}

var apiKeyWindows = struct {
	sync.Mutex
	items map[int]apiKeyWindow
}{items: make(map[int]apiKeyWindow)}

func allowAPIKeyRequest(id, limit int, now time.Time) (bool, int) {
	if limit <= 0 {
		return true, 0
	}
	window := apiKeyWindows.items[id]
	if window.started.IsZero() || now.Sub(window.started) >= time.Minute {
		window = apiKeyWindow{started: now}
	}
	if window.count >= limit {
		retryAfter := int(window.started.Add(time.Minute).Sub(now).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		return false, retryAfter
	}
	window.count++
	apiKeyWindows.items[id] = window
	return true, 0
}

// APIKeyRateLimit enforces a fixed requests-per-minute window for one API key.
// A zero limit means unlimited, preserving compatibility with existing keys.
func APIKeyRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := c.GetInt("api_key_requests_per_minute")
		if limit <= 0 {
			c.Next()
			return
		}

		id := c.GetInt("api_key_id")
		now := time.Now()
		apiKeyWindows.Lock()
		allowed, retryAfter := allowAPIKeyRequest(id, limit, now)
		if !allowed {
			apiKeyWindows.Unlock()
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			resp.Error(c, http.StatusTooManyRequests, "API key rate limit exceeded")
			c.Abort()
			return
		}
		apiKeyWindows.Unlock()
		c.Next()
	}
}
