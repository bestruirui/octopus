# Model Market 替换 Price 页设计

## 背景

当前 `octopus` 的 `model` 页面本质上是“模型价格管理”页：

1. 页面主体只展示 `name + input/output/cache_read/cache_write`
2. 支持搜索、`priced/free` 筛选、创建、编辑、删除
3. 价格刷新入口主要放在设置页 `LLMPrice`

你提供的参考页面则是更偏“模型广场”的信息架构：

1. 模型卡片优先展示模型覆盖面和运行态摘要
2. 页面头部有总量统计和轻量控制
3. 卡片支持展开查看更多细节
4. 价格只是模型信息的一部分，不再是页面唯一主题

本次需求是把参考页面里的“模型广场”结构并入 `octopus`，替换现有价格页的主体展示，但不能丢失已有价格能力，并且要保持整个后台 UI 风格一致。

## 目标

1. 用 `octopus` 现有视觉体系重做 `model` 页面，使其从“价格卡片列表”升级为“模型广场”。
2. 保留现有价格相关能力：
   - 模型价格展示
   - 模型价格编辑
   - 模型创建 / 删除
   - 手动刷新价格
   - 设置页中的自动刷新周期配置
3. 不新增数据库表，不变更现有价格存储结构。
4. 不照搬参考 HTML/CSS，而是复用现有导航、toolbar、虚拟滚动、卡片、tooltip、overlay、motion 交互。

## 非目标

1. 不新建一个独立“模型广场”路由；继续复用现有 `model` 路由入口。
2. 不引入参考页面的整套样式类、分页器或独立设计系统。
3. 不重写现有设置页价格配置逻辑。
4. 不在本轮引入新的历史统计表或离线报表。
5. 不顺手改动无关页面的导航结构或视觉规范。

## 已确认语义映射

参考页面中的指标语义，在 `octopus` 中按以下方式落地：

1. `账号数` 映射为 `覆盖渠道数`
   - 含义：声明支持该模型的渠道数量

2. `令牌数` 映射为 `渠道 key 数`
   - 含义：这些覆盖渠道下已启用 key 的总数

3. `标签区`
   - 第一个标签是模型供应商标签
   - 后续标签是覆盖该模型的渠道名称

4. `平均延迟`
   - 使用现有 `StatsModel` 聚合后的平均 `wait_time`

5. `成功率`
   - 使用现有 `request_success / (request_success + request_failed)` 聚合计算

6. 页面顶部汇总
   - `模型总数`
   - `覆盖渠道`：当前可见模型的渠道关联总数
   - `去重渠道`：当前可见模型覆盖到的唯一渠道数
   - `平均延迟`
   - `上次价格更新时间`

## 方案概述

选择方案二：在现有 `model` 模块上重做为“模型广场”视图。

核心原则是：

1. 页面壳不变
   - 继续保留当前 `route id = model`
   - 继续使用当前 navbar / toolbar / `VirtualizedGrid`

2. 数据接口分层
   - 保留原有 `/api/v1/model/list`、`/update`、`/delete`、`/update-price`、`/last-update-time`
   - 新增一个只读聚合接口 `/api/v1/model/market`
   - 新接口只负责给前端提供“模型广场 ViewModel”，不替代价格写接口

3. 交互边界保持兼容
   - 搜索和 `priced/free` 筛选继续沿用 toolbar
   - 创建 / 编辑 / 删除继续沿用原有模型管理能力
   - 价格编辑仍由现有 overlay 完成
   - 新增卡片“展开”态承载更多只读信息

4. 视觉统一优先
   - 不复制参考页面的 class 名
   - 使用现有 `rounded-3xl / border / bg-card / custom-shadow / tooltip / badge` 体系实现

## 后端设计

### 新增只读聚合接口

新增：

- `GET /api/v1/model/market`

权限边界：

- 与现有 `GET /api/v1/model/list` 相同，继续挂在 `PermSettingsRead`

### 新接口返回的数据结构

建议后端返回两个层级：

1. 页面级汇总
2. 模型卡片级列表

示意结构：

```json
{
  "summary": {
    "model_count": 120,
    "coverage_count": 380,
    "unique_channel_count": 42,
    "average_latency_ms": 812,
    "last_update_time": "2026-04-29T10:32:15+08:00"
  },
  "items": [
    {
      "name": "gpt-5.2",
      "input": 1.25,
      "output": 10,
      "cache_read": 0.125,
      "cache_write": 1.25,
      "channel_count": 8,
      "enabled_key_count": 14,
      "average_latency_ms": 796,
      "success_rate": 0.984,
      "request_success": 1240,
      "request_failed": 20,
      "provider_hint": "openai",
      "channels": [
        {
          "channel_id": 12,
          "channel_name": "NMapi",
          "enabled": true,
          "enabled_key_count": 3
        }
      ]
    }
  ]
}
```

### 数据来源

不新增存储，直接基于现有内存 / 缓存聚合：

1. 价格数据：`llmModelCache`
2. 模型-渠道关系：`ChannelLLMList()` / `channelCache`
3. 渠道启用状态与 key 数：`channelCache`
4. 成功率与延迟：`statsModelCache`
5. 价格更新时间：现有 `price.GetLastUpdateTime()`

### 聚合规则

对每个模型：

1. 取该模型的价格记录
2. 找出所有声明支持该模型的渠道
3. `channel_count = 覆盖渠道条数`
4. `enabled_key_count = 所有已启用 key 数量之和`
5. `average_latency_ms = 聚合 wait_time / 聚合请求总数`
6. `success_rate = request_success / (request_success + request_failed)`
7. `channels` 明细作为展开态数据

页面汇总基于当前返回全量数据，由后端直接给出，前端不重复计算总览逻辑。

### 兼容性要求

1. 原有 `/api/v1/model/list` 不变，避免影响现有创建/编辑/删除逻辑
2. 原有 `/api/v1/model/channel` 可以保留，但新页面尽量直接消费 `/api/v1/model/market`
3. 不修改数据库 schema
4. 新 DTO 使用 `TableName() string { return "-" }`，避免 GORM 误映射

## 前端设计

### 路由与导航

1. 内部路由 id 继续使用 `model`
2. 用户可见文案从“价格”改为“模型广场”
3. PWA/manifest 中对应名称同步更新

这样不影响现有 `route/config.tsx` 与 preload 逻辑，但用户认知上完成页面替换。

### 页面结构

新的 `model` 页面建议分为两层：

1. 页面头部摘要区
   - 标题：模型广场
   - 副信息：模型总数、覆盖渠道、去重渠道、平均延迟、上次更新时间
   - 轻量操作：立即更新价格

2. 卡片网格区
   - 继续走 toolbar 的搜索 / 筛选 / grid/list
   - grid 模式展示广场卡片
   - list 模式展示压缩版广场条目，而不是旧价格条目

### 卡片结构

每张卡片包含：

1. 左上
   - 模型图标
   - 模型名称

2. 中部元信息
   - 覆盖渠道数
   - 渠道 key 数
   - 平均延迟
   - 成功率

3. 标签区
   - 供应商标签
   - 渠道名称标签

4. 右侧操作区
   - 复制模型名
   - 展开/收起
   - 编辑价格
   - 删除

### 展开态内容

展开态不承担写操作主入口，只做详情补充：

1. 价格详情
   - 输入价格
   - 输出价格
   - 缓存读价格
   - 缓存写价格

2. 统计详情
   - 成功请求数
   - 失败请求数
   - 成功率
   - 平均延迟

3. 覆盖渠道列表
   - 渠道名称
   - 渠道是否启用
   - 渠道下启用 key 数

### 价格编辑保留方式

价格编辑继续复用当前 overlay 交互：

1. 用户从卡片点击编辑
2. 打开当前 `ModelEditOverlay`
3. 保存仍走 `/api/v1/model/update`

也就是说：

- “模型广场”负责重排展示信息
- “价格编辑”仍然是当前已验证的交互能力

### 价格刷新保留方式

1. 设置页 `LLMPrice` 卡片保留自动刷新周期和完整刷新入口
2. 模型广场页头部增加轻量“立即更新价格”按钮
3. 页面只显示上次更新时间，不复制周期配置表单

这样能保持：

- 设置页仍是系统配置入口
- 模型广场页拥有必要的快捷操作

## UI 一致性要求

### 必须保持一致的部分

1. 页面容器与留白节奏
2. toolbar 搜索 / 筛选入口
3. 现有 motion 动画节奏
4. badge / tooltip / overlay / destructive 交互风格
5. `VirtualizedGrid` 的滚动和大列表性能策略

### 明确不照搬的部分

1. 不照搬参考页分页器
   - 继续使用虚拟滚动

2. 不照搬参考页表格视图
   - 现有 `list` 视图承担“紧凑视图”角色

3. 不照搬参考页原始颜色 token
   - 继续使用 `octopus` 主题变量

## 文件级设计建议

### 后端

建议新增或修改：

1. `internal/model/llm.go`
   - 增加模型广场 DTO

2. `internal/op/model_market.go`
   - 承载聚合逻辑

3. `internal/op/stats.go`
   - 增加 `StatsModelList()` 之类的只读 helper

4. `internal/server/handlers/model.go`
   - 注册 `/market`

### 前端

建议新增或修改：

1. `web/src/api/endpoints/model.ts`
   - 增加 `ModelMarketSummary` / `ModelMarketItem` / `useModelMarket()`

2. `web/src/components/modules/model/index.tsx`
   - 改为页面摘要 + 虚拟卡片列表

3. `web/src/components/modules/model/Item.tsx`
   - 重做为模型广场卡片

4. `web/src/components/modules/model/ItemOverlays.tsx`
   - 保留价格编辑 / 删除确认

5. `web/src/components/modules/model/MarketSummary.tsx`
   - 页面摘要区独立组件

6. `web/src/lib/model-icons.tsx`
   - 在现有图标匹配基础上补充供应商显示标签能力

7. `web/public/locale/*.json`
   - 调整导航和模型广场文案

8. `web/public/manifest.json`
   - 同步用户可见入口名称

## 错误与空态处理

1. 无价格数据
   - 显示空态，不报错

2. 统计缺失
   - 延迟显示 `—`
   - 成功率显示 `—`

3. 模型没有渠道覆盖
   - 允许存在，但卡片要明确显示 `0`

4. 刷新价格失败
   - 保持现有 toast 行为

5. 长标签过多
   - 卡片只展示部分标签，其余通过 tooltip 或“+N”收纳

## 验证标准

### 后端

1. `/api/v1/model/market` 能返回汇总和模型列表
2. 单模型聚合出的渠道数、key 数、成功率、平均延迟正确
3. 不影响原有 `/api/v1/model/list`、`/update`、`/delete`、`/update-price`

### 前端

1. `model` 页面从价格卡片切换为模型广场卡片
2. 搜索与 `priced/free` 筛选保持可用
3. 复制、展开、编辑、删除保持可用
4. 立即更新价格按钮可用
5. list/grid 两种视图都可读

### 构建与人工验收

1. `go test ./internal/op/...`
2. `go test ./internal/server/handlers/...`
3. `cd web && pnpm lint`
4. `cd web && $env:NEXT_PUBLIC_APP_VERSION='dev'; pnpm build`
5. 人工检查：
   - 导航文案
   - 页面摘要
   - 卡片展开
   - 价格编辑
   - 长列表滚动性能

## 风险与取舍

1. 风险：`StatsModel` 当前没有直接对外 list helper
   - 处理：在 `op` 层新增只读 snapshot helper，不改持久化逻辑

2. 风险：单卡标签过多导致布局拥挤
   - 处理：只展示前几项，其余折叠

3. 风险：如果完全照搬参考页结构，会和现有后台风格割裂
   - 处理：只借用信息架构，不借用样式体系

4. 风险：页面“模型广场化”后用户担心价格功能丢失
   - 处理：保留编辑 overlay、设置页刷新卡片、页面快捷刷新按钮
