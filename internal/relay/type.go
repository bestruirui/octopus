package relay

import (
	"os"
	"strconv"
	"strings"

	"github.com/bestruirui/octopus/internal/conf"
	dbmodel "github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/gin-gonic/gin"
)

// maxSSEEventSize 定义 SSE 事件的最大大小。
// 对于图像生成模型（如 gemini-3-pro-image-preview），返回的 base64 编码图像数据
// 可能非常大（高分辨率图像可能超过 10MB），因此需要设置足够大的缓冲区。
// 默认 32MB，可通过环境变量 OCTOPUS_RELAY_MAX_SSE_EVENT_SIZE 覆盖。
var maxSSEEventSize = 32 * 1024 * 1024

func init() {
	if raw := strings.TrimSpace(os.Getenv(strings.ToUpper(conf.APP_NAME) + "_RELAY_MAX_SSE_EVENT_SIZE")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			maxSSEEventSize = v
		}
	}
}

// hopByHopHeaders 定义不应转发的 HTTP 头
var hopByHopHeaders = map[string]bool{
	"authorization":       true,
	"x-api-key":           true,
	"connection":          true,
	"keep-alive":          true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailer":             true,
	"transfer-encoding":   true,
	"upgrade":             true,
	"content-length":      true,
	"host":                true,
	"accept-encoding":     true,
}

// relayContext 保存请求转发过程中的上下文信息
type relayContext struct {
	c               *gin.Context
	inAdapter       model.Inbound
	outAdapter      model.Outbound
	internalRequest *model.InternalLLMRequest
	channel         *dbmodel.Channel
	metrics         *RelayMetrics

	usedKey dbmodel.ChannelKey

	// firstTokenTimeOutSec: streaming-only "time to first token" timeout for the selected group/channel.
	// When >0 and stream doesn't produce any transformed output within this duration, we abort and retry next channel.
	firstTokenTimeOutSec int
}
