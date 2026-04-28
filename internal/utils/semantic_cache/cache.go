package semantic_cache

import (
	"math"
	"sync"
	"time"
)

// CacheEntry holds a cached request/response pair with its embedding.
type CacheEntry struct {
	RequestJSON  string
	ResponseJSON string
	Embedding    []float64
	CreatedAt    time.Time
	LastAccessAt time.Time
	HitCount     int64
}

// SemanticCache is an in-memory vector store with cosine similarity lookup.
type SemanticCache struct {
	mu         sync.RWMutex
	entries    []CacheEntry
	maxEntries int
	threshold  float64
	ttl        time.Duration
	hits       int64
	misses     int64
}

var globalCache *SemanticCache

// Init creates or reconfigures the global semantic cache.
func Init(maxEntries int, threshold float64, ttlSec int) {
	if maxEntries <= 0 {
		globalCache = nil
		return
	}
	if globalCache == nil || globalCache.maxEntries != maxEntries || globalCache.threshold != threshold || globalCache.ttl != time.Duration(ttlSec)*time.Second {
		globalCache = &SemanticCache{
			entries:    make([]CacheEntry, 0, maxEntries),
			maxEntries: maxEntries,
			threshold:  threshold,
			ttl:        time.Duration(ttlSec) * time.Second,
		}
	}
}

// Lookup finds the best matching cache entry for the given embedding.
// Returns the response JSON and true if a match above threshold is found.
func Lookup(embedding []float64) (responseJSON string, found bool) {
	if globalCache == nil {
		return "", false
	}
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	globalCache.pruneExpiredLocked()

	if len(globalCache.entries) == 0 {
		globalCache.misses++
		return "", false
	}

	bestIdx := -1
	bestSim := -1.0
	for i, entry := range globalCache.entries {
		sim := cosineSimilarity(embedding, entry.Embedding)
		if sim > bestSim {
			bestSim = sim
			bestIdx = i
		}
	}

	if bestIdx >= 0 && bestSim >= globalCache.threshold {
		globalCache.entries[bestIdx].HitCount++
		globalCache.entries[bestIdx].LastAccessAt = time.Now()
		globalCache.hits++
		return globalCache.entries[bestIdx].ResponseJSON, true
	}

	globalCache.misses++
	return "", false
}

// Store adds a new entry to the cache. If the cache is full, the oldest entry is evicted.
func Store(requestJSON, responseJSON string, embedding []float64) {
	if globalCache == nil {
		return
	}
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	entry := CacheEntry{
		RequestJSON:  requestJSON,
		ResponseJSON: responseJSON,
		Embedding:    cloneEmbedding(embedding),
		CreatedAt:    time.Now(),
		LastAccessAt: time.Now(),
		HitCount:     1,
	}

	if len(globalCache.entries) >= globalCache.maxEntries {
		// Evict the oldest entry
		oldestIdx := 0
		for i, e := range globalCache.entries {
			if e.LastAccessAt.Before(globalCache.entries[oldestIdx].LastAccessAt) {
				oldestIdx = i
			}
		}
		globalCache.entries[oldestIdx] = entry
	} else {
		globalCache.entries = append(globalCache.entries, entry)
	}
}

// Stats returns hit/miss counts and current cache size.
func Stats() (hits, misses int64, size int) {
	if globalCache == nil {
		return 0, 0, 0
	}
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()
	return globalCache.hits, globalCache.misses, len(globalCache.entries)
}

// Clear empties the cache.
func Clear() {
	if globalCache == nil {
		return
	}
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	globalCache.entries = make([]CacheEntry, 0, globalCache.maxEntries)
	globalCache.hits = 0
	globalCache.misses = 0
}

// Enabled returns true if the semantic cache is initialized and active.
func Enabled() bool {
	return globalCache != nil
}

func (sc *SemanticCache) pruneExpiredLocked() {
	if sc.ttl <= 0 {
		return
	}
	now := time.Now()
	n := 0
	for _, entry := range sc.entries {
		if now.Sub(entry.LastAccessAt) < sc.ttl {
			sc.entries[n] = entry
			n++
		}
	}
	sc.entries = sc.entries[:n]
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func cloneEmbedding(src []float64) []float64 {
	dst := make([]float64, len(src))
	copy(dst, src)
	return dst
}
