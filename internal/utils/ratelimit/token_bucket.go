package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket implements a simple token bucket rate limiter.
type TokenBucket struct {
	rate       float64 // tokens per second
	burst      float64 // max burst size
	tokens     float64 // current tokens
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a TokenBucket with the given rate (per minute) and burst.
func NewTokenBucket(ratePerMinute int, burst int) *TokenBucket {
	rate := float64(ratePerMinute) / 60.0
	if burst <= 0 {
		burst = ratePerMinute
		if burst <= 0 {
			burst = 1
		}
	}
	return &TokenBucket{
		rate:       rate,
		burst:      float64(burst),
		tokens:     float64(burst),
		lastUpdate: time.Now(),
	}
}

// Allow checks if one token can be consumed. Returns true if allowed.
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN checks if n tokens can be consumed. Returns true if allowed.
func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.burst {
		tb.tokens = tb.burst
	}
	tb.lastUpdate = now

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	return false
}

// TokensRemaining returns the current number of tokens available.
func (tb *TokenBucket) TokensRemaining() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tokens := tb.tokens + elapsed*tb.rate
	if tokens > tb.burst {
		tokens = tb.burst
	}
	return int(tokens)
}

// ResetAt returns the time when the bucket will be fully refilled from empty.
func (tb *TokenBucket) ResetAt() time.Time {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.lastUpdate.Add(time.Duration(tb.burst/tb.rate) * time.Second)
}
