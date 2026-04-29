package op

import (
	"encoding/json"
	"strings"

	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/utils/semantic_cache"
)

func NormalizeNavOrder(raw string, defaults []string) []string {
	var input []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &input); err != nil {
		return append([]string(nil), defaults...)
	}

	seen := make(map[string]struct{}, len(defaults))
	allowed := make(map[string]struct{}, len(defaults))
	for _, id := range defaults {
		allowed[id] = struct{}{}
	}

	out := make([]string, 0, len(defaults))
	for _, id := range input {
		id = strings.TrimSpace(id)
		if _, ok := allowed[id]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}

	for _, id := range defaults {
		if _, ok := seen[id]; ok {
			continue
		}
		out = append(out, id)
	}

	return out
}

func buildSemanticCacheEvaluationSummary(
	enabled bool,
	runtimeEnabled bool,
	ttlSeconds int,
	threshold int,
	maxEntries int,
	currentEntries int,
	hits int64,
	misses int64,
	stats semantic_cache.RuntimeStats,
) model.SemanticCacheEvaluationSummary {
	totalLookups := hits + misses
	hitRate := 0.0
	if totalLookups > 0 {
		hitRate = (float64(hits) / float64(totalLookups)) * 100
	}

	usageRate := 0.0
	if maxEntries > 0 {
		usageRate = (float64(currentEntries) / float64(maxEntries)) * 100
	}

	return model.SemanticCacheEvaluationSummary{
		Enabled:           enabled,
		RuntimeEnabled:    runtimeEnabled,
		TTLSeconds:        ttlSeconds,
		Threshold:         threshold,
		MaxEntries:        maxEntries,
		CurrentEntries:    currentEntries,
		Hits:              hits,
		Misses:            misses,
		HitRate:           hitRate,
		UsageRate:         usageRate,
		EvaluatedRequests: stats.EvaluatedRequests,
		CacheHitResponses: stats.CacheHitResponses,
		CacheMissRequests: stats.CacheMissRequests,
		BypassedRequests:  stats.BypassedRequests,
		StoredResponses:   stats.StoredResponses,
	}
}
