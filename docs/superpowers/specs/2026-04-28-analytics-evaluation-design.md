# Analytics Evaluation 最小接入设计

## 背景

当前 `Analytics -> Evaluation` 只有静态说明卡片，没有真实数据或可操作入口；这与 OmniRoute 概述页中“评估中心”应承担的入口作用不一致。

现有仓库里已经有两类可复用能力：

1. 分组可用性检测
   - 前端已有 `useTestGroup` / `useGroupTestProgress`
   - 入口位于分组卡片的 `testAvailability`
   - 对应接口为 `/api/v1/group/test` 与 `/api/v1/group/test/progress/:id`

2. AI 路由分析任务
   - 前端已有 `useGenerateAIRoute` / `useGenerateAIRouteProgress`
   - 入口位于分组页面 `AIRouteButton`
   - 对应接口为 `/api/v1/route/ai-generate`、`/progress/:id`、`/result/:id`、`/stream/:id`

本次设计不新增 benchmark 存储，不引入新的评测表或任务归档，只把 `Evaluation` 从说明壳页升级为“真实入口 + 轻量状态摘要”。

## 目标

1. 让 `Analytics -> Evaluation` 成为真实可用的评测入口，而不是静态说明页。
2. 复用现有后端与前端能力，保持最小改动和兼容优先。
3. 保持与现有 Analytics / Ops 一致的卡片、页头、按钮密度和空态风格。

## 非目标

1. 不新增数据库表，不保存 benchmark 历史。
2. 不在本轮实现趋势图、排行榜或多次评测对比。
3. 不把 Group 页面和 AI Route 页面完整复制进 Analytics。
4. 不新增新的后端评测接口；优先复用现有 `/api/v1/group/test` 与 `/api/v1/route/ai-generate/*`。

## 方案概述

`Evaluation` 改为三个功能卡片 + 一个状态摘要区：

1. 可用性检测入口卡
   - 展示现有“分组可用性检测”的能力说明
   - 提供“前往分组检测”按钮，跳转到 `group`
   - 如果当前系统存在分组，显示轻量提示：检测动作在分组卡片内触发

2. AI 路由分析入口卡
   - 展示现有 AI 路由能力说明
   - 提供“前往路由分析”按钮，跳转到 `group`
   - 如果存在进行中的 AI Route 任务，则展示当前任务状态摘要

3. 历史归档状态卡
   - 保留当前“暂无 benchmark 历史归档”的明确说明
   - 明确这是当前版本的有意留白，而非加载失败

4. 最近评测状态摘要区
   - 本地复用已有 progress/result 数据源做瞬时摘要
   - 不做持久化，只展示“当前会话 / 当前运行态”信息
   - 两类摘要：
     - 最近 AI Route 任务：状态、批次进度、结果是否 ready
     - 最近分组检测任务：完成数、通过数、失败数

## 数据来源与复用策略

### 1. 分组可用性检测

直接复用 `web/src/api/endpoints/group.ts` 中现有类型和 hooks：

- `useTestGroup`
- `useGroupTestProgress`
- `GroupTestProgress`
- `GroupTestResult`

本轮不把“发起检测”动作直接搬到 `Evaluation` 页内部，因为检测本身需要明确目标分组；当前最小接入仅展示入口与最近进度摘要，真正发起动作仍由 Group 页面负责。

### 2. AI 路由分析

直接复用 `web/src/api/endpoints/group.ts` 中现有能力：

- `useGenerateAIRoute`
- `useGenerateAIRouteProgress`
- `isGenerateAIRouteTerminal`
- 现有 `sessionStorage` 任务恢复逻辑

`Evaluation` 不直接复制 `AIRouteButton` 的全部交互，也不复刻完整进度弹窗；本轮只展示：

- 当前是否存在运行中任务
- 当前步骤 / 完成状态
- 已完成时的 group_count / route_count / item_count 摘要

如需真正发起任务，仍通过 Group 页面现有入口完成。

## 前端结构调整

### 保留的文件

- `web/src/components/modules/analytics/Evaluation.tsx`

### 新增的前端查询封装

在 `web/src/api/endpoints/analytics.ts` 新增轻量封装，用于给 `Evaluation` 页消费现有 group 能力：

1. 读取当前分组列表数量
   - 可直接复用现有 group list 接口或单独从 `group.ts` 导入 hook

2. 读取 AI Route 最近任务状态
   - 优先复用当前前端已有 sessionStorage 恢复逻辑
   - 必要时新增一个只读 helper，而不是新增后端接口

3. 读取最近分组检测任务状态
   - 复用当前前端进度对象
   - 不落后端历史

### Evaluation 页面布局

页面结构：

1. 顶部说明区
   - 标题和描述延续当前结构

2. 三张能力卡片
   - `可用性检测`
   - `AI 路由分析`
   - `历史归档`

3. 底部运行态摘要区
   - `最近 AI 路由任务`
   - `最近分组检测任务`

所有卡片继续复用现有 `rounded-3xl / border / bg-card / custom-shadow` 风格，不引入新的视觉体系。

## 交互设计

### 可用性检测卡

- 主文案：说明当前检测能力已存在
- 次文案：检测入口位于分组卡片
- CTA：`前往分组检测`
- 行为：`setActiveItem('group')`

### AI 路由分析卡

- 主文案：说明当前支持整表 / 单分组 AI 路由分析
- 次文案：实际入口位于分组页现有 `AI Route` 按钮
- CTA：`前往路由分析`
- 行为：`setActiveItem('group')`
- 如果检测到运行中任务：
  - 显示 `running / completed / failed / timeout`
  - 显示当前 step 或结果摘要

### 历史归档卡

- 明确显示“当前版本暂无 benchmark 历史归档”
- 不显示错误态图标，不暗示接口缺失

## 状态与错误处理

1. 无分组时
   - 可用性检测卡提示“当前没有可检测分组”
   - CTA 仍可跳去分组页

2. 无 AI Route 任务时
   - 摘要区显示“当前没有运行中的 AI 路由任务”

3. 有任务但 progress 丢失时
   - 视为无活跃任务处理
   - 不额外报错，不污染页面

4. 历史归档
   - 始终走明确 empty-state，不进入 error 分支

## 权限与边界

`Evaluation` 仍然是 Analytics 页面的一部分，权限继续继承 `PermStatsRead` 的访问边界。

本轮不新增需要独立权限判断的后端接口，因此不会引入新的 RBAC 风险。

## 验证标准

### 前端

1. `Analytics -> Evaluation` 不再是纯静态说明页。
2. 页面至少能看到：
   - 两个真实入口按钮
   - 一个明确的“暂无历史归档”状态卡
   - 一个运行态摘要区
3. 点击入口按钮能跳转到 `group` 页面。
4. 有进行中的 AI Route 或 Group Test 时，摘要区能显示当前状态。

### 构建

1. `cd web && pnpm lint`
2. `cd web && $env:NEXT_PUBLIC_APP_VERSION='v1.4.2'; pnpm build`

### 人工验收

1. 文案和卡片密度与现有 Analytics / Ops 一致。
2. 不出现“伪功能按钮”或无法解释的空白卡片。
3. 不新增新的后端数据表或接口依赖。

## 后续可扩展方向

如果后续要把 Evaluation 做成完整评测中心，再单独开下一阶段：

1. benchmark 历史归档表
2. 多次评测记录查询
3. 趋势图与 Top 排行
4. 评测结果导出
5. 统一的“发起评测”控制台
