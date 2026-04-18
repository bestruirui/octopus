package op

import (
	"context"
	"testing"
	"time"
)

func TestClampAIRouteParallelism(t *testing.T) {
	if got := clampAIRouteParallelism(0, 5); got != defaultAIRouteParallelism {
		t.Fatalf("clampAIRouteParallelism(0, 5) = %d, want %d", got, defaultAIRouteParallelism)
	}
	if got := clampAIRouteParallelism(9, 10); got != maxAIRouteParallelism {
		t.Fatalf("clampAIRouteParallelism(9, 10) = %d, want %d", got, maxAIRouteParallelism)
	}
	if got := clampAIRouteParallelism(4, 2); got != 2 {
		t.Fatalf("clampAIRouteParallelism(4, 2) = %d, want 2", got)
	}
}

func TestAIRouteServicePoolAcquireNextAndCooldown(t *testing.T) {
	pool := newAIRouteServicePool([]aiRouteService{
		{Index: 0, Name: "svc-a"},
		{Index: 1, Name: "svc-b"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	first, err := pool.Acquire(ctx, aiRouteServiceHint{BucketIndex: 1}, nil)
	if err != nil {
		t.Fatalf("Acquire() error = %v, want nil", err)
	}
	if first.Service.Name != "svc-a" {
		t.Fatalf("Acquire() service = %q, want %q", first.Service.Name, "svc-a")
	}

	pool.Release(first, aiRouteServiceOutcome{
		Retryable: true,
		Cooldown:  50 * time.Millisecond,
		Err:       context.DeadlineExceeded,
	})

	next, err := pool.Next(ctx, first, aiRouteServiceHint{BucketIndex: 1}, map[int]struct{}{first.Index: {}}, context.DeadlineExceeded)
	if err != nil {
		t.Fatalf("Next() error = %v, want nil", err)
	}
	if next.Service.Name != "svc-b" {
		t.Fatalf("Next() service = %q, want %q", next.Service.Name, "svc-b")
	}
	pool.Release(next, aiRouteServiceOutcome{Success: true})
}
