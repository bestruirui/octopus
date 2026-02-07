package relay

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

type RelayErrorSource string

const (
	RelayErrorSourceUpstream RelayErrorSource = "upstream"
	RelayErrorSourceNetwork  RelayErrorSource = "network"
	RelayErrorSourceTimeout  RelayErrorSource = "timeout"
	RelayErrorSourceLocal    RelayErrorSource = "local"
)

type RelayError struct {
	StatusCode int
	Source     RelayErrorSource
	Retryable  bool
	Trippable  bool
	Cause      error
	Message    string
}

func (e *RelayError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "relay error"
}

func newRelayError(statusCode int, source RelayErrorSource, retryable, trippable bool, cause error, message string) *RelayError {
	return &RelayError{
		StatusCode: statusCode,
		Source:     source,
		Retryable:  retryable,
		Trippable:  trippable,
		Cause:      cause,
		Message:    message,
	}
}

func classifyTransportError(err error) *RelayError {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return newRelayError(0, RelayErrorSourceTimeout, true, true, err, err.Error())
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return newRelayError(0, RelayErrorSourceTimeout, true, true, err, err.Error())
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout"),
		strings.Contains(msg, "deadline exceeded"),
		strings.Contains(msg, "first token timeout"):
		return newRelayError(0, RelayErrorSourceTimeout, true, true, err, err.Error())
	default:
		return newRelayError(0, RelayErrorSourceNetwork, true, true, err, err.Error())
	}
}

func classifyUpstreamStatus(statusCode int, body string) *RelayError {
	message := fmt.Sprintf("upstream error: %d", statusCode)
	if body != "" {
		message = fmt.Sprintf("upstream error: %d: %s", statusCode, body)
	}
	switch {
	case statusCode == 429:
		return newRelayError(statusCode, RelayErrorSourceUpstream, true, false, nil, message)
	case statusCode >= 500 && statusCode < 600:
		return newRelayError(statusCode, RelayErrorSourceUpstream, true, true, nil, message)
	case statusCode == 401 || statusCode == 403:
		return newRelayError(statusCode, RelayErrorSourceUpstream, false, false, nil, message)
	default:
		return newRelayError(statusCode, RelayErrorSourceUpstream, false, false, nil, message)
	}
}
