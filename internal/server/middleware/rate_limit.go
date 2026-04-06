package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/gin-gonic/gin"
)

const (
	loginRateLimitWindow    = 10 * time.Minute
	loginRateLimitMaxFailed = 5
)

type loginAttempt struct {
	FailedCount int
	BlockedUntil time.Time
	LastFailedAt time.Time
}

var loginAttemptCache = struct {
	sync.Mutex
	items map[string]*loginAttempt
}{
	items: make(map[string]*loginAttempt),
}

func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if key == "" {
			key = c.RemoteIP()
		}
		if isLoginBlocked(key, time.Now()) {
			resp.Error(c, http.StatusTooManyRequests, resp.ErrTooManyRequests)
			c.Abort()
			return
		}
		c.Set("login_rate_limit_key", key)
		c.Next()
	}
}

func RecordLoginFailure(key string, now time.Time) {
	if key == "" {
		return
	}

	loginAttemptCache.Lock()
	defer loginAttemptCache.Unlock()

	attempt, ok := loginAttemptCache.items[key]
	if !ok || now.Sub(attempt.LastFailedAt) > loginRateLimitWindow {
		attempt = &loginAttempt{}
		loginAttemptCache.items[key] = attempt
	}

	attempt.FailedCount++
	attempt.LastFailedAt = now
	if attempt.FailedCount >= loginRateLimitMaxFailed {
		attempt.BlockedUntil = now.Add(loginRateLimitWindow)
	}
}

func ClearLoginFailures(key string) {
	if key == "" {
		return
	}
	loginAttemptCache.Lock()
	delete(loginAttemptCache.items, key)
	loginAttemptCache.Unlock()
}

func isLoginBlocked(key string, now time.Time) bool {
	if key == "" {
		return false
	}

	loginAttemptCache.Lock()
	defer loginAttemptCache.Unlock()

	attempt, ok := loginAttemptCache.items[key]
	if !ok {
		return false
	}
	if !attempt.BlockedUntil.IsZero() && now.Before(attempt.BlockedUntil) {
		return true
	}
	if now.Sub(attempt.LastFailedAt) > loginRateLimitWindow {
		delete(loginAttemptCache.items, key)
		return false
	}
	return false
}
