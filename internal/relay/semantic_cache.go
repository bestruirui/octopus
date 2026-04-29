package relay

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	dbmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	transmodel "github.com/lingyuins/octopus/internal/transformer/model"
	"github.com/lingyuins/octopus/internal/utils/log"
	"github.com/lingyuins/octopus/internal/utils/semantic_cache"
)

const (
	semanticCacheNamespaceMetadataKey = "semantic_cache_namespace"
	semanticCacheTextMetadataKey      = "semantic_cache_text"
)

func semanticCacheEndpointFamily(req *transmodel.InternalLLMRequest) string {
	if req == nil {
		return ""
	}

	switch req.RawAPIFormat {
	case transmodel.APIFormatOpenAIChatCompletion:
		return "chat"
	case transmodel.APIFormatOpenAIResponse:
		return "responses"
	default:
		return ""
	}
}

func buildSemanticCacheLookupInput(apiKeyID int, endpointFamily string, req *transmodel.InternalLLMRequest) (string, string, bool) {
	if req == nil || apiKeyID <= 0 {
		return "", "", false
	}
	if strings.TrimSpace(endpointFamily) == "" || strings.TrimSpace(req.Model) == "" {
		return "", "", false
	}
	if req.Stream != nil && *req.Stream {
		return "", "", false
	}

	text, ok := semantic_cache.ExtractNormalizedText(req)
	if !ok {
		return "", "", false
	}

	return semantic_cache.BuildNamespace(apiKeyID, endpointFamily, req.Model), text, true
}

func maybeServeSemanticCacheHit(c *gin.Context, req *relayRequest, endpointFamily string) (bool, error) {
	if c == nil || req == nil || req.internalRequest == nil {
		return false, nil
	}

	namespace, text, ok := buildSemanticCacheLookupInput(req.apiKeyID, endpointFamily, req.internalRequest)
	if !ok {
		return false, nil
	}

	cfg, ok := loadSemanticCacheRuntimeConfig()
	if !ok {
		return false, nil
	}
	ensureSemanticCacheInitialized(cfg)

	embedding, err := semantic_cache.NewEmbeddingClient(cfg).CreateEmbedding(req.operationCtx, text)
	if err != nil {
		log.Warnf("semantic cache lookup bypassed: %v", err)
		return false, nil
	}

	if payload, found := semantic_cache.Lookup(namespace, embedding); found {
		c.Data(http.StatusOK, "application/json", payload)
		return true, nil
	}

	if req.internalRequest.TransformerMetadata == nil {
		req.internalRequest.TransformerMetadata = make(map[string]string, 2)
	}
	req.internalRequest.TransformerMetadata[semanticCacheNamespaceMetadataKey] = namespace
	req.internalRequest.TransformerMetadata[semanticCacheTextMetadataKey] = text

	return false, nil
}

func storeSemanticCacheResponse(ctx context.Context, req *transmodel.InternalLLMRequest, responseJSON []byte) {
	if req == nil || len(responseJSON) == 0 || !json.Valid(responseJSON) {
		return
	}

	namespace, text, ok := semanticCacheStoreMetadata(req)
	if !ok {
		return
	}

	cfg, ok := loadSemanticCacheRuntimeConfig()
	if !ok {
		return
	}
	ensureSemanticCacheInitialized(cfg)

	embedding, err := semantic_cache.NewEmbeddingClient(cfg).CreateEmbedding(ctx, text)
	if err != nil {
		log.Warnf("semantic cache store bypassed: %v", err)
		return
	}

	semantic_cache.Store(namespace, text, responseJSON, embedding)
}

func semanticCacheStoreMetadata(req *transmodel.InternalLLMRequest) (string, string, bool) {
	if req == nil || req.TransformerMetadata == nil {
		return "", "", false
	}

	namespace := strings.TrimSpace(req.TransformerMetadata[semanticCacheNamespaceMetadataKey])
	text := strings.TrimSpace(req.TransformerMetadata[semanticCacheTextMetadataKey])
	if namespace == "" || text == "" {
		return "", "", false
	}

	return namespace, text, true
}

func loadSemanticCacheRuntimeConfig() (semantic_cache.RuntimeConfig, bool) {
	enabled, err := op.SettingGetBool(dbmodel.SettingKeySemanticCacheEnabled)
	if err != nil || !enabled {
		return semantic_cache.RuntimeConfig{}, false
	}

	ttl, _ := op.SettingGetInt(dbmodel.SettingKeySemanticCacheTTL)
	if ttl <= 0 {
		ttl = 3600
	}

	thresholdRaw, _ := op.SettingGetInt(dbmodel.SettingKeySemanticCacheThreshold)
	if thresholdRaw < 0 || thresholdRaw > 100 {
		thresholdRaw = 98
	}

	maxEntries, _ := op.SettingGetInt(dbmodel.SettingKeySemanticCacheMaxEntries)
	if maxEntries <= 0 {
		maxEntries = 1000
	}

	baseURL, _ := op.SettingGetString(dbmodel.SettingKeySemanticCacheEmbeddingBaseURL)
	modelName, _ := op.SettingGetString(dbmodel.SettingKeySemanticCacheEmbeddingModel)
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(modelName) == "" {
		return semantic_cache.RuntimeConfig{}, false
	}

	apiKey, _ := op.SettingGetString(dbmodel.SettingKeySemanticCacheEmbeddingAPIKey)
	timeoutSeconds, _ := op.SettingGetInt(dbmodel.SettingKeySemanticCacheEmbeddingTimeoutSeconds)
	if timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}

	return semantic_cache.RuntimeConfig{
		Enabled:          true,
		MaxEntries:       maxEntries,
		Threshold:        float64(thresholdRaw) / 100.0,
		TTL:              time.Duration(ttl) * time.Second,
		EmbeddingBaseURL: strings.TrimSpace(baseURL),
		EmbeddingAPIKey:  strings.TrimSpace(apiKey),
		EmbeddingModel:   strings.TrimSpace(modelName),
		EmbeddingTimeout: time.Duration(timeoutSeconds) * time.Second,
	}, true
}

func ensureSemanticCacheInitialized(cfg semantic_cache.RuntimeConfig) {
	if semantic_cache.Enabled() {
		return
	}
	semantic_cache.Init(cfg.MaxEntries, cfg.Threshold, int(cfg.TTL/time.Second))
}
