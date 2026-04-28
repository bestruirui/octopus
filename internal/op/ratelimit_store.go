package op

import (
	"fmt"
	"sync"

	"github.com/lingyuins/octopus/internal/utils/ratelimit"
)

var (
	rateLimitRequestBuckets sync.Map // "apiKeyID:modelName" -> *ratelimit.TokenBucket
	rateLimitTokenBuckets   sync.Map // "apiKeyID:modelName" -> *ratelimit.TokenBucket
)

func rateLimitKey(apiKeyID int, modelName string) string {
	return fmt.Sprintf("%d:%s", apiKeyID, modelName)
}

// CheckRateLimit checks if the request is within the rate limits.
// Returns: allowed, remaining requests, retry-after seconds.
func CheckRateLimit(apiKeyID int, modelName string, rpm int, tpm int, tokenCount int) (allowed bool, remaining int, retryAfter int) {
	key := rateLimitKey(apiKeyID, modelName)

	// Check request rate limit
	if rpm > 0 {
		reqBucket := getOrCreateRateLimitBucket(&rateLimitRequestBuckets, key, rpm, rpm)
		if !reqBucket.Allow() {
			return false, 0, int(reqBucket.ResetAt().Unix())
		}
	}

	// Check token rate limit
	if tpm > 0 {
		tokenBucket := getOrCreateRateLimitBucket(&rateLimitTokenBuckets, key, tpm, tpm)
		if tokenCount <= 0 {
			tokenCount = 1
		}
		if !tokenBucket.AllowN(tokenCount) {
			return false, 0, int(tokenBucket.ResetAt().Unix())
		}
	}

	// Return remaining
	if rpm > 0 {
		reqBucket := getOrCreateRateLimitBucket(&rateLimitRequestBuckets, key, rpm, rpm)
		remaining = reqBucket.TokensRemaining()
	}
	return true, remaining, 0
}

// ConsumeTokens deducts the actual token count from the rate limit bucket after a successful request.
func ConsumeTokens(apiKeyID int, modelName string, tpm int, tokenCount int) {
	if tpm <= 0 || tokenCount <= 0 {
		return
	}
	key := rateLimitKey(apiKeyID, modelName)
	tokenBucket := getOrCreateRateLimitBucket(&rateLimitTokenBuckets, key, tpm, tpm)
	tokenBucket.AllowN(tokenCount)
}

func getOrCreateRateLimitBucket(m *sync.Map, key string, ratePerMinute int, burst int) *ratelimit.TokenBucket {
	if v, ok := m.Load(key); ok {
		return v.(*ratelimit.TokenBucket)
	}
	bucket := ratelimit.NewTokenBucket(ratePerMinute, burst)
	actual, _ := m.LoadOrStore(key, bucket)
	return actual.(*ratelimit.TokenBucket)
}
