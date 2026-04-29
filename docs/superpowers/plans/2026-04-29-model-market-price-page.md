# Model Market 替换 Price 页实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将现有 `model` 页面从“模型价格管理”升级为“模型广场”，保留创建/编辑/删除/刷新价格能力，并且与 `octopus` 现有后台 UI 保持一致。

**架构：** 后端新增只读聚合接口 `/api/v1/model/market`，把价格、模型-渠道覆盖、渠道 key 数和运行统计拼成前端可直接消费的 ViewModel；前端继续沿用现有 `model` 路由、toolbar、虚拟滚动和 overlay 体系，只替换摘要区和卡片内容，不改底层价格写接口。

**技术栈：** Go、Gin、GORM、Next.js App Router、TanStack Query、Zustand、Motion、Lucide

---

## 文件结构

### 后端

- 修改：`internal/model/llm.go`
  - 增加 `ModelMarketSummary`、`ModelMarketItem`、`ModelMarketChannel`、`ModelMarketResponse`

- 创建：`internal/op/model_market.go`
  - 聚合价格、渠道覆盖、启用 key 数、成功率、平均延迟、页面汇总

- 创建：`internal/op/model_market_test.go`
  - 覆盖模型广场聚合规则

- 修改：`internal/op/stats.go`
  - 提供 `StatsModelList()` 只读快照 helper

- 修改：`internal/server/handlers/model.go`
  - 新增 `GET /api/v1/model/market`

### 前端

- 修改：`web/src/api/endpoints/model.ts`
  - 新增模型广场接口类型与 `useModelMarket()`
  - 继续保留原有价格 CRUD hooks

- 创建：`web/src/components/modules/model/MarketSummary.tsx`
  - 模型广场顶部摘要区

- 修改：`web/src/components/modules/model/index.tsx`
  - 改为消费 `useModelMarket()`，输出摘要区 + 卡片列表

- 修改：`web/src/components/modules/model/Item.tsx`
  - 从纯价格卡片升级为模型广场卡片
  - 保留复制、编辑、删除，新增展开态

- 修改：`web/src/components/modules/model/ItemOverlays.tsx`
  - 保留价格编辑 overlay
  - 如需要，抽出只读详情子块

- 修改：`web/src/lib/model-icons.tsx`
  - 在现有图标映射上补充供应商标签能力

- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/zh_hant.json`
- 修改：`web/public/locale/en.json`
  - 更新“价格”到“模型广场 / Model Market”相关文案
  - 新增摘要区、卡片元信息、展开态文案

- 修改：`web/public/manifest.json`
  - 同步入口名称

## 任务 1：定义后端 DTO 与聚合测试

**文件：**
- 修改：`internal/model/llm.go`
- 创建：`internal/op/model_market.go`
- 创建：`internal/op/model_market_test.go`
- 修改：`internal/op/stats.go`
- 测试：`internal/op/model_market_test.go`

- [ ] **步骤 1：先写模型广场聚合失败测试**

```go
func TestBuildModelMarket_AggregatesChannelsKeysAndStats(t *testing.T) {
	items, summary := buildModelMarket(
		[]model.LLMInfo{
			{Name: "gpt-5.2", LLMPrice: model.LLMPrice{Input: 1, Output: 2, CacheRead: 0.1, CacheWrite: 0.2}},
		},
		[]model.LLMChannel{
			{Name: "gpt-5.2", ChannelID: 1, ChannelName: "NMapi", Enabled: true},
			{Name: "gpt-5.2", ChannelID: 2, ChannelName: "Ygxz", Enabled: false},
		},
		map[int]model.Channel{
			1: {ID: 1, Enabled: true, Keys: []model.ChannelKey{{Enabled: true}, {Enabled: true}, {Enabled: false}}},
			2: {ID: 2, Enabled: false, Keys: []model.ChannelKey{{Enabled: true}}},
		},
		[]model.StatsModel{
			{ID: 1, Name: "gpt-5.2", ChannelID: 1, StatsMetrics: model.StatsMetrics{WaitTime: 3000, RequestSuccess: 9, RequestFailed: 1}},
			{ID: 2, Name: "gpt-5.2", ChannelID: 2, StatsMetrics: model.StatsMetrics{WaitTime: 1000, RequestSuccess: 1, RequestFailed: 1}},
		},
		time.Date(2026, 4, 29, 10, 0, 0, 0, time.FixedZone("CST", 8*3600)),
	)

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ChannelCount != 2 {
		t.Fatalf("ChannelCount = %d, want 2", items[0].ChannelCount)
	}
	if items[0].EnabledKeyCount != 3 {
		t.Fatalf("EnabledKeyCount = %d, want 3", items[0].EnabledKeyCount)
	}
	if items[0].AverageLatencyMS != 333 {
		t.Fatalf("AverageLatencyMS = %d, want 333", items[0].AverageLatencyMS)
	}
	if items[0].SuccessRate != 0.8333333333333334 {
		t.Fatalf("SuccessRate = %v", items[0].SuccessRate)
	}
	if summary.UniqueChannelCount != 2 {
		t.Fatalf("UniqueChannelCount = %d, want 2", summary.UniqueChannelCount)
	}
}
```

- [ ] **步骤 2：运行测试确认当前缺实现**

运行：`go test ./internal/op -run TestBuildModelMarket_AggregatesChannelsKeysAndStats -count=1`

预期：FAIL，提示缺少 `buildModelMarket` 或 DTO 字段未定义。

- [ ] **步骤 3：补齐 DTO 和聚合骨架**

```go
type ModelMarketChannel struct {
	ChannelID       int    `json:"channel_id"`
	ChannelName     string `json:"channel_name"`
	Enabled         bool   `json:"enabled"`
	EnabledKeyCount int    `json:"enabled_key_count"`
}

type ModelMarketItem struct {
	Name             string               `json:"name"`
	Input            float64              `json:"input"`
	Output           float64              `json:"output"`
	CacheRead        float64              `json:"cache_read"`
	CacheWrite       float64              `json:"cache_write"`
	ChannelCount     int                  `json:"channel_count"`
	EnabledKeyCount  int                  `json:"enabled_key_count"`
	AverageLatencyMS int64                `json:"average_latency_ms"`
	SuccessRate      float64              `json:"success_rate"`
	RequestSuccess   int64                `json:"request_success"`
	RequestFailed    int64                `json:"request_failed"`
	Channels         []ModelMarketChannel `json:"channels"`
}

type ModelMarketSummary struct {
	ModelCount          int       `json:"model_count"`
	CoverageCount       int       `json:"coverage_count"`
	UniqueChannelCount  int       `json:"unique_channel_count"`
	AverageLatencyMS    int64     `json:"average_latency_ms"`
	LastUpdateTime      time.Time `json:"last_update_time"`
}
```

- [ ] **步骤 4：实现只读统计快照 helper**

```go
func StatsModelList() []model.StatsModel {
	statsModelMutationLock.Lock()
	defer statsModelMutationLock.Unlock()

	out := make([]model.StatsModel, 0, statsModelCache.Len())
	for _, v := range statsModelCache.GetAll() {
		out = append(out, v)
	}
	return out
}
```

- [ ] **步骤 5：实现聚合逻辑并让测试通过**

运行：`go test ./internal/op -run TestBuildModelMarket_AggregatesChannelsKeysAndStats -count=1`

预期：PASS。

## 任务 2：暴露 `/api/v1/model/market` 接口

**文件：**
- 修改：`internal/server/handlers/model.go`
- 修改：`internal/model/llm.go`
- 创建：`internal/op/model_market.go`
- 测试：`internal/op/model_market_test.go`

- [ ] **步骤 1：在 handler 中增加路由和失败 smoke case**

```go
router.NewRoute("/market", http.MethodGet).
	Handle(getModelMarket)
```

- [ ] **步骤 2：实现 handler**

```go
func getModelMarket(c *gin.Context) {
	respData, err := op.ModelMarketGet(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, respData)
}
```

- [ ] **步骤 3：实现 op 层入口**

```go
func ModelMarketGet(ctx context.Context) (model.ModelMarketResponse, error) {
	models, err := LLMList(ctx)
	if err != nil {
		return model.ModelMarketResponse{}, err
	}
	modelChannels, err := ChannelLLMList(ctx)
	if err != nil {
		return model.ModelMarketResponse{}, err
	}
	items, summary := buildModelMarket(models, modelChannels, channelCache.GetAll(), StatsModelList(), price.GetLastUpdateTime())
	return model.ModelMarketResponse{
		Summary: summary,
		Items:   items,
	}, nil
}
```

- [ ] **步骤 4：运行后端测试**

运行：`go test ./internal/op/... ./internal/server/handlers/...`

预期：PASS 或仅存在与本任务无关的已知失败；若有失败，需确认不是 `/market` 引入。

## 任务 3：补齐前端 API 类型与供应商标签能力

**文件：**
- 修改：`web/src/api/endpoints/model.ts`
- 修改：`web/src/lib/model-icons.tsx`

- [ ] **步骤 1：增加模型广场接口类型**

```ts
export interface ModelMarketChannel {
    channel_id: number;
    channel_name: string;
    enabled: boolean;
    enabled_key_count: number;
}

export interface ModelMarketItem extends LLMInfo {
    channel_count: number;
    enabled_key_count: number;
    average_latency_ms: number;
    success_rate: number;
    request_success: number;
    request_failed: number;
    channels: ModelMarketChannel[];
}

export interface ModelMarketSummary {
    model_count: number;
    coverage_count: number;
    unique_channel_count: number;
    average_latency_ms: number;
    last_update_time: string;
}

export interface ModelMarketResponse {
    summary: ModelMarketSummary;
    items: ModelMarketItem[];
}
```

- [ ] **步骤 2：新增 hook**

```ts
export function useModelMarket() {
    return useQuery({
        queryKey: ['models', 'market'],
        queryFn: async () => apiClient.get<ModelMarketResponse>('/api/v1/model/market'),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}
```

- [ ] **步骤 3：扩展图标 helper 返回供应商标签**

```ts
type ModelIconMatch = {
    Avatar: AvatarComponent;
    color: string;
    label: string;
};

export function getModelIcon(modelName: string): ModelIconMatch { ... }
```

- [ ] **步骤 4：运行前端类型检查入口**

运行：`cd web; pnpm lint`

预期：此时可能仍 FAIL，因为页面组件还未切换到新类型；确认报错集中在 `model` 模块待实现位置。

## 任务 4：把 `model` 页面改造成模型广场壳体

**文件：**
- 创建：`web/src/components/modules/model/MarketSummary.tsx`
- 修改：`web/src/components/modules/model/index.tsx`
- 修改：`web/src/api/endpoints/model.ts`
- 测试：人工页面 smoke

- [ ] **步骤 1：先实现顶部摘要组件**

```tsx
export function ModelMarketSummary({
    summary,
    lastUpdateLabel,
    onRefresh,
    isRefreshing,
}: {
    summary: ModelMarketSummary;
    lastUpdateLabel: string;
    onRefresh: () => void;
    isRefreshing: boolean;
}) {
    return (
        <section className="rounded-3xl border border-border bg-card p-5 custom-shadow">
            {/* 标题、模型总数、覆盖渠道、去重渠道、平均延迟、上次更新时间、立即更新按钮 */}
        </section>
    );
}
```

- [ ] **步骤 2：把 `index.tsx` 从 `useModelList()` 切换到 `useModelMarket()`**

```tsx
const { data } = useModelMarket();
const models = data?.items ?? [];
const summary = data?.summary;
```

- [ ] **步骤 3：保留现有搜索和 `priced/free` 筛选逻辑**

```tsx
const hasPricing = (model: ModelMarketItem) =>
    model.input + model.output + model.cache_read + model.cache_write > 0;
```

- [ ] **步骤 4：在页面顶部接入手动刷新价格按钮**

```tsx
const updatePrice = useUpdateModelPrice();
const { data: lastUpdateTime } = useLastUpdateTime();
```

- [ ] **步骤 5：继续复用 `VirtualizedGrid`**

运行：`cd web; pnpm lint`

预期：PASS 或剩余错误只集中在 `Item.tsx` 的旧 props。

## 任务 5：重写模型卡片为模型广场卡片，并保留价格编辑/删除

**文件：**
- 修改：`web/src/components/modules/model/Item.tsx`
- 修改：`web/src/components/modules/model/ItemOverlays.tsx`

- [ ] **步骤 1：把卡片 props 切换为 `ModelMarketItem`**

```tsx
interface ModelItemProps {
    model: ModelMarketItem;
    layout?: 'grid' | 'list';
}
```

- [ ] **步骤 2：保留现有编辑/删除 overlay 状态，新增 `isExpanded`**

```tsx
const [isExpanded, setIsExpanded] = useState(false);
```

- [ ] **步骤 3：把卡片主区改成广场信息结构**

```tsx
<div className="flex-1 min-w-0 flex flex-col gap-2">
    <div className="flex items-start justify-between gap-3">
        {/* 模型名 + provider */}
    </div>
    <div className="flex flex-wrap gap-2 text-sm text-muted-foreground">
        <span>{model.channel_count} 个渠道</span>
        <span>{model.enabled_key_count} 个 Key</span>
        <span>延迟 {formatLatency(model.average_latency_ms)}</span>
        <span>成功率 {formatPercent(model.success_rate)}</span>
    </div>
    <div className="flex flex-wrap gap-2">
        {/* provider 标签 + 渠道标签 */}
    </div>
</div>
```

- [ ] **步骤 4：新增展开态只读详情**

```tsx
{isExpanded ? (
    <div className="mt-4 border-t border-border/60 pt-4">
        {/* 价格详情、成功/失败数、渠道列表 */}
    </div>
) : null}
```

- [ ] **步骤 5：保留现有编辑/删除行为不变**

运行：`cd web; pnpm lint`

预期：PASS。

## 任务 6：更新导航和文案

**文件：**
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/zh_hant.json`
- 修改：`web/public/locale/en.json`
- 修改：`web/public/manifest.json`

- [ ] **步骤 1：更新导航文案**

```json
"navbar": {
  "model": "模型广场"
}
```

```json
"navbar": {
  "model": "Model Market"
}
```

- [ ] **步骤 2：为模型广场补充新文案**

```json
"model": {
  "summary": {
    "title": "模型广场",
    "coverage": "覆盖渠道",
    "uniqueChannels": "去重渠道",
    "averageLatency": "平均延迟",
    "lastUpdate": "上次更新",
    "refresh": "立即更新"
  },
  "card": {
    "provider": "供应商",
    "channels": "渠道",
    "keys": "Key",
    "successRate": "成功率",
    "latency": "延迟",
    "expand": "展开",
    "collapse": "收起"
  }
}
```

- [ ] **步骤 3：同步 manifest 入口名称**

运行：`cd web; pnpm lint`

预期：PASS。

## 任务 7：完整验证

**文件：**
- 验证：后端接口、前端构建、人工检查

- [ ] **步骤 1：运行后端测试**

运行：`go test ./internal/op/... ./internal/server/handlers/...`

预期：PASS，或仅剩无关已知失败。

- [ ] **步骤 2：运行前端 lint**

运行：`cd web; pnpm lint`

预期：PASS。

- [ ] **步骤 3：运行前端 build**

运行：`cd web; $env:NEXT_PUBLIC_APP_VERSION='dev'; pnpm build`

预期：PASS。

- [ ] **步骤 4：人工验收页面**

运行：

```powershell
go run main.go start
cd web
$env:NEXT_PUBLIC_API_BASE_URL='http://127.0.0.1:8080'
pnpm dev
```

检查项：

1. 导航已显示“模型广场”
2. 页面顶部摘要正常
3. 搜索与 `priced/free` 筛选正常
4. 卡片复制模型名正常
5. 卡片展开/收起正常
6. 价格编辑与删除正常
7. 立即更新价格按钮正常
8. list/grid 视图都可读
9. 长列表滚动无明显卡顿

- [ ] **步骤 5：整理提交范围**

```bash
git add internal/model/llm.go internal/op/model_market.go internal/op/model_market_test.go internal/op/stats.go internal/server/handlers/model.go web/src/api/endpoints/model.ts web/src/components/modules/model/MarketSummary.tsx web/src/components/modules/model/index.tsx web/src/components/modules/model/Item.tsx web/src/components/modules/model/ItemOverlays.tsx web/src/lib/model-icons.tsx web/public/locale/zh_hans.json web/public/locale/zh_hant.json web/public/locale/en.json web/public/manifest.json
git commit -m "feat: replace model price page with model market"
```
