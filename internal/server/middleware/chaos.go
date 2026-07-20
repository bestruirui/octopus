package middleware

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/gin-gonic/gin"
)

// Chaos 混沌工程中间件：仅作用于 /v1 转发路径。
// 默认关闭；开启后可注入延迟 / 随机 503 / 直接断开。
// 用于验证熔断、探活、故障迁移是否按预期工作，勿在生产常开。
func Chaos() gin.HandlerFunc {
	return func(c *gin.Context) {
		enabled, err := op.SettingGetBool(model.SettingKeyChaosEnabled)
		if err != nil || !enabled {
			c.Next()
			return
		}

		// 丢包：直接 Abort，不写任何响应（模拟连接被掐）
		if rate, err := op.SettingGetInt(model.SettingKeyChaosDropRate); err == nil && rate > 0 {
			if rand.Intn(100) < rate {
				log.Warnf("chaos: drop request path=%s", c.Request.URL.Path)
				// 强制断开：劫持底层连接并关闭
				if hj, ok := c.Writer.(http.Hijacker); ok {
					if conn, _, err := hj.Hijack(); err == nil {
						_ = conn.Close()
						c.Abort()
						return
					}
				}
				c.Abort()
				return
			}
		}

		// 错误注入：随机 503
		if rate, err := op.SettingGetInt(model.SettingKeyChaosErrorRate); err == nil && rate > 0 {
			if rand.Intn(100) < rate {
				log.Warnf("chaos: inject 503 path=%s", c.Request.URL.Path)
				resp.Error(c, http.StatusServiceUnavailable, "chaos injected error")
				c.Abort()
				return
			}
		}

		// 延迟注入
		if delayMS, err := op.SettingGetInt(model.SettingKeyChaosDelayMS); err == nil && delayMS > 0 {
			log.Debugf("chaos: delay %dms path=%s", delayMS, c.Request.URL.Path)
			select {
			case <-time.After(time.Duration(delayMS) * time.Millisecond):
			case <-c.Request.Context().Done():
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
