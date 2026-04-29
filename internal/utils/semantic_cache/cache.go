package semantic_cache

import (
	"math"
	"sync"
	"time"
)

// CacheEntry holds a cached request/response pair with its embedding.
type CacheEntry struct {
	Namespace    string
	RequestKey   string
	ResponseJSON []byte
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

type RuntimeConfig struct {
	Enabled          bool
	MaxEntries       int
	Threshold        float64
	TTL              time.Duration
	EmbeddingBaseURL string
	EmbeddingAPIKey  string
	EmbeddingModel   string
	EmbeddingTimeout time.Duration
}

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

// ApplyRuntimeConfig creates or reconfigures the global semantic cache from runtime settings.
func ApplyRuntimeConfig(cfg RuntimeConfig) {
	if !cfg.Enabled || cfg.MaxEntries <= 0 {
		Reset()
		return
	}

	ttl := cfg.TTL
	globalCache = &SemanticCache{
		entries:    make([]CacheEntry, 0, cfg.MaxEntries),
		maxEntries: cfg.MaxEntries,
		threshold:  cfg.Threshold,
		ttl:        ttl,
	}
}

// Reset clears the cache and runtime configuration.
func Reset() {
	globalCache = nil
}

// Lookup finds the best matching cache entry for the given embedding.
// Returns the response JSON and true if a match above threshold is found.
func Lookup(namespace string, embedding []float64) (responseJSON []byte, found bool) {
	if globalCache == nil {
		return nil, false
	}
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.pruneExpiredLocked()

	if len(globalCache.entries) == 0 {
		globalCache.misses++
		return nil, false
	}

	bestIdx := -1
	bestSim := -1.0
	for i, entry := range globalCache.entries {
		if entry.Namespace != namespace {
			continue
		}
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
		return append([]byte(nil), globalCache.entries[bestIdx].ResponseJSON...), true
	}

	globalCache.misses++
	return nil, false
}

// Store adds a new entry to the cache. If the cache is full, the oldest entry is evicted.
func Store(namespace, requestKey string, responseJSON []byte, embedding []float64) {
	if globalCache == nil {
		return
	}
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	entry := CacheEntry{
		Namespace:    namespace,
		RequestKey:   requestKey,
		ResponseJSON: append([]byte(nil), responseJSON...),
		Embedding:    cloneEmbedding(embedding),
		CreatedAt:    time.Now(),
		LastAccessAt: time.Now(),
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
