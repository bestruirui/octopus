package balancer

import (
	"fmt"
	"sync"
	"time"
)

type compactUnsupportedEntry struct {
	Timestamp time.Time
}

var (
	compactUnsupportedCache sync.Map
	compactUnsupportedTTL   = defaultCompactUnsupportedTTL
)

const defaultCompactUnsupportedTTL = 10 * time.Minute

func compactUnsupportedKey(channelID int, modelName string) string {
	return fmt.Sprintf("%d:%s", channelID, modelName)
}

// IsCompactUnsupported reports whether channel/model is temporarily marked as
// not supporting /responses/compact.
func IsCompactUnsupported(channelID int, modelName string) (unsupported bool, remaining time.Duration) {
	key := compactUnsupportedKey(channelID, modelName)
	v, ok := compactUnsupportedCache.Load(key)
	if !ok {
		return false, 0
	}
	entry := v.(*compactUnsupportedEntry)
	elapsed := time.Since(entry.Timestamp)
	if elapsed >= compactUnsupportedTTL {
		compactUnsupportedCache.Delete(key)
		return false, 0
	}
	return true, compactUnsupportedTTL - elapsed
}

// MarkCompactUnsupported marks channel/model as temporarily incompatible with
// /responses/compact.
func MarkCompactUnsupported(channelID int, modelName string) {
	key := compactUnsupportedKey(channelID, modelName)
	compactUnsupportedCache.Store(key, &compactUnsupportedEntry{
		Timestamp: time.Now(),
	})
}

// MarkCompactSupported clears temporary incompatibility marker.
func MarkCompactSupported(channelID int, modelName string) {
	key := compactUnsupportedKey(channelID, modelName)
	compactUnsupportedCache.Delete(key)
}
