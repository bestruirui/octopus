# 渠道编辑 - 高级设置：自动分组与自动同步功能说明

## 概述

在渠道编辑的高级设置中，有两个重要的自动化功能：**自动分组（Auto Group）** 和 **自动同步（Auto Sync）**。这两个功能可以帮助你自动管理模型列表和分组关系，减少手动维护的工作量。

---

## 一、自动同步（Auto Sync）

### 功能说明

自动同步功能会定期从渠道的 API 端点获取最新的模型列表，并自动更新渠道的模型配置。

### 触发条件

自动同步有以下几种触发方式：

1. **定时任务（主要方式）**
   - 系统启动时会立即执行一次同步任务
   - 之后按照设定的间隔周期性执行
   - 默认间隔：24 小时（可在设置中修改）
   - 配置项：`sync_llm_interval`（单位：小时）

2. **手动触发**
   - 在设置页面点击"同步渠道"按钮
   - 调用 API：`POST /api/v1/channel/sync`

3. **修改同步间隔**
   - 在设置页面修改"LLM 同步间隔"后，任务会立即使用新的间隔重新调度
   - 如果设置为 0，则停止自动同步任务

### 工作原理

1. **模型获取**：对于启用了自动同步的渠道，系统会调用对应的 API 接口获取可用模型列表
2. **差异对比**：将获取到的新模型列表与当前配置的模型列表进行对比
3. **自动更新**：
   - 新增的模型会自动添加到渠道的模型列表中
   - 已消失的模型会从渠道配置中移除
   - **保护自定义模型**：`custom_model` 字段中的模型会被保留，不会因为 API 未返回而被删除
   - 同时会删除与消失模型相关的分组关联（但不会删除 `custom_model` 中模型的分组关联）
4. **触发自动分组**：如果渠道启用了自动分组，会自动执行分组匹配

### 使用场景

- 渠道提供商经常更新模型列表（如 OpenAI、Anthropic 等）
- 希望自动获取最新可用的模型，无需手动维护
- 确保渠道配置始终与实际可用模型保持同步

### 配置方式

在渠道编辑页面底部的开关区域，启用 **"自动同步"** 开关即可。

### 注意事项

- 自动同步需要渠道的 API Key 和 Base URL 配置正确
- 只有启用了"自动同步"开关的渠道才会被同步
- 同步过程中会自动触发自动分组功能（如果启用）
- 消失的模型会被自动从所有分组中移除
- **`custom_model` 中的模型会被保留**：即使 API 未返回这些模型，它们也不会被删除，分组关联也会保留
- 系统启动时会立即执行一次同步，之后按设定间隔执行

### 如何查看同步状态

1. **查看最后同步时间**
   - 在设置页面可以看到"最后同步时间"
   - 调用 API：`GET /api/v1/channel/last-sync-time`

2. **检查同步间隔设置**
   - 在设置页面查看"LLM 同步间隔"配置
   - 默认值：24 小时
   - 如果设置为 0，则自动同步功能已停用

3. **手动触发同步**
   - 如果发现渠道配置了自动同步但没有同步，可以手动触发一次
   - 在设置页面点击"同步渠道"按钮
   - 检查日志查看是否有错误信息

### 模型过滤（match_regex）

#### 功能说明

`match_regex` 是渠道层级的"模型名称白名单"过滤器。当系统从渠道 API 拉取模型列表后，仅保留正则匹配命中的模型，过滤掉不需要的模型。该字段位于渠道编辑页面的 **"高级设置"** 中，对应 UI 标签为 **"匹配正则"**。

> 注意：UI 占位符提示为"可选：用于匹配请求模型名称的正则表达式"，但实际作用是**过滤渠道 `/v1/models` 接口返回的模型列表**，并不参与运行时请求路由。

#### 生效场景

| 场景 | 调用入口 |
|------|----------|
| 编辑渠道时点击"获取模型列表" | `internal/server/handlers/channel.go` 的 `FetchModels` 处理 |
| 自动同步任务（`auto_sync` 启用） | `internal/task/sync.go` 调用 `helper.FetchModels` |

- 留空 → 不过滤，渠道返回的模型全部保留
- 非空 → 仅保留正则命中的模型
- 编译失败 → 整个同步动作返回错误，渠道模型列表不更新

核心实现位于 `internal/helper/fetch.go:31-47`。

#### 正则引擎

- 使用 `github.com/dlclark/regexp2`，**ECMAScript 模式**（与 JavaScript 正则语法一致）
- 调用 `re.MatchString(model)` 进行**部分匹配**判定，命中即保留
- 支持 Go 原生 `regexp` 包不支持的特性：前向断言 `(?=)` / `(?!)`、后向断言 `(?<=)` / `(?<!)`

#### 填写示例

| 目的 | 正则 |
|------|------|
| 不过滤（默认） | 留空 |
| 只保留 GPT 系列 | `^gpt-` |
| 只保留 GPT-4 系列 | `^gpt-4` |
| 同时保留 gpt 和 claude 系列 | `gpt\|claude` |
| 排除嵌入模型 | `^(?!.*embedding).*$` |
| 排除带 `-realtime` 后缀 | `^(?!.*-realtime).*$` |
| 仅保留指定几个模型 | `^(gpt-4o\|claude-3-5-sonnet\|gemini-2\.0-flash)$` |
| 排除 dall-e、whisper、tts 等非对话模型 | `^(?!(dall-e\|whisper\|tts)).*$` |

#### 使用建议

- **部分匹配特性**：写 `gpt-4` 时所有名字含 `gpt-4` 的模型都会命中。如需严格全串匹配，请用 `^...$` 包裹
- **特殊字符要转义**：版本号中的 `.` 应写为 `\.`，例如 `gemini-2\.0-flash`
- **过严风险**：正则过于严格可能导致同步后模型列表为空，建议先在浏览器控制台用相同正则验证（语法兼容 JS）
- **失败处理**：正则编译失败或匹配异常会导致同步任务返回错误并跳过该渠道，错误信息可在日志中查看

#### 与"分组匹配正则"的区别

| 字段位置 | 作用阶段 | 作用对象 |
|---------|---------|---------|
| **渠道**编辑页 → 高级设置 → 匹配正则（`channel.match_regex`） | 拉取/同步模型列表时 | 过滤渠道返回的模型，决定哪些模型存入渠道配置 |
| **分组**编辑页 → 匹配正则（`group.match_regex`） | 自动分组时（`auto_group=regex`） | 决定渠道中的哪些模型会被加入到该分组 |

二者独立配置，作用阶段不同，请勿混淆。

---

## 二、自动分组（Auto Group）

### 功能说明

自动分组功能会根据设定的匹配规则，自动将渠道的模型添加到对应的分组中。

### 匹配模式

自动分组提供了 4 种匹配模式：

#### 1. 不自动分组（None）
- 默认选项，不执行任何自动分组操作
- 需要手动管理模型与分组的关联关系

#### 2. 模糊匹配（Fuzzy）
- **匹配规则**：模型名称包含分组名称（不区分大小写）
- **示例**：
  - 分组名称：`gpt-4`
  - 会匹配：`gpt-4`, `gpt-4-turbo`, `gpt-4-32k`, `gpt-4o` 等
  - 不会匹配：`gpt-3.5-turbo`, `claude-3` 等

#### 3. 准确匹配（Exact）
- **匹配规则**：模型名称与分组名称完全相同（不区分大小写）
- **示例**：
  - 分组名称：`gpt-4`
  - 会匹配：`gpt-4`, `GPT-4`
  - 不会匹配：`gpt-4-turbo`, `gpt-4o` 等

#### 4. 正则匹配（Regex）
- **匹配规则**：使用分组的正则表达式进行匹配
- **优先级**：
  - 如果分组设置了 `match_regex`，则使用该正则表达式
  - 如果分组未设置正则表达式，则退化为准确匹配模式
- **示例**：
  - 正则表达式：`^gpt-4.*`
  - 会匹配：`gpt-4`, `gpt-4-turbo`, `gpt-4o` 等
  - 不会匹配：`gpt-3.5-turbo`, `claude-3` 等

### 工作流程

1. **触发时机**：
   - 手动保存渠道配置时
   - 自动同步任务执行后
   
2. **匹配过程**：
   - 系统遍历所有分组
   - 根据选择的匹配模式，将渠道的模型与分组进行匹配
   - 匹配成功的模型会自动添加到对应分组中

3. **批量操作**：
   - 一次性处理所有匹配的模型
   - 避免重复添加已存在的关联

### 使用场景

#### 场景 1：统一管理 GPT-4 系列模型
- 创建分组：`gpt-4`
- 自动分组模式：**模糊匹配**
- 效果：所有包含 `gpt-4` 的模型都会自动加入该分组

#### 场景 2：精确控制特定模型
- 创建分组：`gpt-4-turbo`
- 自动分组模式：**准确匹配**
- 效果：只有名称完全为 `gpt-4-turbo` 的模型会加入该分组

#### 场景 3：复杂的模型分类
- 创建分组：`Claude 3 系列`
- 分组正则表达式：`^claude-3-(opus|sonnet|haiku)`
- 自动分组模式：**正则匹配**
- 效果：只有 Claude 3 的 opus、sonnet、haiku 三个版本会加入该分组

### 配置方式

在渠道编辑页面的 **"高级设置"** 折叠面板中，选择 **"自动分组"** 下拉菜单，选择合适的匹配模式。

### 注意事项

- 自动分组不会删除已存在的分组关联，只会添加新的关联
- 正则匹配模式使用 ECMAScript 标准的正则表达式语法
- 自动分组会在每次模型列表更新后自动执行

---

## 三、两者配合使用

### 推荐配置

**最佳实践**：同时启用自动同步和自动分组

```
✓ 自动同步：启用
✓ 自动分组：模糊匹配（或根据需求选择其他模式）
```

### 工作流程

1. **定时同步**：系统定期从 API 获取最新模型列表
2. **更新模型**：自动添加新模型，移除已下线的模型
3. **自动分组**：新增的模型根据匹配规则自动加入对应分组
4. **清理关联**：已下线的模型会从所有分组中自动移除

### 优势

- **零维护**：无需手动更新模型列表和分组关系
- **实时性**：始终使用最新可用的模型
- **一致性**：所有渠道的模型分组规则统一管理
- **可靠性**：自动清理无效的模型和关联关系

---

## 四、技术实现细节

### 代码位置

- **前端配置界面**：`web/src/components/modules/channel/Form.tsx`
- **API 定义**：`web/src/api/endpoints/channel.ts`
- **后端模型定义**：`internal/model/channel.go`
- **同步任务**：`internal/task/sync.go`
- **任务调度**：`internal/task/task.go`
- **任务初始化**：`internal/task/init.go`
- **自动分组逻辑**：`internal/helper/channel.go`
- **渠道操作**：`internal/op/channel.go`
- **API 处理器**：`internal/server/handlers/channel.go`

### 数据库字段

```go
type Channel struct {
    AutoSync   bool          `json:"auto_sync" gorm:"default:false"`
    AutoGroup  AutoGroupType `json:"auto_group" gorm:"default:0"`
    MatchRegex *string       `json:"match_regex"` // 模型过滤正则（ECMAScript），仅在拉取/同步模型列表时生效
    // ...
}

type AutoGroupType int
const (
    AutoGroupTypeNone  AutoGroupType = 0 // 不自动分组
    AutoGroupTypeFuzzy AutoGroupType = 1 // 模糊匹配
    AutoGroupTypeExact AutoGroupType = 2 // 准确匹配
    AutoGroupTypeRegex AutoGroupType = 3 // 正则匹配
)
```

### 系统设置

```go
// 默认设置
{Key: SettingKeySyncLLMInterval, Value: "24"} // 默认24小时同步一次LLM
```

### 任务注册流程

```go
// 1. 系统启动时初始化任务 (cmd/start.go)
task.Init()
go task.RUN()

// 2. 注册同步任务 (internal/task/init.go)
syncLLMIntervalHours, _ := op.SettingGetInt(model.SettingKeySyncLLMInterval)
syncLLMInterval := time.Duration(syncLLMIntervalHours) * time.Hour
Register(string(model.SettingKeySyncLLMInterval), syncLLMInterval, true, SyncModelsTask)
// 参数说明：
// - 任务名称：sync_llm_interval
// - 执行间隔：从设置中读取（默认24小时）
// - runOnStart: true（启动时立即执行一次）
// - 任务函数：SyncModelsTask

// 3. 任务执行器 (internal/task/task.go)
// - 如果 runOnStart=true，启动时立即执行
// - 之后按照 interval 周期性执行
// - 支持动态更新间隔（无需重启）
```

### 同步任务执行逻辑

```go
// SyncModelsTask 函数流程 (internal/task/sync.go)
func SyncModelsTask() {
    // 1. 获取所有渠道
    channels, _ := op.ChannelList(ctx)
    
    // 2. 遍历每个渠道
    for _, channel := range channels {
        // 只处理启用了自动同步的渠道
        if !channel.AutoSync {
            continue
        }
        
        // 3. 调用 API 获取模型列表
        fetchModels, _ := helper.FetchModels(ctx, channel)
        
        // 4. 排除已在 custom_model 中的模型
        customModels := xstrings.SplitTrimCompact(",", channel.CustomModel)
        for _, m := range fetchModels {
            if m != "" && m != " " {
               if _, isCustom := customModels[m]; !isCustom {
                  mergedModels[m] = struct{}{}
               }
            }
         }
        
        // 5. 对比新旧模型列表
        oldModels := strings.Split(channel.Model, ",")
        deletedModels, addedModels := diff.Diff(oldModels, mergedModels)
        
        // 6. 更新渠道模型配置
        if len(deletedModels) > 0 || len(addedModels) > 0 {
            op.ChannelUpdate(&model.ChannelUpdateRequest{
                ID:    channel.ID,
                Model: &mergedModelStr,
            }, ctx)
        }
        
        // 7. 删除消失模型的分组关联（但保留 custom_model 中的模型）
        if len(deletedModels) > 0 {
            // 过滤掉 custom_model 中的模型
            actualDeletedModels := filterOutCustomModels(deletedModels, customModels)
            op.GroupItemBatchDelByChannelAndModels(keys, ctx)
        }
        
        // 8. 执行自动分组
        if len(mergedModels) > 0 {
            helper.ChannelAutoGroup(&channel, ctx)
        }
    }
    
    // 9. 更新全局模型价格表
    // ...
    
    // 10. 记录最后同步时间
    lastSyncModelsTime = time.Now()
}
```

### 自动分组执行逻辑

```go
// ChannelAutoGroup 函数流程 (internal/helper/channel.go)
func ChannelAutoGroup(channel *model.Channel, ctx context.Context) {
    // 1. 检查是否启用自动分组
    if channel.AutoGroup == model.AutoGroupTypeNone {
        return
    }
    
    // 2. 获取所有分组
    groups, _ := op.GroupList(ctx)
    
    // 3. 获取渠道的所有模型
    channelModelNames := strings.Split(channel.Model, ",")
    
    // 4. 遍历每个分组
    for _, group := range groups {
        matchedModelNames := []string{}
        
        // 5. 根据匹配模式进行匹配
        switch channel.AutoGroup {
        case model.AutoGroupTypeExact:
            // 准确匹配：模型名 == 分组名
            if strings.EqualFold(modelName, group.Name) {
                matchedModelNames = append(matchedModelNames, modelName)
            }
            
        case model.AutoGroupTypeFuzzy:
            // 模糊匹配：模型名包含分组名
            if strings.Contains(strings.ToLower(modelName), groupNameLower) {
                matchedModelNames = append(matchedModelNames, modelName)
            }
            
        case model.AutoGroupTypeRegex:
            // 正则匹配：使用分组的正则表达式
            re, _ := regexp2.Compile(group.MatchRegex, regexp2.ECMAScript)
            matched, _ := re.MatchString(modelName)
            if matched {
                matchedModelNames = append(matchedModelNames, modelName)
            }
        }
        
        // 6. 批量添加匹配的模型到分组
        if len(matchedModelNames) > 0 {
            op.GroupItemBatchAdd(group.ID, items, ctx)
        }
    }
}
```

### API 接口

```go
// 手动触发同步
POST /api/v1/channel/sync
// 实现：直接调用 task.SyncModelsTask()

// 获取最后同步时间
GET /api/v1/channel/last-sync-time
// 返回：lastSyncModelsTime (time.Time)

// 更新同步间隔
POST /api/v1/setting/set
Body: {
    "key": "sync_llm_interval",
    "value": "24"  // 小时
}
// 实现：更新设置后调用 task.Update() 动态调整任务间隔
```

---

## 五、常见问题

### Q1：自动同步多久执行一次？
A：
- 默认间隔：24 小时
- 可在设置页面修改"LLM 同步间隔"配置（单位：小时）
- 系统启动时会立即执行一次，之后按设定间隔周期性执行
- 如果设置为 0，则停止自动同步功能

### Q2：为什么我的渠道配置了自动同步但没有同步？
A：可能的原因：
1. **同步间隔设置为 0**：检查设置页面的"LLM 同步间隔"是否为 0
2. **系统刚启动**：等待下一个同步周期，或手动触发同步
3. **API Key 或 Base URL 配置错误**：检查渠道配置是否正确
4. **网络问题**：检查系统日志，查看是否有网络错误
5. **渠道未启用自动同步开关**：确认渠道编辑页面底部的"自动同步"开关已开启

解决方法：
- 在设置页面查看"最后同步时间"
- 手动点击"同步渠道"按钮触发一次同步
- 查看系统日志排查错误信息

### Q3：自动分组会覆盖手动添加的分组关联吗？
A：不会。自动分组只会添加新的关联，不会删除已存在的关联。只有当模型从渠道中消失时，相关的分组关联才会被自动清理。

### Q4：自动同步会删除我手动添加的自定义模型吗？
A：不会。`custom_model` 字段中的模型会被保护，即使 API 未返回这些模型，它们也会被保留在渠道的模型列表中，相关的分组关联也不会被删除。

### Q5：如果 API Key 失效，自动同步会怎样？
A：同步任务会记录错误日志，但不会影响其他渠道的同步。建议定期检查日志，确保所有渠道的 API Key 有效。

### Q6：正则匹配模式下，如何设置分组的正则表达式？
A：在分组编辑页面，有一个 "匹配正则" 字段，填入符合 ECMAScript 标准的正则表达式即可。

### Q7：可以为不同渠道设置不同的自动分组模式吗？
A：可以。每个渠道的自动分组模式是独立配置的，互不影响。

### Q8：如何手动触发一次同步？
A：有两种方式：
1. 在设置页面点击"同步渠道"按钮
2. 调用 API：`POST /api/v1/channel/sync`

### Q9：修改同步间隔后需要重启系统吗？
A：不需要。修改"LLM 同步间隔"后，任务会立即使用新的间隔重新调度，无需重启。

---

## 六、故障排查指南

### 问题 1：渠道配置了自动同步但没有执行

**排查步骤：**

1. **检查同步间隔设置**
   ```
   设置页面 → LLM 同步间隔
   确认值不为 0（0 表示停用）
   ```

2. **查看最后同步时间**
   ```
   设置页面 → 最后同步时间
   如果时间很久远，说明任务可能未正常运行
   ```

3. **检查渠道配置**
   ```
   渠道编辑页面 → 底部开关区域
   确认"自动同步"开关已开启
   ```

4. **验证 API 配置**
   ```
   - Base URL 是否正确
   - API Key 是否有效
   - 网络是否可达
   ```

5. **手动触发测试**
   ```
   设置页面 → 点击"同步渠道"按钮
   观察是否有错误提示
   ```

6. **查看系统日志**
   ```
   查找关键词：
   - "sync models task"
   - "failed to fetch models"
   - "failed to list channels"
   ```

### 问题 2：自动分组没有生效

**排查步骤：**

1. **确认自动分组模式**
   ```
   渠道编辑 → 高级设置 → 自动分组
   确认不是"不自动分组"
   ```

2. **检查分组配置**
   ```
   - 分组名称是否正确
   - 正则匹配模式下，正则表达式是否正确
   ```

3. **验证匹配规则**
   ```
   模糊匹配：模型名包含分组名
   准确匹配：模型名等于分组名
   正则匹配：模型名匹配正则表达式
   ```

4. **手动触发测试**
   ```
   - 保存渠道配置（会触发自动分组）
   - 或手动触发同步任务
   ```

### 问题 3：同步后模型消失了

**可能原因：**

1. **API 返回的模型列表为空**
   - API Key 权限不足
   - API 端点返回错误
   - 网络问题导致请求失败

2. **匹配正则过滤了模型**
   - 检查渠道的"匹配正则"配置
   - 该正则会过滤 API 返回的模型列表

**解决方法：**
- 检查 API Key 权限
- 测试 API 端点是否正常
- 检查"匹配正则"配置是否过于严格

### 问题 4：同步任务执行很慢

**可能原因：**

1. **渠道数量过多**
   - 每个渠道都需要调用 API
   - 串行执行可能耗时较长

2. **网络延迟**
   - API 端点响应慢
   - 代理配置不当

3. **超时设置**
   - 默认超时 30 分钟
   - 可能在等待慢速 API 响应

**优化建议：**
- 只对必要的渠道启用自动同步
- 检查网络和代理配置
- 增加同步间隔，减少执行频率

### 问题 5：修改同步间隔后没有生效

**排查步骤：**

1. **确认设置已保存**
   ```
   设置页面 → LLM 同步间隔
   刷新页面确认值已更新
   ```

2. **无需重启系统**
   ```
   任务会自动使用新的间隔
   下一次执行时生效
   ```

3. **查看日志确认**
   ```
   查找关键词：
   - "task sync_llm_interval interval updated"
   ```

### 调试技巧

1. **启用调试日志**
   ```
   配置文件中设置 log.level = "debug"
   可以看到更详细的执行信息
   ```

2. **使用 API 测试**
   ```bash
   # 手动触发同步
   curl -X POST http://localhost:8080/api/v1/channel/sync \
     -H "Authorization: Bearer YOUR_TOKEN"
   
   # 查看最后同步时间
   curl http://localhost:8080/api/v1/channel/last-sync-time \
     -H "Authorization: Bearer YOUR_TOKEN"
   ```

3. **检查数据库**
   ```sql
   -- 查看渠道配置
   SELECT id, name, auto_sync, auto_group FROM channels;
   
   -- 查看同步间隔设置
   SELECT * FROM settings WHERE key = 'sync_llm_interval';
   ```

---

## 七、总结

- **自动同步**：自动获取和更新渠道的模型列表，保持配置与实际可用模型同步
- **自动分组**：根据匹配规则自动将模型添加到分组，支持模糊、准确、正则三种匹配模式
- **配合使用**：两者结合可以实现完全自动化的模型管理，大幅减少运维工作量

建议根据实际需求选择合适的配置方式，对于经常更新模型的渠道（如 OpenAI），强烈推荐启用自动同步和自动分组功能。
