package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// KeyRateLimiter 按 API Key ID 做令牌桶限流
// 限流参数在 APIKey 表的 RateLimit 字段（每分钟请求数，0=不限）
type keyRateLimiter struct {
	limiters sync.Map // key: int (apiKeyID) -> value: *rateEntry
}

type rateEntry struct {
	limiter *rate.Limiter
	rpm     int       // 每分钟请求数
	lastUse time.Time // 最后使用时间，用于 GC
}

var globalKeyRL = &keyRateLimiter{}

// init 启动定期清理
func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			globalKeyRL.gc()
		}
	}()
}

// Allow 检查 API Key 是否被限流
func Allow(apiKeyID int, rpm int) bool {
	if rpm <= 0 {
		return true // 不限流
	}

	entry := globalKeyRL.getOrCreate(apiKeyID, rpm)
	entry.lastUse = time.Now()

	// rate.Limiter.Allow 非阻塞：有令牌返回 true，无则 false
	return entry.limiter.Allow()
}

// RateLimit API Key 限流中间件
// 需在 APIKeyAuth 之后使用，从 context 中读取 api_key_id
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKeyID := c.GetInt("api_key_id")
		if apiKeyID == 0 {
			c.Next()
			return
		}
		// 从 context 读 rpm（由 APIKeyAuth 写入）
		rpm := c.GetInt("rate_limit_rpm")
		if rpm > 0 && !Allow(apiKeyID, rpm) {
			c.Header("Retry-After", strconv.Itoa(60/rpm))
			resp.Error(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (rl *keyRateLimiter) getOrCreate(apiKeyID int, rpm int) *rateEntry {
	if v, ok := rl.limiters.Load(apiKeyID); ok {
		entry := v.(*rateEntry)
		// 如果 rpm 变了，重建 limiter
		if entry.rpm != rpm {
			newEntry := &rateEntry{
				limiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(rpm)), rpm),
				rpm:     rpm,
				lastUse: time.Now(),
			}
			rl.limiters.Store(apiKeyID, newEntry)
			return newEntry
		}
		return entry
	}

	entry := &rateEntry{
		limiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(rpm)), rpm),
		rpm:     rpm,
		lastUse: time.Now(),
	}
	actual, _ := rl.limiters.LoadOrStore(apiKeyID, entry)
	return actual.(*rateEntry)
}

// gc 清理 10 分钟未使用的 limiter，防止内存泄漏
func (rl *keyRateLimiter) gc() {
	cutoff := time.Now().Add(-10 * time.Minute)
	rl.limiters.Range(func(key, value any) bool {
		entry := value.(*rateEntry)
		if entry.lastUse.Before(cutoff) {
			rl.limiters.Delete(key)
		}
		return true
	})
}
