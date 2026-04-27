package relay

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewRelayOperationContextDoesNotTrackRequestCancel(t *testing.T) {
	originalTimeout := relayUpstreamTimeout
	relayUpstreamTimeout = 0
	defer func() {
		relayUpstreamTimeout = originalTimeout
	}()

	requestCtx, cancelRequest := context.WithCancel(context.Background())
	defer cancelRequest()

	ctx, cancel := newRelayOperationContext()
	defer cancel()

	cancelRequest()

	select {
	case <-ctx.Done():
		t.Fatalf("newRelayOperationContext() err = %v, want request cancellation to be ignored", ctx.Err())
	case <-time.After(50 * time.Millisecond):
	}

	if !errors.Is(requestCtx.Err(), context.Canceled) {
		t.Fatalf("requestCtx err = %v, want %v", requestCtx.Err(), context.Canceled)
	}
}

func TestNewRelayOperationContextCancelFuncCancelsContext(t *testing.T) {
	originalTimeout := relayUpstreamTimeout
	relayUpstreamTimeout = 0
	defer func() {
		relayUpstreamTimeout = originalTimeout
	}()

	ctx, cancel := newRelayOperationContext()
	cancel()

	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("newRelayOperationContext() err = %v, want %v", ctx.Err(), context.Canceled)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("newRelayOperationContext() cancel did not stop context")
	}
}

func TestNewRelayOperationContextAppliesIndependentTimeout(t *testing.T) {
	originalTimeout := relayUpstreamTimeout
	relayUpstreamTimeout = 20 * time.Millisecond
	defer func() {
		relayUpstreamTimeout = originalTimeout
	}()

	ctx, cancel := newRelayOperationContext()
	defer cancel()

	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("newRelayOperationContext() err = %v, want %v", ctx.Err(), context.DeadlineExceeded)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("newRelayOperationContext() did not time out")
	}
}
