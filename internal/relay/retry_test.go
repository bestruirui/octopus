package relay

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestClassifyRelayError_Success(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		written    bool
		wantScope  RetryScope
		wantError  bool
	}{
		{
			name:       "success 200",
			statusCode: 200,
			err:        nil,
			written:    false,
			wantScope:  ScopeNone,
			wantError:  false,
		},
		{
			name:       "success 201",
			statusCode: 201,
			err:        nil,
			written:    false,
			wantScope:  ScopeNone,
			wantError:  false,
		},
		{
			name:       "success 299",
			statusCode: 299,
			err:        nil,
			written:    false,
			wantScope:  ScopeNone,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyRelayError(tt.statusCode, tt.err, tt.written)
			if got.Scope != tt.wantScope {
				t.Errorf("ClassifyRelayError() Scope = %v, want %v", got.Scope, tt.wantScope)
			}
			if got.IsError != tt.wantError {
				t.Errorf("ClassifyRelayError() IsError = %v, want %v", got.IsError, tt.wantError)
			}
		})
	}
}

func TestClassifyRelayError_Written(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		written    bool
		wantScope  RetryScope
		wantError  bool
	}{
		{
			name:       "written with error",
			statusCode: 500,
			err:        errors.New("some error"),
			written:    true,
			wantScope:  ScopeAbortAll,
			wantError:  true,
		},
		{
			name:       "written with success",
			statusCode: 200,
			err:        nil,
			written:    true,
			wantScope:  ScopeNone,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyRelayError(tt.statusCode, tt.err, tt.written)
			if got.Scope != tt.wantScope {
				t.Errorf("ClassifyRelayError() Scope = %v, want %v", got.Scope, tt.wantScope)
			}
			if got.IsError != tt.wantError {
				t.Errorf("ClassifyRelayError() IsError = %v, want %v", got.IsError, tt.wantError)
			}
		})
	}
}

func TestClassifyRelayError_HTTP4xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		wantScope  RetryScope
		wantReason string
	}{
		{
			name:       "400 bad request",
			statusCode: 400,
			err:        errors.New("bad request"),
			wantScope:  ScopeNone,
			wantReason: "bad request",
		},
		{
			name:       "401 unauthorized",
			statusCode: 401,
			err:        errors.New("unauthorized"),
			wantScope:  ScopeSameChannel,
			wantReason: "unauthorized",
		},
		{
			name:       "403 forbidden",
			statusCode: 403,
			err:        errors.New("forbidden"),
			wantScope:  ScopeSameChannel,
			wantReason: "forbidden",
		},
		{
			name:       "404 not found",
			statusCode: 404,
			err:        errors.New("not found"),
			wantScope:  ScopeNextChannel,
			wantReason: "not found",
		},
		{
			name:       "429 rate limited",
			statusCode: 429,
			err:        errors.New("rate limited"),
			wantScope:  ScopeSameChannel,
			wantReason: "rate limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyRelayError(tt.statusCode, tt.err, false)
			if got.Scope != tt.wantScope {
				t.Errorf("ClassifyRelayError() Scope = %v, want %v", got.Scope, tt.wantScope)
			}
			if !got.IsError {
				t.Errorf("ClassifyRelayError() IsError should be true for %d", tt.statusCode)
			}
		})
	}
}

func TestClassifyRelayError_HTTP5xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		wantScope  RetryScope
	}{
		{
			name:       "500 internal server error",
			statusCode: 500,
			err:        errors.New("internal error"),
			wantScope:  ScopeNextChannel,
		},
		{
			name:       "502 bad gateway",
			statusCode: 502,
			err:        errors.New("bad gateway"),
			wantScope:  ScopeNextChannel,
		},
		{
			name:       "503 service unavailable",
			statusCode: 503,
			err:        errors.New("service unavailable"),
			wantScope:  ScopeNextChannel,
		},
		{
			name:       "504 gateway timeout",
			statusCode: 504,
			err:        errors.New("gateway timeout"),
			wantScope:  ScopeNextChannel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyRelayError(tt.statusCode, tt.err, false)
			if got.Scope != tt.wantScope {
				t.Errorf("ClassifyRelayError() Scope = %v, want %v", got.Scope, tt.wantScope)
			}
			if !got.IsError {
				t.Errorf("ClassifyRelayError() IsError should be true for %d", tt.statusCode)
			}
		})
	}
}

func TestClassifyRelayError_NetworkErrors(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantScope RetryScope
	}{
		{
			name:      "timeout error",
			err:       &timeoutError{},
			wantScope: ScopeNextChannel,
		},
		{
			name:      "connection refused",
			err:       errors.New("connection refused"),
			wantScope: ScopeSameChannel,
		},
		{
			name:      "connection reset",
			err:       errors.New("connection reset by peer"),
			wantScope: ScopeSameChannel,
		},
		{
			name:      "network unreachable",
			err:       errors.New("network is unreachable"),
			wantScope: ScopeSameChannel,
		},
		{
			name:      "DNS error",
			err:       errors.New("no such host"),
			wantScope: ScopeSameChannel,
		},
		{
			name:      "generic error",
			err:       errors.New("some unknown error"),
			wantScope: ScopeNextChannel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyRelayError(0, tt.err, false)
			if got.Scope != tt.wantScope {
				t.Errorf("ClassifyRelayError() Scope = %v, want %v", got.Scope, tt.wantScope)
			}
			if !got.IsError {
				t.Errorf("ClassifyRelayError() IsError should be true")
			}
		})
	}
}

func TestClassifyRelayError_TransformerError(t *testing.T) {
	// 200 status code but with error (transformer error)
	got := ClassifyRelayError(200, errors.New("transformer failed"), false)
	if got.Scope != ScopeNextChannel {
		t.Errorf("ClassifyRelayError() Scope = %v, want %v", got.Scope, ScopeNextChannel)
	}
	if !got.IsError {
		t.Errorf("ClassifyRelayError() IsError should be true for transformer error")
	}
}

func TestRelayAttemptForward_ReturnsUpstreamStatusCodeOnError(t *testing.T) {
	ra := &relayAttempt{}
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       io.NopCloser(strings.NewReader("rate limited")),
	}

	statusCode, err := ra.handleForwardResponse(resp)
	if statusCode != http.StatusTooManyRequests {
		t.Fatalf("handleForwardResponse() statusCode = %d, want %d", statusCode, http.StatusTooManyRequests)
	}
	if err == nil {
		t.Fatal("handleForwardResponse() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Fatalf("handleForwardResponse() error = %q, want to contain 429", err.Error())
	}
}

func TestGetMaxAttemptsPerCandidate_Default(t *testing.T) {
	got := getMaxAttemptsPerCandidate()
	want := defaultMaxRetryPerCandidate + 1
	if got != want {
		t.Fatalf("getMaxAttemptsPerCandidate() = %d, want %d", got, want)
	}
}

func TestRetryScope_String(t *testing.T) {
	tests := []struct {
		scope RetryScope
		want  string
	}{
		{ScopeNone, "none"},
		{ScopeSameChannel, "same_channel"},
		{ScopeNextChannel, "next_channel"},
		{ScopeAbortAll, "abort_all"},
		{RetryScope(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.scope.String(); got != tt.want {
				t.Errorf("RetryScope.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryAllowed(t *testing.T) {
	tests := []struct {
		name         string
		decision     RetryDecision
		wantContinue bool
		wantSwitch   bool
	}{
		{
			name:         "ScopeNone - no retry",
			decision:     RetryDecision{Scope: ScopeNone},
			wantContinue: false,
			wantSwitch:   false,
		},
		{
			name:         "ScopeSameChannel - retry same channel",
			decision:     RetryDecision{Scope: ScopeSameChannel},
			wantContinue: true,
			wantSwitch:   false,
		},
		{
			name:         "ScopeNextChannel - switch channel",
			decision:     RetryDecision{Scope: ScopeNextChannel},
			wantContinue: true,
			wantSwitch:   true,
		},
		{
			name:         "ScopeAbortAll - stop all",
			decision:     RetryDecision{Scope: ScopeAbortAll},
			wantContinue: false,
			wantSwitch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			continueRetry, switchChannel := IsRetryAllowed(tt.decision)
			if continueRetry != tt.wantContinue {
				t.Errorf("IsRetryAllowed() continueRetry = %v, want %v", continueRetry, tt.wantContinue)
			}
			if switchChannel != tt.wantSwitch {
				t.Errorf("IsRetryAllowed() switchChannel = %v, want %v", switchChannel, tt.wantSwitch)
			}
		})
	}
}

// timeoutError implements net.Error with Timeout() returning true
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

var _ net.Error = &timeoutError{}
