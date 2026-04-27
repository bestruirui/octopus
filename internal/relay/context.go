package relay

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lingyuins/octopus/internal/conf"
	"github.com/lingyuins/octopus/internal/utils/log"
)

const (
	clientDisconnectedLogMessage   = "client disconnected before response completed, continuing generation"
	defaultRelayPersistenceTimeout = 10 * time.Second
)

var relayUpstreamTimeout time.Duration

func init() {
	if raw := strings.TrimSpace(os.Getenv(strings.ToUpper(conf.APP_NAME) + "_RELAY_UPSTREAM_TIMEOUT_SECONDS")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			relayUpstreamTimeout = time.Duration(v) * time.Second
		}
	}
}

func newRelayOperationContext() (context.Context, context.CancelFunc) {
	if relayUpstreamTimeout > 0 {
		return context.WithTimeout(context.Background(), relayUpstreamTimeout)
	}
	return context.WithCancel(context.Background())
}

func newRelayPersistenceContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultRelayPersistenceTimeout)
}

func logRelayErrorfByContext(err error, format string, args ...any) {
	switch {
	case err == nil:
		log.Warnf(format, args...)
	case errors.Is(err, context.Canceled):
		log.Warnf(format, args...)
	case errors.Is(err, context.DeadlineExceeded):
		log.Errorf(format, args...)
	default:
		log.Warnf(format, args...)
	}
}
