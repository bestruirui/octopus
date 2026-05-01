<div align="center">

<img src="web/public/logo.svg" alt="Octopus Logo" width="120" height="120">

### Octopus

**为个人打造的简单、美观、优雅的 LLM API 聚合与负载均衡服务**

简体中文 | [English](README.md)

</div>


## ✨ 特性

- 🔀 **多渠道聚合** - 支持接入多个 LLM 供应商渠道，统一管理
- 🔑 **多Key支持** - 单渠道支持配置多 Key
- ⚡ **智能优选** - 单渠道多端点，智能选择延迟最小的端点请求
- ⚖️ **负载均衡** - 支持轮询、随机、故障转移、加权分配、智能选择五种策略
- 🤖 **Auto 智能策略** - 先探索样本不足的候选，再优先选择窗口内成功率更高的渠道
- 🧠 **AI 路由、自动分组与条件分组** - 支持在路由页生成整张路由表，在分组编辑弹窗中补全单个分组，并用 JSON 条件控制分组命中
- 🔄 **协议互转** - 支持 OpenAI Chat / OpenAI Responses / OpenAI Embeddings / Anthropic 四种 API 格式互相转换
- 🌐 **多供应商支持** - 内置支持 OpenAI 兼容、Anthropic、Gemini、Volcengine 渠道
- 🛰️ **媒体与工具类中继** - 支持通过同一套分组 / 重试 / 熔断基础设施转发 OpenAI Images、音频、视频、搜索、重排和审核类端点
- 🧾 **API Key 治理** - 支持模型白名单、过期时间、费用上限、RPM / TPM 限额，以及可选的按模型配额
- 🔐 **角色化管理权限** - 内置 `admin`、`editor`、`viewer` 三种角色，并由服务端强制执行权限控制
- 🚨 **Webhook 告警** - 支持错误率、费用阈值、额度超限、渠道下线等告警规则，并记录通知历史
- 💎 **模型广场与价格能力** - 模型页已升级为 `Model Market` 视图，同时保留创建 / 编辑 / 删除 / 刷新价格能力，并展示覆盖渠道、可用 Key、延迟和成功率
- 🔃 **模型同步** - 自动与渠道同步可用模型列表，省心省力
- 📊 **Analytics 与 Evaluation** - 提供概览、供应商 / 模型 / API Key 利用率、路由健康、语义缓存评估，以及分组测试 / AI 路由入口
- 🛠️ **Ops 与审计** - 提供缓存、配额、健康、系统、审计面板，以及管理面写操作审计链路
- 🧠 **语义缓存** - 为非流式 OpenAI Chat / OpenAI Responses 文本请求提供基于 embedding 的语义缓存，并暴露运行态和成效指标
- 🧭 **页面顺序可配置** - 支持在设置页拖拽调整一级导航顺序，并持久化到服务端设置中
- 💾 **运行时状态持久化** - Auto 策略窗口和熔断器状态会持久化到数据库
- 🎨 **优雅界面** - 简洁美观的 Web 管理面板
- 🗄️ **多数据库支持** - 支持 SQLite、MySQL、PostgreSQL


## 🚀 快速开始

### 🐳 Docker 运行

直接运行

```bash
docker run -d --name octopus \
  --restart unless-stopped \
  -p 8080:8080 \
  -v octopus-data:/app/data \
  -e OCTOPUS_AUTH_JWT_SECRET="replace-with-a-long-random-secret" \
  lingyuins/octopus:latest
```

Windows Docker Desktop 推荐直接使用：

```powershell
docker run -d --name octopus `
  --restart unless-stopped `
  -p 8080:8080 `
  -v octopus-data:/app/data `
  -e OCTOPUS_AUTH_JWT_SECRET="replace-with-a-long-random-secret" `
  lingyuins/octopus:latest
```

或者使用 docker compose 运行

```yaml
services:
  octopus:
    image: lingyuins/octopus:latest
    container_name: octopus
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      OCTOPUS_AUTH_JWT_SECRET: "change-this-to-a-long-random-secret"
```

然后执行：

```bash
docker compose up -d
```

注意：官方镜像默认以非 root 用户 `octopus` 运行，UID/GID 为 `1000`。上面的 `docker run` 默认使用 Docker named volume，这样能避开大多数宿主机目录权限问题，尤其是 Windows Docker Desktop。如果把宿主机目录绑定挂载到 `/app/data`，这个目录必须对 UID/GID `1000` 可写，否则启动时创建 `config.json` 或 `data.db` 会报 `permission denied`。

官方 Docker 镜像会在构建阶段重新编译前端，并把最新导出的管理界面嵌入 Go 二进制，因此容器内前端和对应发布版本保持一致。

如果是从旧版前端升级，升级后浏览器若仍出现旧页面脚本报错，请清理一次站点数据 / Service Worker 缓存，确保加载到最新嵌入式前端资源。


### 📦 从 Release 下载

从 [Releases](https://github.com/lingyuins/octopus/releases) 下载对应平台的二进制文件，然后运行：

```bash
./octopus start
```

### 🛠️ 源码运行

**环境要求：**
- Go 1.24.4
- Node.js 20+
- pnpm

```bash
# 克隆项目
git clone https://github.com/lingyuins/octopus.git
cd octopus
# 可选：通过环境变量预置初始管理员账户
export OCTOPUS_INITIAL_ADMIN_USERNAME="admin"
export OCTOPUS_INITIAL_ADMIN_PASSWORD="change-this-password-long"
# 可选但强烈建议：设置持久化 JWT 密钥
export OCTOPUS_AUTH_JWT_SECRET="replace-with-a-long-random-secret"
# 直接启动后端服务（即使还没构建前端，也可以先以 API-only 模式启动）
go run main.go start
```

如果 `static/out/` 中已经有前端构建产物，Go 二进制会直接提供管理界面；如果还没有构建产物，Octopus 仍然可以正常启动并提供 API，但必须先构建前端并在执行 `go build` / `go run` 前将导出的资源放到 `static/out/` 下，管理界面才能访问。

**构建嵌入式管理界面资源**

```bash
cd web && pnpm install && NEXT_PUBLIC_APP_VERSION="$(git describe --tags --always 2>/dev/null || printf 'dev')" pnpm run build && cd ..
# 将前端构建产物移动到 Go 二进制预期的嵌入目录
mkdir -p static/out
mv web/out/* static/out/
# 如果 Next.js 导出了空的 _not-found 目录，请在构建 Go 前补一个占位文件
printf 'placeholder for go:embed\n' > static/out/_not-found/.keep
# 重新启动后端，此时可直接访问嵌入式管理界面
go run main.go start
```

**开发模式**

```bash
cd web && pnpm install && NEXT_PUBLIC_API_BASE_URL="http://127.0.0.1:8080" NEXT_PUBLIC_APP_VERSION="$(git describe --tags --always 2>/dev/null || printf 'dev')" pnpm run dev
## 新建终端，可选：通过环境变量自动创建初始管理员账户
export OCTOPUS_INITIAL_ADMIN_USERNAME="admin"
export OCTOPUS_INITIAL_ADMIN_PASSWORD="change-this-password-long"
## 可选但强烈建议：设置持久化 JWT 密钥
export OCTOPUS_AUTH_JWT_SECRET="replace-with-a-long-random-secret"
## 启动后端服务
go run main.go start
## 访问前端地址
http://localhost:3000
```

### 🔐 初始管理员设置

首次启动时，可以通过以下任一方式完成管理员初始化：

- 设置 `OCTOPUS_INITIAL_ADMIN_USERNAME` 和 `OCTOPUS_INITIAL_ADMIN_PASSWORD`，在启动时自动创建初始管理员账户
- 或在首次访问 Web UI 时，在引导页面中手动创建初始管理员账户

> ⚠️ **安全提示**：初始管理员密码长度必须至少为 12 个字符。
>
> ⚠️ **安全提示**：如果未配置 `OCTOPUS_AUTH_JWT_SECRET` 或 `auth.jwt_secret`，Octopus 会在启动时生成仅当前进程有效的 JWT 密钥。服务重启后，已有登录 token 会失效。

### 👥 管理员角色

管理 API 和内嵌 Web 管理界面内置三种角色：

- `admin`：完整权限，包括用户管理
- `editor`：可写的运维权限，包括渠道、分组、设置、API Key、日志、告警和 AI 路由
- `viewer`：只读权限

权限校验由服务端执行，服务端会按当前存储的角色判权，而不是只信任 JWT 中的角色声明。

### 📝 配置文件

配置文件默认位于 `data/config.json`，首次启动时自动生成。

**完整配置示例：**

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080
  },
  "database": {
    "type": "sqlite",
    "path": "data/data.db"
  },
  "log": {
    "level": "info"
  },
  "auth": {
    "jwt_secret": "replace-with-a-long-random-secret"
  }
}
```

大多数运行时调优项不存放在 `config.json` 中。重试策略、熔断阈值、Auto 策略调优、日志保留周期、对外 API 基础地址、AI 路由服务配置、语义缓存开关等，都通过设置页 / 管理 API 动态写入数据库。

**配置项说明：**

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.host` | 监听地址 | `0.0.0.0` |
| `server.port` | 服务端口 | `8080` |
| `database.type` | 数据库类型 | `sqlite` |
| `database.path` | 数据库连接地址 | `data/data.db` |
| `log.level` | 日志级别 | `info` |
| `auth.jwt_secret` | JWT 签名密钥 | 空（未设置时启动生成临时密钥） |

> 💡 **提示**：在生产环境运行 Octopus 前，请设置 `OCTOPUS_AUTH_JWT_SECRET` 或 `auth.jwt_secret`，这样登录 token 才能在服务重启后继续有效。

**数据库配置：**

支持三种数据库：

| 类型 | `database.type` | `database.path` 格式 |
|------|-----------------|---------------------|
| SQLite | `sqlite` | `data/data.db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/dbname` |
| PostgreSQL | `postgres` | `postgresql://user:password@host:port/dbname?sslmode=disable` |

**MySQL 配置示例：**

```json
{
  "database": {
    "type": "mysql",
    "path": "root:password@tcp(127.0.0.1:3306)/octopus"
  }
}
```

**PostgreSQL 配置示例：**

```json
{
  "database": {
    "type": "postgres",
    "path": "postgresql://user:password@localhost:5432/octopus?sslmode=disable"
  }
}
```

> 💡 **提示**：MySQL 和 PostgreSQL 需要先手动创建数据库，程序会自动创建表结构。

**环境变量：**

所有配置项均可通过环境变量覆盖，格式为 `OCTOPUS_` + 配置路径（用 `_` 连接）：

| 环境变量 | 对应配置项 |
|----------|-----------|
| `OCTOPUS_SERVER_PORT` | `server.port` |
| `OCTOPUS_SERVER_HOST` | `server.host` |
| `OCTOPUS_DATABASE_TYPE` | `database.type` |
| `OCTOPUS_DATABASE_PATH` | `database.path` |
| `OCTOPUS_DATA_DIR` | 在未显式设置 `database.path` 时，`config.json` 和 SQLite 数据库的默认目录 |
| `OCTOPUS_LOG_LEVEL` | `log.level` |
| `OCTOPUS_AUTH_JWT_SECRET` | `auth.jwt_secret` |
| `OCTOPUS_INITIAL_ADMIN_USERNAME` | 启动时自动创建初始管理员用户名 |
| `OCTOPUS_INITIAL_ADMIN_PASSWORD` | 启动时自动创建初始管理员密码 |
| `OCTOPUS_GITHUB_PAT` | 用于获取最新版本时的速率限制(可选) |
| `OCTOPUS_RELAY_MAX_SSE_EVENT_SIZE` | 最大 SSE 事件大小(可选) |


## 📸 界面预览

> 说明：下方截图主要展示核心管理界面。当前版本仍沿用同一套 UI 风格与导航体系，其中 `Model` 已升级为 `Model Market`，侧边栏也新增了 `Analytics` 与 `Ops`。

### 🖥️ 桌面端

<div align="center">
<table>
<tr>
<td align="center"><b>首页</b></td>
<td align="center"><b>渠道</b></td>
<td align="center"><b>分组</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-home.png" alt="首页" width="400"></td>
<td><img src="web/public/screenshot/desktop-channel.png" alt="渠道" width="400"></td>
<td><img src="web/public/screenshot/desktop-group.png" alt="分组" width="400"></td>
</tr>
<tr>
<td align="center"><b>模型广场</b></td>
<td align="center"><b>日志</b></td>
<td align="center"><b>设置</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-price.png" alt="模型广场" width="400"></td>
<td><img src="web/public/screenshot/desktop-log.png" alt="日志" width="400"></td>
<td><img src="web/public/screenshot/desktop-setting.png" alt="设置" width="400"></td>
</tr>
</table>
</div>

### 📱 移动端

<div align="center">
<table>
<tr>
<td align="center"><b>首页</b></td>
<td align="center"><b>渠道</b></td>
<td align="center"><b>分组</b></td>
<td align="center"><b>模型广场</b></td>
<td align="center"><b>日志</b></td>
<td align="center"><b>设置</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/mobile-home.png" alt="移动端首页" width="140"></td>
<td><img src="web/public/screenshot/mobile-channel.png" alt="移动端渠道" width="140"></td>
<td><img src="web/public/screenshot/mobile-group.png" alt="移动端分组" width="140"></td>
<td><img src="web/public/screenshot/mobile-price.png" alt="移动端模型广场" width="140"></td>
<td><img src="web/public/screenshot/mobile-log.png" alt="移动端日志" width="140"></td>
<td><img src="web/public/screenshot/mobile-setting.png" alt="移动端设置" width="140"></td>
</tr>
</table>
</div>


## 📖 功能说明

### 🧭 管理台模块

当前内嵌管理台包含以下一级模块：

| 模块 | 作用 |
|------|------|
| Home | 版本信息、运行状态和高层摘要 |
| Channel | 上游渠道、Key、Header、同步和延迟探测 |
| Group | 模型路由、负载均衡、会话保持、分组测试和 AI 路由 |
| Model Market | 模型目录、自定义价格、渠道覆盖、可用 Key 数、延迟和成功率摘要 |
| Analytics | 首页承载概览指标后的利用率、路由健康、评估中心 |
| Log | Relay 请求历史、错误详情、Token 使用和费用记录 |
| Alert | 告警规则、通知渠道、状态和历史 |
| Ops | 语义缓存、API Key 配额、系统健康、运行时摘要、审计轨迹 |
| Setting | 版本更新信息、外观与导航偏好、运行时调优、语义缓存、AI 路由服务池、API Key 默认配置、重试、熔断、备份和危险操作 |
| User | 管理员用户和角色管理 |

### 📡 渠道管理

渠道是连接 LLM 供应商的基础配置单元。

**Base URL 说明：**

程序会根据渠道类型自动补全 API 路径，您只需填写基础 URL 即可：

| 渠道类型 | 自动补全路径 | 填写 URL | 完整请求地址示例 |
|----------|-------------|----------|-----------------|
| OpenAI Chat | `/chat/completions` | `https://api.openai.com/v1` | `https://api.openai.com/v1/chat/completions` |
| OpenAI Responses | `/responses` | `https://api.openai.com/v1` | `https://api.openai.com/v1/responses` |
| OpenAI Embeddings | `/embeddings` | `https://api.openai.com/v1` | `https://api.openai.com/v1/embeddings` |
| OpenAI Images | `/images/generations`、`/images/edits`、`/images/variations` | `https://api.openai.com/v1` | `https://api.openai.com/v1/images/generations` |
| Anthropic | `/messages` | `https://api.anthropic.com/v1` | `https://api.anthropic.com/v1/messages` |
| Gemini | `/models/:model:generateContent` | `https://generativelanguage.googleapis.com/v1beta` | `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent` |
| Volcengine | `/responses` | `https://ark.cn-beijing.volces.com/api/v3` | `https://ark.cn-beijing.volces.com/api/v3/responses` |

> 💡 **提示**：填写 Base URL 时无需包含具体的 API 端点路径，程序会自动处理。

### 🌐 公共 Relay 端点

公共 relay API 同时支持 OpenAI 风格和 Anthropic 风格客户端：

- OpenAI 风格客户端：`Authorization: Bearer sk-octopus-...`
- Anthropic 风格客户端：`x-api-key: sk-octopus-...`

| 类别 | 路径 | 说明 |
|------|------|------|
| OpenAI 兼容 LLM | `/v1/chat/completions`、`/v1/responses`、`/v1/embeddings`、`/v1/models` | JSON 请求 / 响应 |
| Anthropic 兼容 LLM | `/v1/messages` | Anthropic 风格请求 / 响应 |
| JSON 媒体 / 工具类 | `/v1/images/generations`、`/v1/audio/speech`、`/v1/videos/generations`、`/v1/music/generations`、`/v1/search`、`/v1/rerank`、`/v1/moderations` | 复用同一套分组 / 重试 / 熔断逻辑 |
| Multipart 媒体类 | `/v1/images/edits`、`/v1/images/variations`、`/v1/audio/transcriptions` | 透传 multipart 上传 |

当上游支持 `stream=true` 时，JSON 媒体类端点也可以直接透传 SSE 流。

语义缓存当前只会评估非流式的 OpenAI Chat 与 OpenAI Responses 文本请求。Anthropic、embeddings、流式请求以及媒体 / 工具类端点都会直接旁路缓存，继续走正常 relay 链路。

---

### 📁 分组管理

分组用于将多个渠道聚合为一个统一的对外模型名称。

**核心概念：**

- **分组名称** 即程序对外暴露的模型名称
- 调用 API 时，将请求中的 `model` 参数设置为分组名称即可
- **首字超时**：单位秒，仅对流式响应生效，`0` 表示不限制
- **会话保持**：单位秒，在设定时间窗口内，同一 API Key + 模型会优先复用上次成功的渠道，`0` 表示禁用
- **Condition (JSON)**：可选的 AND 条件规则，当前只在主 LLM relay 路径里生效；内置请求上下文目前包含 `model`、`api_key_id`、`hour`

**负载均衡模式：**

| 模式 | 说明 |
|------|------|
| 🔄 **轮询** | 每次请求依次切换到下一个渠道 |
| 🎲 **随机** | 每次请求随机选择一个可用渠道 |
| 🛡️ **故障转移** | 优先使用高优先级渠道，仅当其故障时才切换到低优先级渠道 |
| ⚖️ **加权分配** | 按权重从高到低排序后依次尝试渠道 |
| 🤖 **智能选择** | 优先探索样本不足的候选，样本充足后按时间窗口内成功率优选 |

**Auto 智能策略默认值：**

- **最小样本数**：`10`
- **时间窗口**：`300` 秒
- **滑动窗口大小**：每个渠道-模型对保留 `100` 条记录
- **延迟权重**：`30`
- 当候选未达到最小样本数时，系统优先进行探索
- 候选都完成探索后，系统按成功率排序；成功率相同时，再按样本量、权重、优先级和延迟调优兜底
- Auto 策略窗口会在启动时从数据库恢复，并在定时任务和优雅退出时持久化

**AI 路由行为：**

- 在路由页点击 **AI路由** 时，系统会把全部模型发送给 AI，批量生成整张路由表
- 遇到同名已有分组时，只会追加缺失的路由项，不会清空或替换已有分组
- 在分组编辑弹窗点击 **AI补全当前分组** 时，系统会把全部模型发送给 AI，并只向当前分组追加匹配到的路由项
- 原先的 “AI路由目标分组” 设置现在只作为单分组兼容模式下的默认目标分组使用

> 💡 **示例**：创建分组名称为 `gpt-4o`，将多个供应商的 GPT-4o 渠道加入该分组，即可通过统一的 `model: gpt-4o` 访问所有渠道。

---

### 💎 模型广场与价格

`Model` 路由现在已经从单纯的价格列表升级为模型广场视图。它会在一个页面里同时展示模型价格、渠道覆盖、可用 Key 数、平均延迟和成功 / 失败统计，但原本的价格管理能力仍然保留。

**每张卡片整合的数据：**

- LLM 价格目录中的自定义价格或同步价格
- 渠道与模型关系中的覆盖渠道数、可用 Key 数
- 模型统计中的平均延迟、成功次数、失败次数

**顶部摘要指标：**

| 指标 | 含义 |
|------|------|
| Models | 当前筛选结果里的模型卡片数 |
| Coverage | 当前结果集中渠道对模型的覆盖总数 |
| Unique Channels | 当前结果集中涉及的去重渠道数 |
| Average Latency | 按请求统计加权后的平均延迟 |

**数据来源：**

- 系统会定期从 [models.dev](https://github.com/sst/models.dev) 同步更新模型价格数据
- 当创建渠道或同步渠道模型时，如果某个模型还不在本地目录里，Octopus 会自动创建本地价格记录，确保后续仍可手动维护价格
- 也支持手动创建 models.dev 中已存在的模型，用于自定义价格

**价格优先级：**

| 优先级 | 来源 | 说明 |
|:------:|------|------|
| 🥇 高 | 本页面 | 用户在模型广场页面设置的价格 |
| 🥈 低 | models.dev | 自动同步的默认价格 |

> 💡 **提示**：如需覆盖某个模型的默认价格，只需在模型广场页面为其设置自定义价格即可。

**页面仍保留的操作：**

- 创建自定义模型价格
- 编辑已有模型的输入 / 输出 / 缓存价格
- 删除自定义模型条目
- 在页面头部手动刷新上游价格
- 在设置页 `LLM Price` 卡片里继续维护定时刷新策略

---

### 📈 Analytics

Analytics 是偏只读的分析模块，当前包含 3 个页签：

| 页签 | 展示内容 |
|------|----------|
| Utilization | 按供应商、模型、API Key 的利用率拆分 |
| Route Health | 每个分组的健康分、启用 / 禁用项数量、近期失败压力 |
| Evaluation | 分组可用性、AI 路由进度、分组测试进度、语义缓存成效 |

**时间范围：** `1d`、`7d`、`30d`、`90d`、`ytd`、`all`

`/api/v1/analytics/overview` 这个概览接口仍然保留，但当前 UI 中这些摘要指标的主要入口已经迁移到首页。首页现在额外承载独立的 `7d / 30d / 90d` 概览范围切换，以及 Hero 摘要、趋势图、活跃热力图和排行榜。

`Evaluation` 不会复制完整的分组页和设置页，而是作为轻量入口，把分组测试、AI 路由和语义缓存评估串起来。

---

### 🛠️ Ops

Ops 模块面向运行态诊断和运维视角，当前包含：

| 页签 | 展示内容 |
|------|----------|
| Cache | 语义缓存的配置开关、运行态是否生效、TTL、阈值、命中 / 未命中、占用率 |
| Quota | API Key 在 RPM、TPM、费用上限、按模型配额上的整体姿态 |
| Health | 数据库连通性、缓存就绪状态、任务运行状态、近期错误量、异常分组 |
| System | 构建信息、数据库类型、Public API Base URL、代理、保留周期、AI 路由模式和服务列表 |
| Audit | 管理面写操作的分页审计日志 |

**审计范围：**

- 覆盖已纳入白名单的管理面写接口，例如 channel / group / model / setting / API key / alert / user 变更、AI 路由生成、日志清理、价格刷新、导入、自更新等
- 不记录公共 `/v1/...` relay 流量

---

### ⚙️ 设置

系统全局配置项。

**统计保存周期（分钟）：**

由于程序涉及大量统计项目，若每次请求都直接写入数据库会影响读写性能。因此程序采用以下策略：

- 统计数据先保存在 **内存** 中
- 按设定的周期 **定期批量写入** 数据库
- 中继负载均衡运行时状态也采用同样的周期持久化方式

**运行时状态持久化：**

- Auto 策略窗口会在启动时从数据库恢复
- 熔断器状态会在启动时从数据库恢复
- 二者都会按统计保存周期定时落库
- 二者也会在优雅退出时主动保存

**当前设置页的重点卡片：**

| 卡片 | 作用 |
|------|------|
| Info | 当前版本、最新发布检查、前后端版本不一致检测、站内自更新入口 |
| Appearance | 主题、语言、告警通知语言，以及一级导航顺序拖拽偏好 |
| System | Public API Base URL、代理、CORS 白名单和统计落库周期 |
| Account | 当前控制台暴露的账户 / 会话相关偏好 |
| Semantic Cache | 开关、TTL、相似度阈值、最大条目数、embedding Base URL / API Key / 模型 / 超时 |
| AI Route | 单分组兼容默认目标、超时、并发度、服务池配置 |
| API Key | API Key 创建默认值和配额相关控制 |
| Retry / Auto Strategy / Circuit Breaker | Relay 重试和候选优选调优 |
| Log / LLM Price / LLM Sync | 日志保留、价格刷新节奏、上游模型同步 |
| Backup | 数据库导出与导入 |
| Route Group Danger | 二次确认后删除全部路由分组 |

**语义缓存的真实生效条件：**

- 只作用于非流式 OpenAI Chat / OpenAI Responses 文本请求
- 缓存命名空间按 `api_key_id + endpoint_family + requested_model` 隔离
- 即使开关打开，只要 embedding 客户端没有配完整，或查询 / 写入 embedding 失败，也会自动旁路，不阻断正常转发
- 运行态与成效可同时在 `Analytics -> Evaluation` 和 `Ops -> Cache` 中查看

**设置页危险操作：**

- 设置页新增了 **删除全部路由分组**
- 执行前要求二次确认
- 操作会删除全部分组和分组项，并把单分组 AI 路由默认目标分组重置为 `0`，避免残留悬挂引用

> ⚠️ **重要提示**：退出程序时，请使用正常的关闭方式（如 `Ctrl+C` 或发送 `SIGTERM` 信号），以确保内存中的统计数据能正确写入数据库。**请勿使用 `kill -9` 等强制终止方式**，否则可能导致统计数据丢失。




## 🔌 客户端接入

### OpenAI SDK

```python
from openai import OpenAI
import os

client = OpenAI(   
    base_url="http://127.0.0.1:8080/v1",   
    api_key="sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg", 
)
completion = client.chat.completions.create(
    model="octopus-openai",  # 填写正确的分组名称
    messages = [
        {"role": "user", "content": "Hello"},
    ],
)
print(completion.choices[0].message.content)
```

### Claude Code

编辑 `~/.claude/settings.json`

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8080",
    "ANTHROPIC_AUTH_TOKEN": "sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg",
    "API_TIMEOUT_MS": "3000000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "ANTHROPIC_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_SMALL_FAST_MODEL": "octopus-haiku-4-5",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "octopus-haiku-4-5"
  }
}
```

### Codex

编辑 `~/.codex/config.toml`

```toml
model = "octopus-codex" # 填写正确的分组名称

model_provider = "octopus"

[model_providers.octopus]
name = "octopus"
base_url = "http://127.0.0.1:8080/v1"
```
编辑 `~/.codex/auth.json`

```json
{
  "OPENAI_API_KEY": "sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg"
}
```


---

## 🤝 致谢

- 🙏 [looplj/axonhub](https://github.com/looplj/axonhub) - 本项目的 LLM API 适配模块直接源自该仓库的实现
- 📊 [sst/models.dev](https://github.com/sst/models.dev) - AI 模型数据库，提供模型价格数据
