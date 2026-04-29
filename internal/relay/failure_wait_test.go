package relay

import (
	"context"
	"testing"
	"time"
)

func TestWaitBeforeAllChannelsFailed_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := waitBeforeAllChannelsFailed(ctx)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatal("expected cancelled wait to return quickly")
	}
}
