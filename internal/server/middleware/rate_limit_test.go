package middleware

import (
	"testing"
	"time"
)

func TestAllowAPIKeyRequest(t *testing.T) {
	apiKeyWindows.Lock()
	apiKeyWindows.items = make(map[int]apiKeyWindow)
	apiKeyWindows.Unlock()
	now := time.Unix(1000, 0)
	if ok, _ := allowAPIKeyRequest(7, 2, now); !ok {
		t.Fatal("first request should be allowed")
	}
	if ok, _ := allowAPIKeyRequest(7, 2, now.Add(time.Second)); !ok {
		t.Fatal("second request should be allowed")
	}
	if ok, retry := allowAPIKeyRequest(7, 2, now.Add(2*time.Second)); ok || retry < 1 {
		t.Fatalf("third request should be limited, retry=%d", retry)
	}
	if ok, _ := allowAPIKeyRequest(7, 2, now.Add(time.Minute)); !ok {
		t.Fatal("next window should allow a request")
	}
}