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
- 🧠 **AI 路由与自动分组** - 支持在路由页生成整张路由表，或在分组编辑弹窗中补全单个分组
- 🔄 **协议互转** - 支持 OpenAI Chat / OpenAI Responses / OpenAI Embeddings / Anthropic 四种 API 格式互相转换
- 🌐 **多供应商支持** - 内置支持 OpenAI 兼容、Anthropic、Gemini、Volcengine 渠道
- 💰 **价格同步** - 自动更新模型价格
- 🔃 **模型同步** - 自动与渠道同步可用模型列表，省心省力
- 📊 **数据统计** - 全面的请求统计、Token 消耗、费用追踪和中继日志
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
| `OCTOPUS_GITHUB_PAT` | 用于获取最新版本时的速率限制(可选) |
| `OCTOPUS_RELAY_MAX_SSE_EVENT_SIZE` | 最大 SSE 事件大小(可选) |


## 📸 界面预览

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
<td align="center"><b>价格</b></td>
<td align="center"><b>日志</b></td>
<td align="center"><b>设置</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-price.png" alt="价格" width="400"></td>
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
<td align="center"><b>价格</b></td>
<td align="center"><b>日志</b></td>
<td align="center"><b>设置</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/mobile-home.png" alt="移动端首页" width="140"></td>
<td><img src="web/public/screenshot/mobile-channel.png" alt="移动端渠道" width="140"></td>
<td><img src="web/public/screenshot/mobile-group.png" alt="移动端分组" width="140"></td>
<td><img src="web/public/screenshot/mobile-price.png" alt="移动端价格" width="140"></td>
<td><img src="web/public/screenshot/mobile-log.png" alt="移动端日志" width="140"></td>
<td><img src="web/public/screenshot/mobile-setting.png" alt="移动端设置" width="140"></td>
</tr>
</table>
</div>


## 📖 功能说明

### 📡 渠道管理

渠道是连接 LLM 供应商的基础配置单元。

**Base URL 说明：**

程序会根据渠道类型自动补全 API 路径，您只需填写基础 URL 即可：

| 渠道类型 | 自动补全路径 | 填写 URL | 完整请求地址示例 |
|----------|-------------|----------|-----------------|
| OpenAI Chat | `/chat/completions` | `https://api.openai.com/v1` | `https://api.openai.com/v1/chat/completions` |
| OpenAI Responses | `/responses` | `https://api.openai.com/v1` | `https://api.openai.com/v1/responses` |
| OpenAI Embeddings | `/embeddings` | `https://api.openai.com/v1` | `https://api.openai.com/v1/embeddings` |
| Anthropic | `/messages` | `https://api.anthropic.com/v1` | `https://api.anthropic.com/v1/messages` |
| Gemini | `/models/:model:generateContent` | `https://generativelanguage.googleapis.com/v1beta` | `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent` |
| Volcengine | `/responses` | `https://ark.cn-beijing.volces.com/api/v3` | `https://ark.cn-beijing.volces.com/api/v3/responses` |

> 💡 **提示**：填写 Base URL 时无需包含具体的 API 端点路径，程序会自动处理。

---

### 📁 分组管理

分组用于将多个渠道聚合为一个统一的对外模型名称。

**核心概念：**

- **分组名称** 即程序对外暴露的模型名称
- 调用 API 时，将请求中的 `model` 参数设置为分组名称即可
- **首字超时**：单位秒，仅对流式响应生效，`0` 表示不限制
- **会话保持**：单位秒，在设定时间窗口内，同一 API Key + 模型会优先复用上次成功的渠道，`0` 表示禁用

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
- 当候选未达到最小样本数时，系统优先进行探索
- 候选都完成探索后，系统按成功率排序；成功率相同时，再按样本量、权重、优先级兜底
- Auto 策略窗口会在启动时从数据库恢复，并在定时任务和优雅退出时持久化

**AI 路由行为：**

- 在路由页点击 **AI路由** 时，系统会把全部模型发送给 AI，批量生成整张路由表
- 遇到同名已有分组时，只会追加缺失的路由项，不会清空或替换已有分组
- 在分组编辑弹窗点击 **AI补全当前分组** 时，系统会把全部模型发送给 AI，并只向当前分组追加匹配到的路由项
- 原先的 “AI路由目标分组” 设置现在只作为单分组兼容模式下的默认目标分组使用

> 💡 **示例**：创建分组名称为 `gpt-4o`，将多个供应商的 GPT-4o 渠道加入该分组，即可通过统一的 `model: gpt-4o` 访问所有渠道。

---

### 💰 价格管理

管理系统中的模型价格信息。

**数据来源：**

- 系统会定期从 [models.dev](https://github.com/sst/models.dev) 同步更新模型价格数据
- 当创建渠道时，若渠道包含的模型不在 models.dev 中，系统会自动在此页面创建该模型的价格信息,所以此页面显示的是没有从上游获取到价格的模型，用户可以手动设置价格
- 也支持手动创建 models.dev 中已存在的模型，用于自定义价格

**价格优先级：**

| 优先级 | 来源 | 说明 |
|:------:|------|------|
| 🥇 高 | 本页面 | 用户在价格管理页面设置的价格 |
| 🥈 低 | models.dev | 自动同步的默认价格 |

> 💡 **提示**：如需覆盖某个模型的默认价格，只需在价格管理页面为其设置自定义价格即可。

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
