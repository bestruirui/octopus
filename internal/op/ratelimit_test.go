package op

import "testing"

func TestRateLimitCheck(t *testing.T) {
	const keyID = 987654321
	RateLimitDel(keyID)
	t.Cleanup(func() { RateLimitDel(keyID) })

	if allowed, retryAfter := RateLimitCheck(keyID, 2); !allowed || retryAfter != 0 {
		t.Fatalf("first request: allowed=%v retryAfter=%d", allowed, retryAfter)
	}
	if allowed, retryAfter := RateLimitCheck(keyID, 2); !allowed || retryAfter != 0 {
		t.Fatalf("second request: allowed=%v retryAfter=%d", allowed, retryAfter)
	}
	if allowed, retryAfter := RateLimitCheck(keyID, 2); allowed || retryAfter < 1 || retryAfter > 60 {
		t.Fatalf("third request: allowed=%v retryAfter=%d", allowed, retryAfter)
	}
	if allowed, retryAfter := RateLimitCheck(keyID, 0); !allowed || retryAfter != 0 {
		t.Fatalf("unlimited request: allowed=%v retryAfter=%d", allowed, retryAfter)
	}
}
