package semantic_cache

type RuntimeStats struct {
	EvaluatedRequests int64
	CacheHitResponses int64
	CacheMissRequests int64
	BypassedRequests  int64
	StoredResponses   int64
}
