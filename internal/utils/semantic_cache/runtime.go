package semantic_cache

import "sync/atomic"

type RuntimeStats struct {
	EvaluatedRequests int64
	CacheHitResponses int64
	CacheMissRequests int64
	BypassedRequests  int64
	StoredResponses   int64
}

var runtimeStats struct {
	evaluatedRequests atomic.Int64
	cacheHitResponses atomic.Int64
	cacheMissRequests atomic.Int64
	bypassedRequests  atomic.Int64
	storedResponses   atomic.Int64
}

func RecordEvaluated() {
	runtimeStats.evaluatedRequests.Add(1)
}

func RecordHit() {
	runtimeStats.cacheHitResponses.Add(1)
}

func RecordMiss() {
	runtimeStats.cacheMissRequests.Add(1)
}

func RecordBypass() {
	runtimeStats.bypassedRequests.Add(1)
}

func RecordStored() {
	runtimeStats.storedResponses.Add(1)
}

func GetRuntimeStats() RuntimeStats {
	return RuntimeStats{
		EvaluatedRequests: runtimeStats.evaluatedRequests.Load(),
		CacheHitResponses: runtimeStats.cacheHitResponses.Load(),
		CacheMissRequests: runtimeStats.cacheMissRequests.Load(),
		BypassedRequests:  runtimeStats.bypassedRequests.Load(),
		StoredResponses:   runtimeStats.storedResponses.Load(),
	}
}

func RuntimeEnabled() bool {
	return Enabled()
}

func ResetRuntimeStats() {
	runtimeStats.evaluatedRequests.Store(0)
	runtimeStats.cacheHitResponses.Store(0)
	runtimeStats.cacheMissRequests.Store(0)
	runtimeStats.bypassedRequests.Store(0)
	runtimeStats.storedResponses.Store(0)
}
