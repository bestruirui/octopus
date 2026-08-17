package op

import (
	"sync"
	"time"
)

type rateLimitEntry struct {
	mu     sync.Mutex
	minute int64
	count  int64
}

var rateLimitEntries = struct {
	sync.Mutex
	items map[int]*rateLimitEntry
}{items: make(map[int]*rateLimitEntry)}

// RateLimitCheck applies a fixed one-minute request window to one API key.
// A non-positive limit means unlimited.
func RateLimitCheck(apiKeyID int, maxRPM int) (bool, int) {
	if maxRPM <= 0 {
		return true, 0
	}

	rateLimitEntries.Lock()
	entry := rateLimitEntries.items[apiKeyID]
	if entry == nil {
		entry = &rateLimitEntry{}
		rateLimitEntries.items[apiKeyID] = entry
	}
	rateLimitEntries.Unlock()

	now := time.Now()
	currentMinute := now.Unix() / 60
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.minute != currentMinute {
		entry.minute = currentMinute
		entry.count = 0
	}
	entry.count++
	if entry.count <= int64(maxRPM) {
		return true, 0
	}
	retryAfter := 60 - int(now.Unix()%60)
	if retryAfter < 1 {
		retryAfter = 1
	}
	return false, retryAfter
}

func RateLimitDel(apiKeyID int) {
	rateLimitEntries.Lock()
	delete(rateLimitEntries.items, apiKeyID)
	rateLimitEntries.Unlock()
}
