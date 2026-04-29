# OmniRoute 分析中心与运维中心实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 在保持 Octopus 现有后台 UI 一致性的前提下，先落地 `Analytics` 模块，再落地 `Ops` 模块，把 OmniRoute 概述页里最有价值的分析与运维能力并入 Octopus。

**架构：** 前端只新增两个一级入口：`analytics` 和 `ops`。`analytics` 聚焦只读聚合分析；`ops` 聚焦缓存、配额、健康、系统与审计。后端优先复用现有 `stats`、`log`、`setting`、`channel`、`group`、`apikey` 数据与权限模型，不在一期引入新的复杂业务流；二期将高风险的审计持久化与服务控制拆到尾段实现。

**技术栈：** Go、Gin、GORM、现有 `internal/op` 缓存层、Next.js 静态导出前端、React Query、`motion/react`、`next-intl`、现有 `PageWrapper` / 卡片 / 导航 / 工具栏交互模式。

---

## 文件结构

**阶段一：Analytics**

- 修改：`web/src/route/config.tsx`
- 修改：`web/src/components/modules/navbar/nav-store.ts`
- 修改：`web/src/components/app.tsx`
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/en.json`
- 修改：`web/public/locale/zh_hant.json`
- 创建：`web/src/api/endpoints/analytics.ts`
- 创建：`web/src/components/modules/analytics/index.tsx`
- 创建：`web/src/components/modules/analytics/Overview.tsx`
- 创建：`web/src/components/modules/analytics/Utilization.tsx`
- 创建：`web/src/components/modules/analytics/GroupHealth.tsx`（展示名称统一为“路由健康”）
- 创建：`web/src/components/modules/analytics/Evaluation.tsx`
- 创建：`web/src/components/modules/analytics/shared.tsx`
- 创建：`internal/model/analytics.go`
- 创建：`internal/op/analytics.go`
- 创建：`internal/op/analytics_test.go`
- 创建：`internal/server/handlers/analytics.go`

**阶段二：Ops**

- 修改：`web/src/route/config.tsx`
- 修改：`web/src/components/modules/navbar/nav-store.ts`
- 修改：`web/src/components/app.tsx`
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/en.json`
- 修改：`web/public/locale/zh_hant.json`
- 创建：`web/src/api/endpoints/ops.ts`
- 创建：`web/src/components/modules/ops/index.tsx`
- 创建：`web/src/components/modules/ops/Cache.tsx`
- 创建：`web/src/components/modules/ops/Quota.tsx`
- 创建：`web/src/components/modules/ops/Health.tsx`
- 创建：`web/src/components/modules/ops/System.tsx`
- 创建：`web/src/components/modules/ops/Audit.tsx`
- 创建：`internal/model/ops.go`
- 创建：`internal/op/ops.go`
- 创建：`internal/op/ops_test.go`
- 创建：`internal/server/handlers/ops.go`

**阶段二尾段：审计日志**

- 修改：`internal/db/db.go`
- 创建：`internal/model/audit_log.go`
- 创建：`internal/op/audit_log.go`
- 创建：`internal/op/audit_log_test.go`
- 创建：`internal/server/middleware/audit.go`
- 创建：`internal/server/handlers/audit.go`

## UI 一致性约束

- 新页面必须复用现有导航、页头和 `PageWrapper` 动画容器，不允许单独发明一套后台壳。
- 新卡片必须沿用现有圆角、边框、阴影、留白、图标密度与 `motion/react` 进入动画风格，参考 `web/src/components/modules/home/total.tsx` 和 `web/src/components/modules/group/Card.tsx`。
- 分析页和运维页内部使用同一套二级页签/分段切换样式，保持和当前 `toolbar`、`tabs trigger`、按钮组一致的交互密度。
- 一期、二期都不改现有一级导航布局，不重做 `NavBar` 结构，只增加路由项。
- 图表优先复用当前首页图表视觉语言，不引入与现有主题冲突的新配色体系。

### 任务 1：新增一级入口并搭好页面壳

**文件：**

- 修改：`web/src/route/config.tsx`
- 修改：`web/src/components/modules/navbar/nav-store.ts`
- 修改：`web/src/components/app.tsx`
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/en.json`
- 修改：`web/public/locale/zh_hant.json`
- 创建：`web/src/components/modules/analytics/index.tsx`
- 创建：`web/src/components/modules/ops/index.tsx`

- [ ] **步骤 1：扩展路由表与导航类型**

```ts
// web/src/route/config.tsx
const Analytics_Module = lazyWithPreload(() => import('@/components/modules/analytics').then(m => ({ default: m.Analytics })));
const Ops_Module = lazyWithPreload(() => import('@/components/modules/ops').then(m => ({ default: m.Ops })));

{ id: 'analytics', label: 'Analytics', icon: AnalyticsIcon, component: Analytics_Module },
{ id: 'ops', label: 'Ops', icon: Wrench, component: Ops_Module },
```

```ts
// web/src/components/modules/navbar/nav-store.ts
export type NavItem =
  | 'home'
  | 'channel'
  | 'group'
  | 'model'
  | 'analytics'
  | 'log'
  | 'alert'
  | 'ops'
  | 'setting'
  | 'user';
```

- [ ] **步骤 2：调整 App 层的类型、预取与页头文案**

```ts
// web/src/components/app.tsx
function HeaderActions({
  activeItem,
}: {
  activeItem: 'home' | 'channel' | 'group' | 'model' | 'analytics' | 'log' | 'setting' | 'ops' | 'user' | 'alert'
}) { /* ... */ }
```

```ts
case 'analytics': {
  prefetches.push(queryClient.prefetchQuery({ queryKey: ['analytics', 'overview', '7d'], queryFn: async () => apiClient.get('/api/v1/analytics/overview?range=7d') }));
  break;
}
case 'ops': {
  prefetches.push(queryClient.prefetchQuery({ queryKey: ['ops', 'health'], queryFn: async () => apiClient.get('/api/v1/ops/health') }));
  break;
}
```

- [ ] **步骤 3：创建空壳页面，先锁定 UI 一致性**

```tsx
// web/src/components/modules/analytics/index.tsx
export function Analytics() {
  return (
    <PageWrapper className="h-full min-h-0 overflow-y-auto overscroll-contain space-y-6 pb-24 md:pb-4 rounded-t-3xl">
      <section className="rounded-3xl bg-card border-card-border border p-5">...</section>
    </PageWrapper>
  );
}
```

```tsx
// web/src/components/modules/ops/index.tsx
export function Ops() {
  return (
    <PageWrapper className="h-full min-h-0 overflow-y-auto overscroll-contain space-y-6 pb-24 md:pb-4 rounded-t-3xl">
      <section className="rounded-3xl bg-card border-card-border border p-5">...</section>
    </PageWrapper>
  );
}
```

- [ ] **步骤 4：补齐一级导航文案**

运行：在 `web/public/locale/zh_hans.json`、`web/public/locale/en.json`、`web/public/locale/zh_hant.json` 中增加：

```json
{
  "navbar": {
    "analytics": "Analytics",
    "ops": "Ops"
  }
}
```

- [ ] **步骤 5：运行前端验证**

运行：`cd web && pnpm lint`
预期：PASS

运行：`cd web && $env:NEXT_PUBLIC_APP_VERSION='dev'; pnpm build`
预期：PASS，静态导出成功

### 任务 2：实现一期后端 Analytics 聚合接口

**文件：**

- 创建：`internal/model/analytics.go`
- 创建：`internal/op/analytics.go`
- 创建：`internal/op/analytics_test.go`
- 创建：`internal/server/handlers/analytics.go`

- [ ] **步骤 1：定义 Analytics 返回结构**

```go
// internal/model/analytics.go
type AnalyticsRange string

type AnalyticsOverview struct {
    TotalTokens int64 `json:"total_tokens"`
    InputTokens int64 `json:"input_tokens"`
    OutputTokens int64 `json:"output_tokens"`
    TotalCost float64 `json:"total_cost"`
    RequestCount int64 `json:"request_count"`
    ProviderCount int `json:"provider_count"`
    APIKeyCount int `json:"api_key_count"`
    ModelCount int `json:"model_count"`
    FallbackRate float64 `json:"fallback_rate"`
}
```

- [ ] **步骤 2：先写 op 层聚合测试**

```go
func TestBuildAnalyticsOverview_NoData(t *testing.T) {
    got := buildAnalyticsOverview(nil, nil, nil)
    if got.RequestCount != 0 || got.TotalTokens != 0 {
        t.Fatalf("unexpected non-zero overview: %+v", got)
    }
}
```

```go
func TestBuildProviderBreakdown_SortsByRequestsDesc(t *testing.T) {
    // 构造两个 provider，断言排序与 share 计算正确
}
```

- [ ] **步骤 3：实现聚合逻辑，复用现有缓存数据**

```go
// internal/op/analytics.go
func AnalyticsOverviewGet(ctx context.Context, r model.AnalyticsRange) (*model.AnalyticsOverview, error) { /* 复用 StatsGetDaily / StatsHourlyGet / ChannelList / APIKeyList */ }
func AnalyticsProviderBreakdownGet(ctx context.Context, r model.AnalyticsRange) ([]model.AnalyticsProviderBreakdownItem, error) { /* 基于 channel stats 聚合 */ }
func AnalyticsModelBreakdownGet(ctx context.Context, r model.AnalyticsRange) ([]model.AnalyticsModelBreakdownItem, error) { /* 基于 model stats 聚合 */ }
func AnalyticsAPIKeyBreakdownGet(ctx context.Context, r model.AnalyticsRange) ([]model.AnalyticsAPIKeyBreakdownItem, error) { /* 基于 stats apikey 聚合 */ }
func AnalyticsGroupHealthGet(ctx context.Context) ([]model.AnalyticsGroupHealthItem, error) { /* 基于 GroupList + item 数量 + endpoint_type + 最近失败日志计算路由健康分 */ }
```

- [ ] **步骤 4：注册新接口，不污染现有 `stats.go`**

```go
// internal/server/handlers/analytics.go
router.NewGroupRouter("/api/v1/analytics").
  Use(middleware.Auth()).
  Use(middleware.RequirePermission(auth.PermStatsRead)).
  AddRoute(router.NewRoute("/overview", http.MethodGet).Handle(getAnalyticsOverview)).
  AddRoute(router.NewRoute("/utilization", http.MethodGet).Handle(getAnalyticsUtilization)).
  AddRoute(router.NewRoute("/group-health", http.MethodGet).Handle(getAnalyticsGroupHealth)). // 前端页签展示为“路由健康”
  AddRoute(router.NewRoute("/provider-breakdown", http.MethodGet).Handle(getAnalyticsProviderBreakdown)).
  AddRoute(router.NewRoute("/model-breakdown", http.MethodGet).Handle(getAnalyticsModelBreakdown)).
  AddRoute(router.NewRoute("/apikey-breakdown", http.MethodGet).Handle(getAnalyticsAPIKeyBreakdown))
```

- [ ] **步骤 5：运行后端验证**

运行：`go test ./internal/op -run TestBuildAnalytics -v`
预期：PASS

运行：`go build ./...`
预期：PASS

### 任务 3：实现一期前端 Analytics 页面

**文件：**

- 创建：`web/src/api/endpoints/analytics.ts`
- 创建：`web/src/components/modules/analytics/index.tsx`
- 创建：`web/src/components/modules/analytics/shared.tsx`
- 创建：`web/src/components/modules/analytics/Overview.tsx`
- 创建：`web/src/components/modules/analytics/Utilization.tsx`
- 创建：`web/src/components/modules/analytics/GroupHealth.tsx`（展示名称统一为“路由健康”）
- 创建：`web/src/components/modules/analytics/Evaluation.tsx`
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/en.json`
- 修改：`web/public/locale/zh_hant.json`

- [ ] **步骤 1：封装前端查询 Hook**

```ts
export function useAnalyticsOverview(range: string) {
  return useQuery({
    queryKey: ['analytics', 'overview', range],
    queryFn: async () => apiClient.get<AnalyticsOverview>(`/api/v1/analytics/overview?range=${range}`),
  });
}
```

- [ ] **步骤 2：先做统一二级页签与时间范围切换**

```tsx
const ANALYTICS_TABS = ['overview', 'utilization', 'group-health', 'evaluation'] as const; // group-health 页签展示为“路由健康”
const RANGE_OPTIONS = ['1d', '7d', '30d', '90d', 'ytd', 'all'] as const;
```

要求：页签按钮使用现有圆角按钮组风格；范围切换保持与首页统计卡片一致的紧凑密度。

- [ ] **步骤 3：完成 `Overview`、`Utilization`、`路由健康（GroupHealth）` 三个可交付面板**

```tsx
// Overview.tsx
<section className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
  <MetricCard title={t('totalTokens')} value={overview.total_tokens} />
</section>
```

```tsx
// Utilization.tsx
<section className="rounded-3xl bg-card border-card-border border p-5">
  <BreakdownList items={providerBreakdown} />
</section>
```

```tsx
// GroupHealth.tsx
// 页面标题、页签与空状态文案统一使用“路由健康”
<VirtualizedGrid
  items={groups}
  layout="list"
  columns={{ default: 1, lg: 2 }}
  estimateItemHeight={180}
  renderItem={(group) => <GroupHealthCard item={group} />}
/>
```

- [ ] **步骤 4：`Evaluation` 先做 MVP，不引入新 benchmark 存储**

```tsx
// Evaluation.tsx
// 第一版只展示：
// 1. 现有 group availability 测试入口说明
// 2. AI Route 任务结果入口说明
// 3. 明确 empty-state：当前版本暂无历史 benchmark 数据归档
```

要求：不要为了一个页签新增数据库表；保持一期最小闭环。

- [ ] **步骤 5：运行前端验证**

运行：`cd web && pnpm lint`
预期：PASS

运行：`cd web && $env:NEXT_PUBLIC_APP_VERSION='dev'; pnpm build`
预期：PASS

### 任务 4：完成一期联调与人工验收

**文件：**

- 修改：`web/src/components/app.tsx`
- 修改：`web/src/api/endpoints/analytics.ts`
- 测试：`internal/op/analytics_test.go`

- [ ] **步骤 1：本地联调启动**

运行：`go run main.go start`
预期：服务正常启动

运行：`cd web && $env:NEXT_PUBLIC_API_BASE_URL='http://127.0.0.1:8080'; pnpm dev`
预期：前端正常启动

- [ ] **步骤 2：人工检查 UI 一致性**

检查项：

```text
1. analytics 页头、导航、圆角、阴影、间距与 home/group/log 一致
2. analytics 卡片没有出现新配色体系或不同按钮密度
3. 移动端和桌面端都不会破坏现有 navbar 布局
4. 无数据状态文案与现有页面一致，不出现“技术占位文本”
```

- [ ] **步骤 3：运行全量验证**

运行：`go build ./...`
预期：PASS

运行：`go test ./...`
预期：PASS

运行：`cd web && pnpm lint`
预期：PASS

运行：`cd web && $env:NEXT_PUBLIC_APP_VERSION='dev'; pnpm build`
预期：PASS

### 任务 5：实现二期 2A 后端 Ops 读模型与接口

**文件：**

- 创建：`internal/model/ops.go`
- 创建：`internal/op/ops.go`
- 创建：`internal/op/ops_test.go`
- 创建：`internal/server/handlers/ops.go`

- [ ] **步骤 1：定义 Ops 面板返回结构**

```go
type OpsCacheStatus struct {
    Enabled bool `json:"enabled"`
    TTLSeconds int `json:"ttl_seconds"`
    Threshold int `json:"threshold"`
    MaxEntries int `json:"max_entries"`
}

type OpsHealthStatus struct {
    DatabaseOK bool `json:"database_ok"`
    CacheOK bool `json:"cache_ok"`
    TaskRuntimeOK bool `json:"task_runtime_ok"`
    RecentErrorCount int `json:"recent_error_count"`
}
```

- [ ] **步骤 2：实现 2A 聚合逻辑，优先复用已有设置和缓存**

```go
func OpsCacheStatusGet(ctx context.Context) (*model.OpsCacheStatus, error) { /* 读取 semantic_cache_* settings */ }
func OpsQuotaSummaryGet(ctx context.Context) (*model.OpsQuotaSummary, error) { /* 聚合 API key rpm/tpm/max_cost/per_model_quota_json */ }
func OpsHealthStatusGet(ctx context.Context) (*model.OpsHealthStatus, error) { /* 读 DB ping、cache init 状态、最近 relay error */ }
func OpsSystemSummaryGet(ctx context.Context) (*model.OpsSystemSummary, error) { /* 复用系统信息、版本、导入导出能力摘要 */ }
```

- [ ] **步骤 3：注册 `/api/v1/ops/*` 接口**

```go
router.NewGroupRouter("/api/v1/ops").
  Use(middleware.Auth()).
  AddRoute(router.NewRoute("/cache", http.MethodGet).Handle(getOpsCache)).
  AddRoute(router.NewRoute("/quota", http.MethodGet).Handle(getOpsQuota)).
  AddRoute(router.NewRoute("/health", http.MethodGet).Handle(getOpsHealth)).
  AddRoute(router.NewRoute("/system", http.MethodGet).Handle(getOpsSystem))
```

权限建议：

```text
- cache / quota / health / system 读取：PermSettingsRead
- 不在 2A 新增写接口，继续复用现有 /api/v1/setting/set、/apikey/*、/log/*
```

- [ ] **步骤 4：运行后端验证**

运行：`go test ./internal/op -run TestOps -v`
预期：PASS

运行：`go build ./...`
预期：PASS

### 任务 6：实现二期 2A 前端 Ops 页面

**文件：**

- 创建：`web/src/api/endpoints/ops.ts`
- 创建：`web/src/components/modules/ops/index.tsx`
- 创建：`web/src/components/modules/ops/Cache.tsx`
- 创建：`web/src/components/modules/ops/Quota.tsx`
- 创建：`web/src/components/modules/ops/Health.tsx`
- 创建：`web/src/components/modules/ops/System.tsx`
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/en.json`
- 修改：`web/public/locale/zh_hant.json`

- [ ] **步骤 1：实现 Ops 二级页签**

```tsx
const OPS_TABS = ['cache', 'quota', 'health', 'system', 'audit'] as const;
```

要求：二期页签样式与一期 Analytics 保持完全一致。

- [ ] **步骤 2：做 `Cache`、`Quota`、`Health`、`System` 四个面板**

```tsx
// Cache.tsx
// 展示 semantic cache 配置摘要 + 跳转现有 setting 修改入口

// Quota.tsx
// 展示 API key 配额、用量、异常项摘要，不复制完整编辑表单

// Health.tsx
// 展示数据库、缓存、任务运行、最近错误计数、最近失败分组

// System.tsx
// 展示版本、公共 API Base URL、导入导出、日志保留、AI Route 服务配置摘要
```

要求：二期优先“聚合现有能力”，不要把 setting 页面整页复制一份。

- [ ] **步骤 3：运行前端验证**

运行：`cd web && pnpm lint`
预期：PASS

运行：`cd web && $env:NEXT_PUBLIC_APP_VERSION='dev'; pnpm build`
预期：PASS

### 任务 7：实现二期 2B 审计日志

**文件：**

- 修改：`internal/db/db.go`
- 创建：`internal/model/audit_log.go`
- 创建：`internal/op/audit_log.go`
- 创建：`internal/op/audit_log_test.go`
- 创建：`internal/server/middleware/audit.go`
- 创建：`internal/server/handlers/audit.go`
- 创建：`web/src/components/modules/ops/Audit.tsx`

- [ ] **步骤 1：先写审计模型与查询测试**

```go
type AuditLog struct {
    ID uint `json:"id" gorm:"primaryKey"`
    UserID uint `json:"user_id"`
    Username string `json:"username"`
    Action string `json:"action"`
    Method string `json:"method"`
    Path string `json:"path"`
    StatusCode int `json:"status_code"`
    Target string `json:"target"`
    CreatedAt int64 `json:"created_at"`
}
```

- [ ] **步骤 2：把模型接入 AutoMigrate**

```go
// internal/db/db.go
if err := db.AutoMigrate(
  // ...
  &model.AuditLog{},
  &migrate.MigrationRecord{},
); err != nil { /* ... */ }
```

- [ ] **步骤 3：只审计管理面写操作**

```go
// internal/server/middleware/audit.go
// 仅记录 /api/v1 下 POST/PUT/DELETE 的管理写操作
// 不记录 /v1 relay 流量，避免日志风暴
```

- [ ] **步骤 4：提供只读查询接口和前端列表**

运行：新增 `/api/v1/audit/list`、`/api/v1/audit/detail`，前端 `Audit.tsx` 以 list + detail dialog 形式展示。

- [ ] **步骤 5：运行后端验证**

运行：`go test ./internal/op -run TestAuditLog -v`
预期：PASS

运行：`go build ./...`
预期：PASS

### 任务 8：服务控制按钮单独留白，不混入本轮实现

**文件：**

- 修改：`web/src/components/modules/ops/System.tsx`

- [ ] **步骤 1：明确本轮不做 `停止服务`**

```text
原因：
1. 部署形态不明（本地进程 / Docker / systemd / PaaS）
2. “停止服务”一旦暴露为 HTTP 管理操作，容易把自己锁在门外
3. 与 OmniRoute 的展示需求相比，这属于高风险、低复用项
```

- [ ] **步骤 2：如果一定要做，只预留 restart 能力说明，不实现 stop**

```text
后续如用户单独确认，再基于 internal/update/core.go 和 internal/utils/shutdown 做“受限 restart”；
本计划默认不实现 stop service。
```

### 任务 9：完成二期联调与最终验收

**文件：**

- 修改：`web/src/components/modules/ops/*`
- 测试：`internal/op/ops_test.go`
- 测试：`internal/op/audit_log_test.go`

- [ ] **步骤 1：本地联调启动**

运行：`go run main.go start`
预期：服务正常启动

运行：`cd web && $env:NEXT_PUBLIC_API_BASE_URL='http://127.0.0.1:8080'; pnpm dev`
预期：前端正常启动

- [ ] **步骤 2：人工检查运维页一致性**

检查项：

```text
1. ops 页面的卡片样式、间距、动画、文案密度与 analytics 保持一致
2. 没有把 setting 里的复杂编辑器直接硬塞成另一个新后台
3. audit 列表滚动与 dialog 交互遵循现有 log / group 页风格
4. 空状态、错误状态、无权限状态使用现有 Toast / 文案模式
```

- [ ] **步骤 3：运行全量验证**

运行：`go build ./...`
预期：PASS

运行：`go test ./...`
预期：PASS

运行：`cd web && pnpm lint`
预期：PASS

运行：`cd web && $env:NEXT_PUBLIC_APP_VERSION='dev'; pnpm build`
预期：PASS

## 自检

- 一期没有新增数据库表，符合“最小改动 / 根因修复 / 兼容优先”。
- 二期把高风险项拆成 `2A 运维聚合` 与 `2B 审计日志`，避免范围耦合。
- UI 一致性被提升为硬约束，并落到了验收步骤里，不只是备注。
- `Evaluation` 页签明确采用 MVP 壳页策略，避免凭空发明 benchmark 存储。
- `stop service` 明确留白，不把部署相关风险混入本次实现。
