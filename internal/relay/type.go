package relay

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/conf"
	dbmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/relay/balancer"
	"github.com/lingyuins/octopus/internal/transformer/model"
)

// maxSSEEventSize 定义 SSE 事件的最大大小。
// 对于图像生成模型（如 gemini-3-pro-image-preview），返回的 base64 编码图像数据
// 可能非常大（高分辨率图像可能超过 10MB），因此需要设置足够大的缓冲区。
// 默认 32MB，可通过环境变量 OCTOPUS_RELAY_MAX_SSE_EVENT_SIZE 覆盖。
var maxSSEEventSize = 32 * 1024 * 1024

const (
	defaultMaxRetryPerCandidate = 3
	defaultRatelimitCooldown    = 300
	defaultMaxTotalAttempts     = 0
)

func init() {
	if raw := strings.TrimSpace(os.Getenv(strings.ToUpper(conf.APP_NAME) + "_RELAY_MAX_SSE_EVENT_SIZE")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			maxSSEEventSize = v
		}
	}
}

func getMaxRetryPerCandidate() int {
	v, err := op.SettingGetInt(dbmodel.SettingKeyRelayRetryCount)
	if err != nil || v < 1 {
		return defaultMaxRetryPerCandidate
	}
	return v
}

func getMaxAttemptsPerCandidate() int {
	return getMaxRetryPerCandidate() + 1
}

func getRatelimitCooldown() int {
	v, err := op.SettingGetInt(dbmodel.SettingKeyRatelimitCooldown)
	if err != nil || v < 0 {
		return defaultRatelimitCooldown
	}
	return v
}

func getMaxTotalAttempts() int {
	v, err := op.SettingGetInt(dbmodel.SettingKeyRelayMaxTotalAttempts)
	if err != nil || v < 0 {
		return defaultMaxTotalAttempts
	}
	return v
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
	"x-forwarded-for":     true,
	"x-forwarded-host":    true,
	"x-forwarded-proto":   true,
	"x-forwarded-port":    true,
	"x-real-ip":           true,
	"forwarded":           true,
	"cf-connecting-ip":    true,
	"true-client-ip":      true,
	"x-client-ip":         true,
	"x-cluster-client-ip": true,
}

type relayRequest struct {
	c               *gin.Context
	inAdapter       model.Inbound
	internalRequest *model.InternalLLMRequest
	metrics         *RelayMetrics
	apiKeyID        int
	requestModel    string
	iter            *balancer.Iterator
}

// relayAttempt 尝试级上下文
type relayAttempt struct {
	*relayRequest // 嵌入请求级上下文

	outAdapter           model.Outbound
	channel              *dbmodel.Channel
	usedKey              dbmodel.ChannelKey
	firstTokenTimeOutSec int
	tryIndex             int
	tryTotal             int
}

// attemptResult 封装单次尝试的结果
type attemptResult struct {
	Success  bool          // 是否成功
	Written  bool          // 流式响应是否已开始写入（不可重试）
	Err      error         // 失败时的错误
	Decision RetryDecision // 重试决策（新增）
}

// RetryScope 重试范围，明确四种行为边界
type RetryScope int

const (
	ScopeNone        RetryScope = iota // 不重试，请求结束（成功或直接失败）
	ScopeSameChannel                   // 同候选换 Key 重试
	ScopeNextChannel                   // 换下一个候选重试
	ScopeAbortAll                      // 停止所有重试（已写入流式响应）
)

func (s RetryScope) String() string {
	switch s {
	case ScopeNone:
		return "none"
	case ScopeSameChannel:
		return "same_channel"
	case ScopeNextChannel:
		return "next_channel"
	case ScopeAbortAll:
		return "abort_all"
	default:
		return "unknown"
	}
}

// RetryDecision 重试决策，包含决策范围、原因和状态码
type RetryDecision struct {
	Scope   RetryScope // 决策范围
	Reason  string     // 决策原因（日志用）
	Code    int        // HTTP 状态码（0 表示非 HTTP 错误）
	IsError bool       // 是否是错误（非成功）
}

// String 返回决策的描述字符串，用于日志
func (d RetryDecision) String() string {
	if d.Reason == "" {
		return d.Scope.String()
	}
	return fmt.Sprintf("%s (%s)", d.Scope.String(), d.Reason)
}

// ClassifyRelayError 根据状态码和错误类型返回重试决策
// 这是错误分类驱动的核心函数，明确区分：
//   - 直接失败（400 类客户端错误）
//   - 换 Key 重试（401/403/429、网络错误）
//   - 换候选重试（404、5xx、超时、上游业务错误）
//   - 停止所有重试（流式响应已部分写出且本次尝试发生错误）
//
// 参数：
//   - statusCode: HTTP 状态码（0 表示非 HTTP 错误）
//   - err: 错误对象（nil 表示成功）
//   - written: 是否已开始向客户端写出响应；仅表示后续不能安全重试，不等同于失败
//
// 返回：RetryDecision 决策结果
func ClassifyRelayError(statusCode int, err error, written bool) RetryDecision {
	// 成功：无需重试。流式成功完成即使已经写出响应，也不应视为失败。
	if err == nil && statusCode >= 200 && statusCode < 300 {
		return RetryDecision{
			Scope:   ScopeNone,
			Reason:  "success",
			Code:    statusCode,
			IsError: false,
		}
	}

	// 流式响应已部分写出且本次尝试发生错误：停止所有重试，避免再次写回客户端。
	if written {
		return RetryDecision{
			Scope:   ScopeAbortAll,
			Reason:  "stream response already written before failure",
			Code:    statusCode,
			IsError: true,
		}
	}

	// HTTP 错误：根据状态码分类
	if statusCode > 0 {
		return classifyHTTPError(statusCode, err)
	}

	// 非 HTTP 错误（网络错误、超时等）
	return classifyNonHTTPError(err)
}

// classifyHTTPError 根据 HTTP 状态码返回重试决策
func classifyHTTPError(statusCode int, err error) RetryDecision {
	switch {
	// 400 Bad Request: 客户端错误，换 channel 也无效
	case statusCode == 400:
		return RetryDecision{
			Scope:   ScopeNone,
			Reason:  "bad request, client error",
			Code:    statusCode,
			IsError: true,
		}

	// 401 Unauthorized: Key 无效，换 key 可能解决
	case statusCode == 401:
		return RetryDecision{
			Scope:   ScopeSameChannel,
			Reason:  "unauthorized, key invalid",
			Code:    statusCode,
			IsError: true,
		}

	// 403 Forbidden: Key 权限问题，换 key
	case statusCode == 403:
		return RetryDecision{
			Scope:   ScopeSameChannel,
			Reason:  "forbidden, key permission issue",
			Code:    statusCode,
			IsError: true,
		}

	// 404 Not Found: 模型/资源不存在，可能其他 channel 有
	case statusCode == 404:
		return RetryDecision{
			Scope:   ScopeNextChannel,
			Reason:  "not found, try next channel",
			Code:    statusCode,
			IsError: true,
		}

	// 429 Rate Limited: 该 key 被限，换 key（cooldown 机制也会生效）
	case statusCode == 429:
		return RetryDecision{
			Scope:   ScopeSameChannel,
			Reason:  "rate limited, try another key",
			Code:    statusCode,
			IsError: true,
		}

	// 500 Internal Server Error: 上游服务问题，换候选
	case statusCode == 500:
		return RetryDecision{
			Scope:   ScopeNextChannel,
			Reason:  "internal server error",
			Code:    statusCode,
			IsError: true,
		}

	// 502 Bad Gateway / 503 Service Unavailable / 504 Gateway Timeout: 上游网关问题，换候选
	case statusCode == 502 || statusCode == 503 || statusCode == 504:
		return RetryDecision{
			Scope:   ScopeNextChannel,
			Reason:  fmt.Sprintf("gateway error %d", statusCode),
			Code:    statusCode,
			IsError: true,
		}

	// 其他 2xx 状态码但有错误（如 transformer 错误）：换候选
	case statusCode >= 200 && statusCode < 300 && err != nil:
		return RetryDecision{
			Scope:   ScopeNextChannel,
			Reason:  "transformer error on success response",
			Code:    statusCode,
			IsError: true,
		}

	// 其他状态码：保守策略，换候选
	default:
		return RetryDecision{
			Scope:   ScopeNextChannel,
			Reason:  fmt.Sprintf("unexpected status code %d", statusCode),
			Code:    statusCode,
			IsError: true,
		}
	}
}

// classifyNonHTTPError 根据非 HTTP 错误类型返回重试决策
func classifyNonHTTPError(err error) RetryDecision {
	if err == nil {
		// 无错误但状态码无效，保守处理
		return RetryDecision{
			Scope:   ScopeNextChannel,
			Reason:  "unknown error",
			Code:    0,
			IsError: true,
		}
	}

	// 超时错误：上游响应慢，换候选
	if isTimeoutError(err) {
		return RetryDecision{
			Scope:   ScopeNextChannel,
			Reason:  "timeout",
			Code:    0,
			IsError: true,
		}
	}

	// 网络连接错误：连接失败，换 key 可能解决（不同 key 可能走不同路径）
	if isNetworkError(err) {
		return RetryDecision{
			Scope:   ScopeSameChannel,
			Reason:  "network error",
			Code:    0,
			IsError: true,
		}
	}

	// 其他错误：保守策略，换候选
	return RetryDecision{
		Scope:   ScopeNextChannel,
		Reason:  err.Error(),
		Code:    0,
		IsError: true,
	}
}

// isTimeoutError 判断是否为超时错误
func isTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// 常见超时错误字符串匹配
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "first token timeout")
}

// isNetworkError 判断是否为网络连接错误
func isNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		// 超时错误已在 isTimeoutError 中处理，这里排除
		if netErr.Timeout() {
			return false
		}
		return true
	}
	// 常见网络错误字符串匹配
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "DNS") ||
		strings.Contains(errStr, "failed to send request")
}
