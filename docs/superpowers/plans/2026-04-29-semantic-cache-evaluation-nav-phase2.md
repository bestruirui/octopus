# Semantic Cache / Evaluation / Nav Order 二期实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 在不改动数据库结构的前提下，把语义缓存做成真实文本请求缓存，补齐设置页语义缓存与页面顺序配置，并将 `Analytics -> Evaluation` 扩展为展示缓存成效的可用页面。

**架构：** 后端将语义缓存拆成独立的运行时模块、embedding 客户端和 relay 接入层；设置保存后即时刷新运行态，analytics 新增轻量 evaluation 摘要接口。前端继续沿用现有 Setting / Analytics / Navbar 结构，在设置页增加两个卡片，在导航层增加服务端顺序加载，在 Evaluation 页增加缓存成效卡而不复制完整控制台。

**技术栈：** Go、Gin、GORM、Next.js App Router、TanStack Query、Zustand、`@hello-pangea/dnd`

---

## 文件结构

### 后端

- 修改：`internal/model/setting.go`
  - 新增语义缓存 embedding 设置键与 `nav_order`
  - 补充默认值与校验逻辑

- 修改：`internal/model/analytics.go`
  - 新增 `AnalyticsEvaluationSummary` 与 `SemanticCacheEvaluationSummary`

- 修改：`internal/model/ops.go`
  - 如需复用，补充更细的缓存状态 DTO 字段

- 创建：`internal/utils/semantic_cache/runtime.go`
  - 管理缓存运行配置、命名空间、运行期统计、刷新与 clear/reset

- 创建：`internal/utils/semantic_cache/client.go`
  - 直连外部 embedding 服务，不经过 Octopus relay

- 创建：`internal/utils/semantic_cache/request.go`
  - 从 `InternalLLMRequest` 提取文本、做归一化、判断是否可缓存

- 修改：`internal/utils/semantic_cache/cache.go`
  - 将缓存 entry 升级为带命名空间
  - 修复 `RLock` 下写计数的问题

- 创建：`internal/utils/semantic_cache/cache_test.go`
  - 覆盖命名空间隔离、TTL、统计计数、clear/reset

- 创建：`internal/utils/semantic_cache/client_test.go`
  - 覆盖 embedding 请求构造、超时、错误处理、响应解析

- 创建：`internal/relay/semantic_cache.go`
  - 封装 relay 侧查找/回写/旁路判定

- 创建：`internal/relay/semantic_cache_test.go`
  - 覆盖请求文本抽取、旁路规则、命中回写边界

- 修改：`internal/relay/relay.go`
  - 删除每请求 `initSemanticCacheFromSettings()`
  - 在非流式文本成功路径接入缓存查找与回写

- 创建：`internal/op/nav_order.go`
  - 负责 `nav_order` 清洗、默认顺序拼补、序列化/反序列化

- 修改：`internal/op/analytics.go`
  - 新增 `AnalyticsEvaluationGet`

- 修改：`internal/op/ops.go`
  - 复用 runtime stats 构造更完整缓存状态

- 修改：`internal/op/ops_test.go`
  - 补充 evaluation summary / nav order normalize 的纯函数测试

- 修改：`internal/server/handlers/analytics.go`
  - 注册 `GET /api/v1/analytics/evaluation`

- 修改：`internal/server/handlers/setting.go`
  - 设置保存后刷新语义缓存运行态

### 前端

- 修改：`web/src/api/endpoints/setting.ts`
  - 新增 SettingKey 常量
  - 提供 nav order / semantic cache 所需键

- 修改：`web/src/api/endpoints/analytics.ts`
  - 新增 `useAnalyticsEvaluationSummary`

- 创建：`web/src/components/modules/setting/SemanticCache.tsx`
  - 语义缓存设置卡

- 创建：`web/src/components/modules/setting/PageOrder.tsx`
  - 页面顺序拖拽卡

- 创建：`web/src/components/modules/navbar/nav-order.ts`
  - 顶级页面默认顺序、前端清洗辅助函数

- 创建：`web/src/components/modules/navbar/nav-order.test.ts`
  - 覆盖页面顺序清洗逻辑

- 修改：`web/src/components/modules/navbar/nav-store.ts`
  - 保存 `navOrder`
  - 用生效顺序计算 `direction`

- 修改：`web/src/components/modules/navbar/navbar.tsx`
  - 按生效顺序渲染导航按钮

- 修改：`web/src/route/config.tsx`
  - 暴露默认路由顺序与 route id 元数据

- 修改：`web/src/components/app.tsx`
  - 认证后预取 settings，减少 navbar 首屏抖动

- 修改：`web/src/components/modules/setting/index.tsx`
  - 插入新卡片

- 修改：`web/src/components/modules/analytics/Evaluation.tsx`
  - 增加语义缓存入口卡与缓存成效摘要

- 修改：`web/public/locale/en.json`
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/zh_hant.json`
  - 补齐 Setting / Analytics 新文案

## 任务 1：设置键、缓存运行时和纯函数骨架

**文件：**
- 创建：`internal/utils/semantic_cache/runtime.go`
- 创建：`internal/op/nav_order.go`
- 修改：`internal/model/setting.go`
- 修改：`internal/model/analytics.go`
- 修改：`internal/op/ops_test.go`
- 测试：`internal/op/ops_test.go`

- [ ] **步骤 1：先写后端纯函数失败测试**

```go
func TestNormalizeNavOrder_AppendsMissingRoutesAndDropsUnknown(t *testing.T) {
	defaults := []string{"home", "channel", "group", "model", "analytics", "log", "alert", "ops", "setting", "user"}
	got := NormalizeNavOrder(`["group","group","unknown","setting"]`, defaults)
	want := []string{"group", "setting", "home", "channel", "model", "analytics", "log", "alert", "ops", "user"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeNavOrder() = %v, want %v", got, want)
	}
}

func TestBuildSemanticCacheEvaluationSummary_ComputesRates(t *testing.T) {
	stats := semantic_cache.RuntimeStats{
		EvaluatedRequests: 12,
		CacheHitResponses: 8,
		CacheMissRequests: 3,
		BypassedRequests: 1,
		StoredResponses:   3,
	}
	got := buildSemanticCacheEvaluationSummary(
		true, true, 3600, 98, 1000, 120, 80, 40, stats,
	)
	if got.HitRate != 66.66666666666666 {
		t.Fatalf("HitRate = %v", got.HitRate)
	}
	if got.UsageRate != 12 {
		t.Fatalf("UsageRate = %v", got.UsageRate)
	}
}
```

- [ ] **步骤 2：运行测试验证当前缺少实现**

运行：`go test ./internal/op -run "TestNormalizeNavOrder|TestBuildSemanticCacheEvaluationSummary" -count=1`

预期：FAIL，出现 `undefined: NormalizeNavOrder`、`undefined: buildSemanticCacheEvaluationSummary` 或缺少 `RuntimeStats` 类型。

- [ ] **步骤 3：补齐设置键、运行态统计结构和清洗纯函数**

```go
const (
	SettingKeySemanticCacheEmbeddingBaseURL        SettingKey = "semantic_cache_embedding_base_url"
	SettingKeySemanticCacheEmbeddingAPIKey         SettingKey = "semantic_cache_embedding_api_key"
	SettingKeySemanticCacheEmbeddingModel          SettingKey = "semantic_cache_embedding_model"
	SettingKeySemanticCacheEmbeddingTimeoutSeconds SettingKey = "semantic_cache_embedding_timeout_seconds"
	SettingKeyNavOrder                             SettingKey = "nav_order"
)

type SemanticCacheEvaluationSummary struct {
	Enabled             bool    `json:"enabled"`
	RuntimeEnabled      bool    `json:"runtime_enabled"`
	TTLSeconds          int     `json:"ttl_seconds"`
	Threshold           int     `json:"threshold"`
	MaxEntries          int     `json:"max_entries"`
	CurrentEntries      int     `json:"current_entries"`
	Hits                int64   `json:"hits"`
	Misses              int64   `json:"misses"`
	HitRate             float64 `json:"hit_rate"`
	UsageRate           float64 `json:"usage_rate"`
	EvaluatedRequests   int64   `json:"evaluated_requests"`
	CacheHitResponses   int64   `json:"cache_hit_responses"`
	CacheMissRequests   int64   `json:"cache_miss_requests"`
	BypassedRequests    int64   `json:"bypassed_requests"`
	StoredResponses     int64   `json:"stored_responses"`
}

type RuntimeStats struct {
	EvaluatedRequests int64
	CacheHitResponses int64
	CacheMissRequests int64
	BypassedRequests  int64
	StoredResponses   int64
}

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
```

- [ ] **步骤 4：重新运行纯函数测试**

运行：`go test ./internal/op -run "TestNormalizeNavOrder|TestBuildSemanticCacheEvaluationSummary" -count=1`

预期：PASS。

- [ ] **步骤 5：提交这一层骨架**

```bash
git add internal/model/setting.go internal/model/analytics.go internal/op/nav_order.go internal/op/ops_test.go internal/utils/semantic_cache/runtime.go
git commit -m "feat: add semantic cache runtime primitives"
```

## 任务 2：实现真实 embedding 客户端与命名空间缓存

**文件：**
- 修改：`internal/utils/semantic_cache/cache.go`
- 创建：`internal/utils/semantic_cache/cache_test.go`
- 创建：`internal/utils/semantic_cache/client.go`
- 创建：`internal/utils/semantic_cache/client_test.go`
- 测试：`internal/utils/semantic_cache/cache_test.go`
- 测试：`internal/utils/semantic_cache/client_test.go`

- [ ] **步骤 1：编写缓存命名空间与客户端失败测试**

```go
func TestLookup_IsolatedByNamespace(t *testing.T) {
	Reset()
	ApplyRuntimeConfig(RuntimeConfig{Enabled: true, MaxEntries: 16, Threshold: 0.95, TTL: time.Hour})
	embedding := []float64{1, 0}
	Store("k1:chat:gpt-4.1", "req-a", []byte(`{"id":"resp-a"}`), embedding)
	if _, ok := Lookup("k2:chat:gpt-4.1", embedding); ok {
		t.Fatal("expected namespace miss")
	}
}

func TestEmbeddingClient_CreateEmbedding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer srv.Close()

	client := NewEmbeddingClient(RuntimeConfig{
		EmbeddingBaseURL: srv.URL,
		EmbeddingModel:   "text-embedding-3-small",
		EmbeddingTimeout: 5 * time.Second,
	})
	got, err := client.CreateEmbedding(context.Background(), "hello")
	if err != nil {
		t.Fatalf("CreateEmbedding() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("embedding length = %d", len(got))
	}
}
```

- [ ] **步骤 2：运行语义缓存包测试确认失败**

运行：`go test ./internal/utils/semantic_cache -count=1`

预期：FAIL，出现 `too many arguments in call to Store`、`undefined: NewEmbeddingClient`、`undefined: Reset` 等错误。

- [ ] **步骤 3：实现命名空间缓存、统计修复和 embedding 客户端**

```go
type CacheEntry struct {
	Namespace    string
	RequestKey   string
	ResponseJSON []byte
	Embedding    []float64
	CreatedAt    time.Time
	LastAccessAt time.Time
	HitCount     int64
}

func Lookup(namespace string, embedding []float64) ([]byte, bool) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	globalCache.pruneExpiredLocked()
	bestIdx := -1
	bestSim := -1.0
	for i, entry := range globalCache.entries {
		if entry.Namespace != namespace {
			continue
		}
		sim := cosineSimilarity(embedding, entry.Embedding)
		if sim > bestSim {
			bestIdx = i
			bestSim = sim
		}
	}
	if bestIdx >= 0 && bestSim >= globalCache.threshold {
		globalCache.hits++
		globalCache.entries[bestIdx].HitCount++
		globalCache.entries[bestIdx].LastAccessAt = time.Now()
		return append([]byte(nil), globalCache.entries[bestIdx].ResponseJSON...), true
	}
	globalCache.misses++
	return nil, false
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

func (c *EmbeddingClient) CreateEmbedding(ctx context.Context, text string) ([]float64, error) {
	body, _ := json.Marshal(embeddingRequest{Model: c.model, Input: text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return nil, fmt.Errorf("embedding upstream error: %d: %s", resp.StatusCode, string(payload))
	}
	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embedding response missing data[0].embedding")
	}
	return parsed.Data[0].Embedding, nil
}
```

- [ ] **步骤 4：运行语义缓存包测试**

运行：`go test ./internal/utils/semantic_cache -count=1`

预期：PASS。

- [ ] **步骤 5：提交底层缓存模块**

```bash
git add internal/utils/semantic_cache/cache.go internal/utils/semantic_cache/cache_test.go internal/utils/semantic_cache/client.go internal/utils/semantic_cache/client_test.go
git commit -m "feat: add real semantic cache runtime and client"
```

## 任务 3：把语义缓存接进 relay 文本非流式链路

**文件：**
- 创建：`internal/utils/semantic_cache/request.go`
- 创建：`internal/relay/semantic_cache.go`
- 创建：`internal/relay/semantic_cache_test.go`
- 修改：`internal/relay/relay.go`
- 测试：`internal/relay/semantic_cache_test.go`

- [ ] **步骤 1：先写 relay 侧失败测试**

```go
func TestBuildSemanticCacheText_ChatMessagesOnlyUsesText(t *testing.T) {
	userText := "hello"
	req := &transmodel.InternalLLMRequest{
		Model: "gpt-4.1",
		Messages: []transmodel.Message{
			{Role: "user", Content: transmodel.MessageContent{Content: &userText}},
		},
	}
	namespace, text, ok := buildSemanticCacheLookupInput(7, "chat", req)
	if !ok {
		t.Fatal("expected cacheable request")
	}
	if namespace != "7:chat:gpt-4.1" {
		t.Fatalf("namespace = %q", namespace)
	}
	if text != "user: hello" {
		t.Fatalf("text = %q", text)
	}
}

func TestBuildSemanticCacheLookupInput_BypassesStreamRequests(t *testing.T) {
	stream := true
	req := &transmodel.InternalLLMRequest{Model: "gpt-4.1", Stream: &stream}
	if _, _, ok := buildSemanticCacheLookupInput(1, "chat", req); ok {
		t.Fatal("expected stream request to bypass semantic cache")
	}
}
```

- [ ] **步骤 2：运行 relay 相关测试确认失败**

运行：`go test ./internal/relay -run "TestBuildSemanticCache" -count=1`

预期：FAIL，出现 `undefined: buildSemanticCacheLookupInput`。

- [ ] **步骤 3：实现文本抽取、旁路规则与 relay 接入**

```go
func buildSemanticCacheLookupInput(apiKeyID int, endpointFamily string, req *transmodel.InternalLLMRequest) (string, string, bool) {
	if req == nil || apiKeyID <= 0 || req == nil || req.Model == "" {
		semantic_cache.RecordBypass()
		return "", "", false
	}
	if req.Stream != nil && *req.Stream {
		semantic_cache.RecordBypass()
		return "", "", false
	}
	text, ok := semantic_cache.ExtractNormalizedText(req)
	if !ok {
		semantic_cache.RecordBypass()
		return "", "", false
	}
	return semantic_cache.BuildNamespace(apiKeyID, endpointFamily, req.Model), text, true
}

func maybeServeSemanticCacheHit(c *gin.Context, req *relayRequest, endpointFamily string) (bool, error) {
	namespace, text, ok := buildSemanticCacheLookupInput(req.apiKeyID, endpointFamily, req.internalRequest)
	if !ok {
		return false, nil
	}
	embedding, err := semantic_cache.CreateEmbedding(req.operationCtx, text)
	if err != nil {
		semantic_cache.RecordBypass()
		return false, nil
	}
	semantic_cache.RecordEvaluated()
	if payload, found := semantic_cache.Lookup(namespace, embedding); found {
		semantic_cache.RecordHit()
		c.Data(http.StatusOK, "application/json", payload)
		return true, nil
	}
	semantic_cache.RecordMiss()
	req.internalRequest.TransformerMetadata["semantic_cache_namespace"] = namespace
	req.internalRequest.TransformerMetadata["semantic_cache_text"] = text
	semantic_cache.AttachEmbedding(req.operationCtx, embedding)
	return false, nil
}
```

在 `relay.go` 中接入点：

```go
endpointFamily := "chat"
if internalRequest.RawAPIFormat == model.APIFormatOpenAIResponse {
	endpointFamily = "responses"
}
if served, err := maybeServeSemanticCacheHit(c, req, endpointFamily); err != nil {
	lastErr = err
	resp.BadGateway(c)
	return
} else if served {
	metrics.Save(true, nil, iter.Attempts())
	return
}
```

并在非流式成功响应后回写：

```go
if err := ra.handleResponse(ctx, response); err != nil {
	return 0, err
}
storeSemanticCacheResponse(ra.internalRequest, ra.c.Writer)
```

- [ ] **步骤 4：运行 relay 包测试**

运行：`go test ./internal/relay -run "TestBuildSemanticCache" -count=1`

预期：PASS。

- [ ] **步骤 5：提交 relay 接入**

```bash
git add internal/relay/relay.go internal/relay/semantic_cache.go internal/relay/semantic_cache_test.go internal/utils/semantic_cache/request.go
git commit -m "feat: wire semantic cache into text relay path"
```

## 任务 4：补齐后端 setting/analytics 接口与运行态刷新

**文件：**
- 修改：`internal/op/analytics.go`
- 修改：`internal/op/ops.go`
- 修改：`internal/server/handlers/analytics.go`
- 修改：`internal/server/handlers/setting.go`
- 修改：`internal/op/ops_test.go`
- 测试：`internal/op/ops_test.go`

- [ ] **步骤 1：编写 evaluation 摘要与 setting 刷新失败测试**

```go
func TestBuildSemanticCacheEvaluationSummary_UsesRuntimeStats(t *testing.T) {
	stats := semantic_cache.RuntimeStats{
		EvaluatedRequests: 20,
		CacheHitResponses: 11,
		CacheMissRequests: 7,
		BypassedRequests:  2,
		StoredResponses:   7,
	}
	got := buildSemanticCacheEvaluationSummary(true, false, 3600, 98, 200, 10, 11, 7, stats)
	if got.RuntimeEnabled {
		t.Fatal("runtime should stay false")
	}
	if got.BypassedRequests != 2 {
		t.Fatalf("BypassedRequests = %d", got.BypassedRequests)
	}
}
```

- [ ] **步骤 2：运行 op 测试确认当前 evaluation 接口缺失**

运行：`go test ./internal/op -run "TestBuildSemanticCacheEvaluationSummary" -count=1`

预期：FAIL，出现字段或函数不匹配。

- [ ] **步骤 3：实现 analytics evaluation 数据拼装与 setting 保存刷新**

```go
func AnalyticsEvaluationGet(ctx context.Context) (*model.AnalyticsEvaluationSummary, error) {
	enabled, err := SettingGetBool(model.SettingKeySemanticCacheEnabled)
	if err != nil {
		return nil, err
	}
	ttlSeconds, _ := SettingGetInt(model.SettingKeySemanticCacheTTL)
	threshold, _ := SettingGetInt(model.SettingKeySemanticCacheThreshold)
	maxEntries, _ := SettingGetInt(model.SettingKeySemanticCacheMaxEntries)
	hits, misses, size := semantic_cache.Stats()
	stats := semantic_cache.GetRuntimeStats()
	summary := &model.AnalyticsEvaluationSummary{
		SemanticCache: buildSemanticCacheEvaluationSummary(
			enabled,
			semantic_cache.RuntimeEnabled(),
			ttlSeconds,
			threshold,
			maxEntries,
			size,
			hits,
			misses,
			stats,
		),
	}
	return summary, nil
}

func getAnalyticsEvaluation(c *gin.Context) {
	data, err := op.AnalyticsEvaluationGet(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, data)
}
```

在 `setSetting` 中补 runtime refresh：

```go
case model.SettingKeySemanticCacheEnabled,
	model.SettingKeySemanticCacheTTL,
	model.SettingKeySemanticCacheThreshold,
	model.SettingKeySemanticCacheMaxEntries,
	model.SettingKeySemanticCacheEmbeddingBaseURL,
	model.SettingKeySemanticCacheEmbeddingAPIKey,
	model.SettingKeySemanticCacheEmbeddingModel,
	model.SettingKeySemanticCacheEmbeddingTimeoutSeconds:
	if err := semantic_cache.RefreshFromSettings(); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
```

- [ ] **步骤 4：运行 op 与 handler 相关测试**

运行：`go test ./internal/op ./internal/server/handlers -count=1`

预期：PASS。

- [ ] **步骤 5：提交后端接口层**

```bash
git add internal/op/analytics.go internal/op/ops.go internal/op/ops_test.go internal/server/handlers/analytics.go internal/server/handlers/setting.go
git commit -m "feat: expose semantic cache evaluation summary"
```

## 任务 5：设置页补齐语义缓存卡与页面顺序卡

**文件：**
- 修改：`web/src/api/endpoints/setting.ts`
- 修改：`web/src/components/modules/setting/index.tsx`
- 创建：`web/src/components/modules/setting/SemanticCache.tsx`
- 创建：`web/src/components/modules/setting/PageOrder.tsx`
- 修改：`web/public/locale/en.json`
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/zh_hant.json`

- [ ] **步骤 1：先写页面顺序纯 helper 失败测试**

```ts
import assert from 'node:assert/strict';
import test from 'node:test';
import { DEFAULT_NAV_ORDER, normalizeNavOrder } from './nav-order.ts';

test('normalizeNavOrder drops unknown ids and appends missing defaults', () => {
  const got = normalizeNavOrder(['group', 'group', 'unknown', 'setting'], DEFAULT_NAV_ORDER);
  assert.deepEqual(got, ['group', 'setting', 'home', 'channel', 'model', 'analytics', 'log', 'alert', 'ops', 'user']);
});
```

- [ ] **步骤 2：运行前端 helper 测试确认失败**

运行：`node --experimental-strip-types --test web/src/components/modules/navbar/nav-order.test.ts`

预期：FAIL，出现 `Cannot find module './nav-order.ts'` 或 `normalizeNavOrder is not exported`。

- [ ] **步骤 3：实现设置卡与拖拽排序卡**

```ts
export const SettingKey = {
  ...,
  SemanticCacheEmbeddingBaseURL: 'semantic_cache_embedding_base_url',
  SemanticCacheEmbeddingAPIKey: 'semantic_cache_embedding_api_key',
  SemanticCacheEmbeddingModel: 'semantic_cache_embedding_model',
  SemanticCacheEmbeddingTimeoutSeconds: 'semantic_cache_embedding_timeout_seconds',
  NavOrder: 'nav_order',
} as const;
```

```tsx
export function SettingSemanticCache() {
  const { data: settings } = useSettingList();
  const setSetting = useSetSetting();
  const [enabled, setEnabled] = useState(false);
  const [baseURL, setBaseURL] = useState('');
  const [apiKey, setAPIKey] = useState('');
  const [model, setModel] = useState('');
  const [timeoutSeconds, setTimeoutSeconds] = useState('10');
  // 同 AIRoute 卡片：load settings -> local state -> onBlur/save
}
```

```tsx
export function SettingPageOrder() {
  const { navOrder, saveNavOrder, resetNavOrder } = useNavStore();
  return (
    <DragDropContext onDragEnd={handleDragEnd}>
      <Droppable droppableId="page-order">
        {(provided) => (
          <div ref={provided.innerRef} {...provided.droppableProps}>
            {navOrder.map((id, index) => (
              <Draggable key={id} draggableId={id} index={index}>
                {(drag) => (
                  <div ref={drag.innerRef} {...drag.draggableProps} {...drag.dragHandleProps}>
                    {t(`navbar.${id}`)}
                  </div>
                )}
              </Draggable>
            ))}
            {provided.placeholder}
          </div>
        )}
      </Droppable>
    </DragDropContext>
  );
}
```

- [ ] **步骤 4：运行前端静态检查**

运行：

```bash
node --experimental-strip-types --test web/src/components/modules/navbar/nav-order.test.ts
cd web && pnpm lint
```

预期：测试 PASS，`pnpm lint` 无新增错误。

- [ ] **步骤 5：提交设置页改动**

```bash
git add web/src/api/endpoints/setting.ts web/src/components/modules/setting/index.tsx web/src/components/modules/setting/SemanticCache.tsx web/src/components/modules/setting/PageOrder.tsx web/src/components/modules/navbar/nav-order.test.ts web/public/locale/en.json web/public/locale/zh_hans.json web/public/locale/zh_hant.json
git commit -m "feat: add semantic cache and page order settings"
```

## 任务 6：导航顺序接入 Navbar 与 App 预取

**文件：**
- 创建：`web/src/components/modules/navbar/nav-order.ts`
- 修改：`web/src/components/modules/navbar/nav-store.ts`
- 修改：`web/src/components/modules/navbar/navbar.tsx`
- 修改：`web/src/route/config.tsx`
- 修改：`web/src/components/app.tsx`
- 测试：`web/src/components/modules/navbar/nav-order.test.ts`

- [ ] **步骤 1：写出方向计算与默认顺序的失败测试**

```ts
test('normalizeNavOrder preserves default order when input is empty', () => {
  assert.deepEqual(normalizeNavOrder([], DEFAULT_NAV_ORDER), DEFAULT_NAV_ORDER);
});
```

- [ ] **步骤 2：运行 helper 测试确认当前默认顺序导出缺失**

运行：`node --experimental-strip-types --test web/src/components/modules/navbar/nav-order.test.ts`

预期：FAIL。

- [ ] **步骤 3：实现 nav order 辅助与 store 接入**

```ts
export const DEFAULT_NAV_ORDER = ['home', 'channel', 'group', 'model', 'analytics', 'log', 'alert', 'ops', 'setting', 'user'] as const;

export function normalizeNavOrder(input: string[], defaults = [...DEFAULT_NAV_ORDER]): string[] {
  const allowed = new Set(defaults);
  const seen = new Set<string>();
  const ordered = input
    .map((item) => item.trim())
    .filter((item) => allowed.has(item) && !seen.has(item) && (seen.add(item), true));
  for (const id of defaults) {
    if (!seen.has(id)) {
      ordered.push(id);
    }
  }
  return ordered;
}
```

```ts
interface NavState {
  activeItem: NavItem;
  prevItem: NavItem | null;
  direction: number;
  navOrder: NavItem[];
  setNavOrder: (order: NavItem[]) => void;
  resetNavOrder: () => void;
}

setActiveItem: (item) => {
  const { activeItem, navOrder } = get();
  const currentIndex = navOrder.indexOf(activeItem);
  const nextIndex = navOrder.indexOf(item);
  const direction = nextIndex > currentIndex ? 1 : -1;
  set({ activeItem: item, prevItem: activeItem, direction });
}
```

在 `AppContainer` 中预取 settings：

```ts
prefetches.push(
  queryClient.prefetchQuery({
    queryKey: ['settings', 'list'],
    queryFn: async () => apiClient.get('/api/v1/setting/list'),
  })
);
```

- [ ] **步骤 4：运行导航 helper 与 lint**

运行：

```bash
node --experimental-strip-types --test web/src/components/modules/navbar/nav-order.test.ts
cd web && pnpm lint
```

预期：PASS。

- [ ] **步骤 5：提交导航顺序接入**

```bash
git add web/src/components/modules/navbar/nav-order.ts web/src/components/modules/navbar/nav-store.ts web/src/components/modules/navbar/navbar.tsx web/src/route/config.tsx web/src/components/app.tsx web/src/components/modules/navbar/nav-order.test.ts
git commit -m "feat: load navbar order from settings"
```

## 任务 7：扩展 Evaluation v1 展示语义缓存成效

**文件：**
- 修改：`web/src/api/endpoints/analytics.ts`
- 修改：`web/src/components/modules/analytics/Evaluation.tsx`
- 修改：`web/public/locale/en.json`
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/zh_hant.json`
- 测试：`cd web && pnpm lint`

- [ ] **步骤 1：先补 hook 类型失败检查**

```ts
export interface AnalyticsEvaluationSummary {
  semantic_cache: {
    enabled: boolean;
    runtime_enabled: boolean;
    hits: number;
    misses: number;
    hit_rate: number;
    usage_rate: number;
    evaluated_requests: number;
    cache_hit_responses: number;
    cache_miss_requests: number;
    bypassed_requests: number;
    stored_responses: number;
  };
}
```

- [ ] **步骤 2：运行 lint 确认当前字段未接入**

运行：`cd web && pnpm lint`

预期：FAIL，出现未使用/不存在的 hook 或翻译 key。

- [ ] **步骤 3：实现 evaluation summary hook 和 UI 卡片**

```ts
export function useAnalyticsEvaluationSummary() {
  return useQuery({
    queryKey: ['analytics', 'evaluation'],
    queryFn: async () => apiClient.get<AnalyticsEvaluationSummary>('/api/v1/analytics/evaluation'),
    refetchInterval: 30000,
    refetchOnMount: 'always',
  });
}
```

```tsx
const { data: evaluationSummary } = useAnalyticsEvaluationSummary();
const semanticCache = evaluationSummary?.semantic_cache;

<article className="rounded-2xl border border-border/60 bg-background/70 p-4">
  <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
    <Database className="h-4 w-4" />
  </div>
  <div className="mt-4 flex items-start justify-between gap-3">
    <div>
      <h4 className="text-sm font-semibold">{t('evaluation.semanticCache.title')}</h4>
      <p className="mt-2 text-sm leading-6 text-muted-foreground">{t('evaluation.semanticCache.description')}</p>
    </div>
    <StatusBadge
      label={semanticCache?.runtime_enabled ? t('evaluation.semanticCache.runtimeOn') : t('evaluation.semanticCache.runtimeOff')}
      tone={semanticCache?.runtime_enabled ? 'success' : (semanticCache?.enabled ? 'warning' : 'neutral')}
    />
  </div>
  <Button className="mt-4 rounded-xl" onClick={() => setActiveItem('setting')}>
    {t('evaluation.actions.openSemanticCache')}
    <ArrowRight className="size-4" />
  </Button>
</article>
```

- [ ] **步骤 4：运行前端构建检查**

运行：

```bash
cd web && pnpm lint
cd web && $env:NEXT_PUBLIC_APP_VERSION='v1.4.2'; pnpm build
```

预期：PASS。

- [ ] **步骤 5：提交 Evaluation 扩展**

```bash
git add web/src/api/endpoints/analytics.ts web/src/components/modules/analytics/Evaluation.tsx web/public/locale/en.json web/public/locale/zh_hans.json web/public/locale/zh_hant.json
git commit -m "feat: show semantic cache evaluation in analytics"
```

## 任务 8：全量验证、静态资源同步与人工验收

**文件：**
- 修改：`static/out/`（由前端构建产物覆盖）
- 验证：后端 / 前端 / 本地 UI

- [ ] **步骤 1：运行后端全量测试**

运行：`go test ./...`

预期：PASS。

- [ ] **步骤 2：运行后端构建**

运行：`go build ./...`

预期：PASS。

- [ ] **步骤 3：运行前端 lint 与 build**

运行：

```bash
cd web && pnpm lint
cd web && $env:NEXT_PUBLIC_APP_VERSION='v1.4.2'; pnpm build
```

预期：PASS。

- [ ] **步骤 4：同步静态产物并启动本地验收**

运行：

```bash
Copy-Item -Path web\\out\\* -Destination static\\out -Recurse -Force
go run main.go start
```

人工验收清单：

```text
1. 打开 Setting，确认出现 Semantic Cache 与 Page Order 两张新卡
2. 修改 nav 顺序并刷新页面，确认顺序保持且切换动画方向正常
3. 配置独立 embedding 服务后，对同一 API Key / 模型发起两次非流式文本请求，第二次命中缓存
4. 打开 Analytics -> Evaluation，确认出现语义缓存入口卡和成效摘要
5. 关闭或打坏 embedding 服务，确认请求仍正常转发，Evaluation 里 bypass 计数增加
```

- [ ] **步骤 5：提交最终整合**

```bash
git add internal web static/out
git commit -m "feat: complete semantic cache phase2"
```

## 自检

### 规格覆盖度

已覆盖的规格章节与对应任务：

1. 语义缓存后端闭环
   - 任务 1、2、3、4
2. 设置页补齐
   - 任务 5
3. 全局页面顺序
   - 任务 1、5、6
4. Evaluation v1 扩展
   - 任务 4、7
5. 错误处理与降级
   - 任务 2、3、4、8
6. 验证标准
   - 任务 8

遗漏：无。

### 占位符扫描

已避免使用占位式描述。每个任务都包含了目标文件、示例代码、验证命令和 commit 粒度。

### 类型一致性

计划统一使用以下命名：

1. `SemanticCacheEvaluationSummary`
2. `AnalyticsEvaluationSummary`
3. `NormalizeNavOrder`
4. `RuntimeStats`
5. `useAnalyticsEvaluationSummary`
6. `DEFAULT_NAV_ORDER`

执行本计划时不要再切换成其他近义名。
