# Semantic Cache / Evaluation / Nav Order 二期设计

## 背景

`2026-04-28-analytics-evaluation-design.md` 已经把 `Analytics -> Evaluation` 从静态说明页升级为“真实入口 + 轻量运行态摘要”，但用户后续又追加了三类明确需求：

1. 把语义缓存真正接入请求路径，而不只是保留设置项与占位实现
2. 把语义缓存相关设置补完整，并保持与现有 Setting UI 一致
3. 让顶部页面切换栏的顺序支持在设置页中自定义，并做服务端全局持久化

当前代码里与本次范围直接相关的现状如下：

1. `internal/utils/semantic_cache/cache.go`
   - 已有内存向量缓存、`Lookup` / `Store` / `Stats` / `Clear`
   - 但主请求链路里还没有真实调用点

2. `internal/utils/semantic_cache/embedder.go`
   - 仍是基于 `SHA-256` 的占位 embedding
   - 只能命中完全相同文本，不是真语义缓存

3. `internal/model/setting.go`
   - 已有 `semantic_cache_enabled` / `ttl` / `threshold` / `max_entries`
   - 没有独立 embedding 服务配置

4. `web/src/components/modules/navbar/nav-store.ts`
   - 页面顺序仍是前端硬编码常量
   - 只能本地决定切换方向，不能按服务端统一顺序渲染

因此本阶段的目标不是“再做一个新中心”，而是在现有架构上补齐一条真实、可降级、可观测的闭环。

## 目标

1. 将语义缓存接入真实文本请求链路，命中时直接返回缓存，失败时自动旁路，不阻断正常转发。
2. 为语义缓存增加独立 embedding 服务配置，且不复用主路由分组解析，避免递归与歧义。
3. 在 Setting 页补齐语义缓存与页面顺序两个配置区，保持当前卡片式 UI 风格。
4. 将 `Analytics -> Evaluation` 扩展为可反映语义缓存状态与当前运行成效的 v1 评估页。

## 非目标

1. 不新增数据库表；仍然复用 `settings` 表与进程内运行态。
2. 不在本轮实现图片、音频、视频、搜索、重排序等非文本请求的语义缓存。
3. 不在本轮实现长期 benchmark 历史、模型级缓存排行榜、成本节省精算报表。
4. 不做页面隐藏、不做按用户或角色区分导航顺序，只支持全局排序。
5. 不把语义缓存 embedding 请求接回 Octopus 自己的 `/v1/embeddings` 路由链路。

## 已确认的核心决策

1. 语义缓存必须是真缓存，不接受“哈希近似占位实现”。
2. embedding 来源采用独立服务配置，不走现有路由组与渠道选择。
3. 页面顺序持久化采用服务端全局设置，而不是浏览器本地偏好。
4. Evaluation 本轮只做 v1：入口、运行态摘要、缓存成效，不做完整评测平台。

## 一、语义缓存后端闭环

### 1.1 覆盖范围

本轮只覆盖主 LLM 文本请求链路中的非流式请求：

1. `/v1/chat/completions`
2. `/v1/responses`

明确旁路的请求：

1. `stream=true` 的流式请求
2. 图片、音频、视频、搜索、重排序、审核等 `media_relay` 路径
3. 无法稳定抽取文本输入的请求
4. embedding 服务调用失败或配置不完整的请求

这样可以确保第一版实现路径单一，先把“命中 / 未命中 / 旁路 / 回写”闭环做实，再考虑扩展到更多协议。

### 1.2 缓存隔离边界

语义缓存不能全局共享，否则会产生错误响应复用和 API Key 之间的数据串用。

因此每条缓存记录必须带命名空间，命名空间至少包含：

1. `api_key_id`
2. `endpoint_family`
3. `requested_model_name`

建议命名空间形式：

```text
<api_key_id>:<endpoint_family>:<requested_model_name>
```

只有同一命名空间内的请求才允许互相命中。

### 1.3 文本归一化

在进入 embedding 之前，对请求构造稳定的文本表示：

1. `chat/completions`
   - 按顺序提取可读文本内容
   - 保留角色边界
   - 忽略图片、音频、二进制等非文本片段

2. `responses`
   - 提取 `input` 中的文本型内容
   - 仅保留能够稳定序列化为纯文本的部分

归一化规则：

1. 去掉首尾空白
2. 统一连续空白为单个空格
3. 如果最终文本为空，直接旁路

### 1.4 Embedding 服务模型

新增独立设置键：

1. `semantic_cache_embedding_base_url`
2. `semantic_cache_embedding_api_key`
3. `semantic_cache_embedding_model`
4. `semantic_cache_embedding_timeout_seconds`

默认值建议：

1. `base_url=""`
2. `api_key=""`
3. `model=""`
4. `timeout_seconds="10"`

运行规则：

1. 直接向配置的 embedding 服务发起 OpenAI-compatible embedding 请求
2. 不经过 group / channel / route 解析
3. 不允许递归调用 Octopus 自己的公共转发面
4. `api_key` 可为空，便于兼容无鉴权内部服务

`runtime_enabled` 的语义定义为：

1. `semantic_cache_enabled=true`
2. `base_url` 有效
3. `model` 非空
4. 进程内缓存已初始化

只要其中一项不满足，语义缓存可视为“已配置但运行态不可用”，请求自动旁路。

### 1.5 请求路径闭环

在主文本 relay 路径上新增如下流程：

1. 判断当前请求是否属于语义缓存覆盖范围
2. 如果不属于，计入 `bypassed_requests`，继续正常转发
3. 如果属于，构造归一化文本并调用独立 embedding 服务
4. embedding 成功后，在命名空间内执行语义查找
5. 命中则直接返回缓存响应
6. 未命中则继续正常上游转发
7. 当上游返回成功的非流式文本响应后，回写缓存
8. 任意缓存相关失败都只影响缓存本身，不影响最终转发

### 1.6 响应存储边界

第一版只缓存成功的、完整拿到的非流式文本响应。

明确不缓存：

1. 上游错误响应
2. 流式响应
3. 非文本协议响应
4. 无法稳定序列化的中间状态

命中时直接返回缓存中的完整响应体；本轮不额外重写响应结构，也不引入单独的历史归档。

### 1.7 运行态统计

在现有 `hits / misses / hit_rate / current_entries / usage_rate` 基础上，增加一份更贴近请求链路的运行摘要：

1. `evaluated_requests`
2. `cache_hit_responses`
3. `cache_miss_requests`
4. `bypassed_requests`
5. `stored_responses`

边界定义：

1. 这些计数均为进程内运行期累计值
2. 重启进程或清空缓存后可以重置
3. 不落数据库，不做时间窗口查询

这份摘要用于 Evaluation v1，不意味着已经具备历史分析平台能力。

## 二、设置页补齐

### 2.1 新增语义缓存设置卡

设置页新增独立 `Semantic Cache` 卡片，保持与现有 `System` / `AI Route` 卡片一致的交互风格。

字段分组：

1. 基础参数
   - `semantic_cache_enabled`
   - `semantic_cache_ttl`
   - `semantic_cache_threshold`
   - `semantic_cache_max_entries`

2. Embedding 服务
   - `semantic_cache_embedding_base_url`
   - `semantic_cache_embedding_api_key`
   - `semantic_cache_embedding_model`
   - `semantic_cache_embedding_timeout_seconds`

交互要求：

1. 沿用当前 setting 卡片的 `onBlur` 保存方式
2. API Key 默认掩码显示
3. 保存失败仍走当前 toast / error 提示模式
4. 不新增复杂服务池，不支持多 embedding 服务轮换

### 2.2 设置验证

需要补充的后端校验：

1. `semantic_cache_embedding_base_url`
   - 允许空值
   - 非空时必须是 `http/https`

2. `semantic_cache_embedding_timeout_seconds`
   - 必须为大于 0 的整数

3. `semantic_cache_threshold`
   - 继续沿用当前 0-100 百分比输入
   - 运行时再转换为 0-1 浮点阈值

### 2.3 运行态一致性

任一语义缓存相关设置保存成功后，需要同步刷新进程内缓存运行态：

1. 启停缓存
2. 重新加载 TTL / threshold / max entries
3. 重新构建 embedding 客户端配置

要求保存行为“改完即生效”，不依赖进程重启。

## 三、全局页面顺序

### 3.1 持久化模型

新增服务端设置键：

1. `nav_order`

值格式为顶级页面 id 数组 JSON，例如：

```json
["home","channel","group","model","analytics","log","alert","ops","setting","user"]
```

### 3.2 默认顺序

默认顺序与当前 `ROUTES` 保持一致：

1. `home`
2. `channel`
3. `group`
4. `model`
5. `analytics`
6. `log`
7. `alert`
8. `ops`
9. `setting`
10. `user`

### 3.3 后端校验与兜底

`nav_order` 保存或读取时都要做清洗：

1. 非法 JSON 时回退默认顺序
2. 未知 id 直接丢弃
3. 重复 id 只保留第一次
4. 缺失的合法页面 id 按默认顺序追加到末尾

这样可以保证后续新增页面时，旧配置不会把新页面永久丢掉。

### 3.4 前端行为

前端改为两层状态：

1. `activeItem`
   - 仍然保留当前本地状态语义
   - 只负责当前页面选中项

2. `navOrder`
   - 改为从服务端设置加载
   - 未加载前先使用默认顺序

`direction` 的计算不再依赖硬编码常量，而是依赖当前生效的 `navOrder`。

### 3.5 设置页新增页面顺序卡

设置页新增 `Page Order` 卡片：

1. 使用项目里已有的 `@hello-pangea/dnd`
2. 仅支持拖拽排序
3. 拖拽完成即保存
4. 提供“恢复默认顺序”按钮

明确不做：

1. 页面隐藏
2. 每角色不同顺序
3. 多套导航布局

## 四、Evaluation v1 扩展

### 4.1 页面定位

`Evaluation` 保持轻量评估台定位，不改成完整控制台。

它应展示三类内容：

1. 现有动作入口
   - 可用性测试
   - AI 路由生成
   - 语义缓存评估

2. 最近运行摘要
   - AI Route 最近任务
   - Group Test 最近任务

3. 语义缓存成效
   - 当前状态
   - 当前容量
   - 当前命中情况
   - 运行期评估摘要

### 4.2 数据来源

保留现有前端本地运行态来源：

1. `useAnalyticsEvaluationRuntime()`
   - AI Route 最近任务
   - Group Test 最近任务

新增一个轻量 analytics 接口，专门返回语义缓存评估摘要：

1. `GET /api/v1/analytics/evaluation`

建议返回结构：

```json
{
  "semantic_cache": {
    "enabled": true,
    "runtime_enabled": true,
    "ttl_seconds": 3600,
    "threshold": 98,
    "max_entries": 1000,
    "current_entries": 132,
    "hits": 84,
    "misses": 27,
    "hit_rate": 75.68,
    "usage_rate": 13.2,
    "evaluated_requests": 118,
    "cache_hit_responses": 84,
    "cache_miss_requests": 27,
    "bypassed_requests": 7,
    "stored_responses": 21
  }
}
```

这样 `Evaluation` 不需要直接拼装多个 Ops 接口，也不会变成 Ops 页的镜像。

### 4.3 UI 结构

页面结构建议如下：

1. 顶部 3 张入口卡
   - `可用性测试`
   - `AI 路由生成`
   - `语义缓存评估`

2. 中部运行态摘要
   - `最近 AI 路由任务`
   - `最近分组检测任务`

3. 底部缓存成效卡
   - 运行状态：未启用 / 运行中 / 已配置但旁路
   - 命中情况：`hits / misses / hit_rate`
   - 容量情况：`current_entries / max_entries / usage_rate`
   - 运行摘要：`evaluated / hit / miss / bypass / stored`
   - CTA：前往设置页

### 4.4 交互边界

本轮不在 `Evaluation` 页内部直接发起新的 Group Test 或 AI Route 请求，继续沿用现有跳转策略：

1. `可用性测试` -> `group`
2. `AI 路由生成` -> `group`
3. `语义缓存评估` -> `setting`

这样可以避免在 Analytics 页复制整套复杂发起表单。

## 五、错误处理与降级规则

### 5.1 语义缓存

任一缓存相关错误都不得中断主请求：

1. embedding 服务失败 -> 记为 `bypass`，继续正常转发
2. 语义查找无匹配 -> 记为 `miss`，继续正常转发
3. 缓存内部异常 -> 记为 `bypass`，继续正常转发
4. 回写缓存失败 -> 仅放弃缓存，不影响当前响应
5. 设置不完整 -> `enabled=true` 但 `runtime_enabled=false`

### 5.2 页面顺序

1. 服务端没有 `nav_order` -> 使用默认顺序
2. `nav_order` 非法 -> 清洗后回退默认顺序
3. 前端加载失败 -> 临时使用默认顺序，不阻塞页面切换

### 5.3 Evaluation

1. 语义缓存评估接口失败 -> 仅该卡片显示空态或降级态
2. AI Route / Group Test 任务不存在 -> 保持当前 empty-state 逻辑
3. 历史 benchmark 继续明确标记为未纳入本轮范围

## 六、验证标准

### 6.1 后端

1. 语义缓存可在真实文本非流式请求上产生 `hit / miss / bypass / stored`
2. embedding 服务失败时，主请求仍能成功正常转发
3. 不同 `api_key_id + requested_model_name` 之间不会共享缓存响应
4. 修改语义缓存设置后，无需重启即可生效
5. `nav_order` 能正确清洗非法值并返回稳定顺序

### 6.2 前端

1. Setting 页出现新的语义缓存卡和页面顺序卡，视觉风格与现有卡片一致
2. 顶部导航顺序会跟随服务端设置变化
3. `Evaluation` 页能展示语义缓存状态与运行摘要
4. `Evaluation` 页的语义缓存入口能跳转到 `setting`

### 6.3 验证命令

1. `go test ./...`
2. `go build ./...`
3. `cd web && pnpm lint`
4. `cd web && $env:NEXT_PUBLIC_APP_VERSION='v1.4.2'; pnpm build`

### 6.4 人工验收

1. 缓存命中时不会影响正常响应结构
2. 缓存异常时用户侧看不到 5xx 回退噪音
3. 页面顺序刷新后保持一致，换浏览器仍一致
4. `Evaluation` 页面不再只有说明卡，而是有真实缓存成效内容

## 七、后续留白

本轮完成后，后续可单独开下一阶段处理：

1. 流式文本响应缓存
2. `/v1/embeddings` 文本输入缓存
3. 历史评估归档与趋势图
4. 渠道级 / 模型级缓存命中分析
5. 节省时延 / 节省成本的更精细统计
