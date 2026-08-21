package relay

import (
	"errors"
	"net/http"
	"testing"

	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

// TestUpstreamErrorStatusCode 验证从不同错误链形态中提取上游 HTTP 状态码。
func TestUpstreamErrorStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		ok         bool
	}{
		{
			name:       "裸 httpclient.Error",
			err:        &httpclient.Error{StatusCode: http.StatusBadRequest},
			statusCode: http.StatusBadRequest,
			ok:         true,
		},
		{
			name:       "裸 llm.ResponseError",
			err:        &llm.ResponseError{StatusCode: http.StatusUnprocessableEntity},
			statusCode: http.StatusUnprocessableEntity,
			ok:         true,
		},
		{
			name:       "UpstreamError 包装的 llm.ResponseError",
			err:        wrapUpstreamErrorForTest(&llm.ResponseError{StatusCode: http.StatusNotFound}),
			statusCode: http.StatusNotFound,
			ok:         true,
		},
		{
			name: "普通错误",
			err:  errors.New("connection refused"),
		},
		{
			name: "无状态码的 httpclient.Error",
			err:  &httpclient.Error{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := upstreamErrorStatusCode(tt.err)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if code != tt.statusCode {
				t.Fatalf("statusCode = %d, want %d", code, tt.statusCode)
			}
		})
	}
}

// TestNonRetryableStatusCodes 验证不可重试状态码集合的边界。
func TestNonRetryableStatusCodes(t *testing.T) {
	nonRetryable := []int{400, 405, 406, 413, 414, 415, 422, 501}
	for _, code := range nonRetryable {
		if !nonRetryableStatusCodes[code] {
			t.Errorf("status %d should be non-retryable", code)
		}
	}
	// 可重试：网络类无状态码、429 限流、401/403 换渠道可能恢复、5xx 服务端故障。
	retryable := []int{401, 403, 404, 408, 429, 500, 502, 503, 504}
	for _, code := range retryable {
		if nonRetryableStatusCodes[code] {
			t.Errorf("status %d should stay retryable", code)
		}
	}
}

// wrapUpstreamErrorForTest 复刻 pipeline.UpstreamError 的包装形态：
// 该类型实现 Unwrap，errors.As 应能穿透提取内部错误的状态码。
type testUpstreamError struct {
	err error
}

func (e *testUpstreamError) Error() string { return e.err.Error() }
func (e *testUpstreamError) Unwrap() error { return e.err }

func wrapUpstreamErrorForTest(err error) error {
	return &testUpstreamError{err: err}
}
