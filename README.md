<div align="center">

<img src="web/public/logo.svg" alt="Octopus Logo" width="120" height="120">

### Octopus

**A Simple, Beautiful, and Elegant LLM API Aggregation & Load Balancing Service for Individuals**

 English | [简体中文](README_zh.md)

</div>


## ✨ Features

- 🔀 **Multi-Channel Aggregation** - Connect multiple LLM provider channels with unified management
- 🔑 **Multi-Key Support** - Support multiple API keys for a single channel
- ⚡ **Smart Selection** - Multiple endpoints per channel, smart selection of the endpoint with the shortest delay
- ⚖️ **Load Balancing** - Support round robin, random, failover, weighted, and auto strategies
- 🤖 **Auto Strategy** - Explore candidates first, then prefer higher in-window success rate automatically
- 🧠 **AI Routing, Auto Grouping & Conditional Groups** - Generate the full routing table from the route page, fill a single group from the edit dialog, and gate groups with JSON conditions
- 🔄 **Protocol Conversion** - Seamless conversion between OpenAI Chat / OpenAI Responses / OpenAI Embeddings / Anthropic API formats
- 🌐 **Multi-Provider Support** - Built-in support for OpenAI-compatible, Anthropic, Gemini, and Volcengine channels
- 🛰️ **Media & Utility Relay** - Relay OpenAI Images, audio, video, search, rerank, and moderation endpoints through the same group / retry / circuit-breaker infrastructure
- 🧾 **API Key Governance** - Supported-model allowlists, expiry, max-cost caps, RPM / TPM limits, and optional per-model quotas
- 🔐 **Role-Based Admin Access** - Built-in `admin`, `editor`, and `viewer` roles with server-side permission enforcement
- 🚨 **Webhook Alerts** - Alert rules for error rate, cost threshold, quota exceeded, and channel down with webhook notifications and history
- 💎 **Model Market & Price Sync** - The model page now surfaces pricing, channel coverage, enabled key counts, latency, and success metrics while preserving create / edit / delete / refresh price workflows
- 🔃 **Model Sync** - Automatic synchronization of available model lists with channels
- 📊 **Analytics & Evaluation** - Overview, provider / model / API key utilization, route health, semantic-cache evaluation, and live entry points for group testing / AI routing
- 🛠️ **Ops & Audit** - Cache, quota, health, system, and audit dashboards for daily operations, plus a management-write audit trail
- 🧠 **Semantic Cache** - Embedding-backed semantic cache for non-streaming OpenAI Chat / OpenAI Responses text requests, with runtime status and effectiveness metrics
- 🧭 **Configurable Navigation Order** - Persist top-level console page order in settings and reuse it across browsers
- 💾 **Runtime State Persistence** - Persist auto strategy windows and circuit breaker state to the database
- 🎨 **Elegant UI** - Clean and beautiful web management panel
- 🗄️ **Multi-Database Support** - Support for SQLite, MySQL, PostgreSQL


## 🚀 Quick Start

### 🐳 Docker

Run directly:

```bash
docker run -d --name octopus \
  --restart unless-stopped \
  -p 8080:8080 \
  -v octopus-data:/app/data \
  -e OCTOPUS_AUTH_JWT_SECRET="replace-with-a-long-random-secret" \
  lingyuins/octopus:latest
```

Recommended on Windows Docker Desktop:

```powershell
docker run -d --name octopus `
  --restart unless-stopped `
  -p 8080:8080 `
  -v octopus-data:/app/data `
  -e OCTOPUS_AUTH_JWT_SECRET="replace-with-a-long-random-secret" `
  lingyuins/octopus:latest
```

Or use docker compose:

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
      OCTOPUS_AUTH_JWT_SECRET: "replace-with-a-long-random-secret"
```

Then run:

```bash
docker compose up -d
```

Note: The official image runs as the non-root user `octopus` with UID/GID `1000`. The `docker run` example above uses a named volume because it avoids most host-permission issues, especially on Windows Docker Desktop. If you bind-mount a host directory to `/app/data`, make sure that directory is writable by UID/GID `1000`, otherwise startup will fail with `permission denied` when creating `config.json` or `data.db`.

The official Docker image rebuilds the frontend during image build and embeds the latest exported UI into the Go binary, so the container includes the matching management UI for that release.

If you are upgrading from an older web build and still see stale frontend errors in the browser, clear the site data / service worker cache once after upgrading so the latest embedded assets are loaded.


### 📦 Download from Release

Download the binary for your platform from [Releases](https://github.com/lingyuins/octopus/releases), then run:

```bash
./octopus start
```

### 🛠️ Build from Source

**Requirements:**
- Go 1.24.4
- Node.js 20+
- pnpm

```bash
# Clone the repository
git clone https://github.com/lingyuins/octopus.git
cd octopus
# Optional: bootstrap the initial admin via environment variables
export OCTOPUS_INITIAL_ADMIN_USERNAME="admin"
export OCTOPUS_INITIAL_ADMIN_PASSWORD="change-this-password-long"
# Optional but recommended: set a persistent JWT secret
export OCTOPUS_AUTH_JWT_SECRET="replace-with-a-long-random-secret"
# Start the backend service directly (API-only mode works even before frontend assets are built)
go run main.go start
```

If `static/out/` already contains built frontend assets, the Go binary serves the management UI directly. Otherwise, Octopus still starts normally and exposes the API endpoints, but the management UI is unavailable until you build the frontend and place the exported assets under `static/out/` before running `go build` / `go run`.

**Build frontend assets for the embedded management UI**

```bash
cd web && pnpm install && NEXT_PUBLIC_APP_VERSION="$(git describe --tags --always 2>/dev/null || printf 'dev')" pnpm run build && cd ..
# Move frontend assets to the embed directory expected by the Go binary
mkdir -p static/out
mv web/out/* static/out/
# If Next.js exports an empty _not-found directory, add a placeholder before building Go
printf 'placeholder for go:embed\n' > static/out/_not-found/.keep
# Start the backend service with embedded UI assets available in the repository
go run main.go start
```

**Development Mode**

```bash
cd web && pnpm install && NEXT_PUBLIC_API_BASE_URL="http://127.0.0.1:8080" NEXT_PUBLIC_APP_VERSION="$(git describe --tags --always 2>/dev/null || printf 'dev')" pnpm run dev
## Open a new terminal, optionally set initial admin credentials for automatic bootstrap
export OCTOPUS_INITIAL_ADMIN_USERNAME="admin"
export OCTOPUS_INITIAL_ADMIN_PASSWORD="change-this-password-long"
## Optional but recommended: set a persistent JWT secret
export OCTOPUS_AUTH_JWT_SECRET="replace-with-a-long-random-secret"
## Start the backend service
go run main.go start
## Access the frontend at
http://localhost:3000
```

### 🔐 Initial Admin Setup

On first launch, you can initialize the admin account in either of these ways:

- Provide `OCTOPUS_INITIAL_ADMIN_USERNAME` and `OCTOPUS_INITIAL_ADMIN_PASSWORD` to bootstrap automatically at startup
- Or open the Web UI on first visit and create the initial admin account there

> ⚠️ **Security Notice**: The initial admin password must be at least 12 characters long.
>
> ⚠️ **Security Notice**: If `OCTOPUS_AUTH_JWT_SECRET` or `auth.jwt_secret` is not configured, Octopus will generate an in-memory JWT secret at startup. Existing login tokens will become invalid after a restart.

### 👥 Admin Roles

The management API and embedded Web UI use three built-in roles:

- `admin`: full access, including user management
- `editor`: operational write access for channels, groups, settings, API keys, logs, alerts, and AI routing
- `viewer`: read-only access to operational data

Role checks are enforced on the server side, using the currently stored role rather than trusting only the JWT claim.

### 📝 Configuration File

The configuration file is located at `data/config.json` by default and is automatically generated on first startup.

**Complete Configuration Example:**

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

Most operational knobs are not stored in `config.json`. Retry policy, circuit breaker thresholds, auto-strategy tuning, relay log retention, public API base URL, AI-route service settings, and semantic-cache switches are managed at runtime from the Settings page / management API and stored in the database.

**Configuration Options:**

| Option | Description | Default |
|--------|-------------|---------|
| `server.host` | Listen address | `0.0.0.0` |
| `server.port` | Server port | `8080` |
| `database.type` | Database type | `sqlite` |
| `database.path` | Database connection string | `data/data.db` |
| `log.level` | Log level | `info` |
| `auth.jwt_secret` | JWT signing secret | empty (ephemeral secret generated at startup if unset) |

> 💡 **Tip**: Set `OCTOPUS_AUTH_JWT_SECRET` or `auth.jwt_secret` before running Octopus in production so login tokens stay valid across restarts.

**Database Configuration:**

Three database types are supported:

| Type | `database.type` | `database.path` Format |
|------|-----------------|-----------------------|
| SQLite | `sqlite` | `data/data.db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/dbname` |
| PostgreSQL | `postgres` | `postgresql://user:password@host:port/dbname?sslmode=disable` |

**MySQL Configuration Example:**

```json
{
  "database": {
    "type": "mysql",
    "path": "root:password@tcp(127.0.0.1:3306)/octopus"
  }
}
```

**PostgreSQL Configuration Example:**

```json
{
  "database": {
    "type": "postgres",
    "path": "postgresql://user:password@localhost:5432/octopus?sslmode=disable"
  }
}
```

> 💡 **Tip**: MySQL and PostgreSQL require manual database creation. The application will automatically create the table structure.

### 🌐 Environment Variables

All configuration options can be overridden via environment variables using the format `OCTOPUS_` + configuration path (joined with `_`):

| Environment Variable | Configuration Option |
|---------------------|---------------------|
| `OCTOPUS_SERVER_PORT` | `server.port` |
| `OCTOPUS_SERVER_HOST` | `server.host` |
| `OCTOPUS_DATABASE_TYPE` | `database.type` |
| `OCTOPUS_DATABASE_PATH` | `database.path` |
| `OCTOPUS_DATA_DIR` | Default directory for `config.json` and the SQLite DB when `database.path` is not explicitly set |
| `OCTOPUS_LOG_LEVEL` | `log.level` |
| `OCTOPUS_AUTH_JWT_SECRET` | `auth.jwt_secret` |
| `OCTOPUS_INITIAL_ADMIN_USERNAME` | Bootstrap the initial admin username at startup |
| `OCTOPUS_INITIAL_ADMIN_PASSWORD` | Bootstrap the initial admin password at startup |
| `OCTOPUS_GITHUB_PAT` | For rate limiting when getting the latest version (optional) |
| `OCTOPUS_RELAY_MAX_SSE_EVENT_SIZE` | Maximum SSE event size (optional) |

## 📸 Screenshots

> Note: The screenshots below show the core console surfaces. Current builds keep the same visual system and navigation, with `Model` presented as `Model Market` and additional `Analytics` / `Ops` entries in the sidebar.

### 🖥️ Desktop

<div align="center">
<table>
<tr>
<td align="center"><b>Dashboard</b></td>
<td align="center"><b>Channel Management</b></td>
<td align="center"><b>Group Management</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-home.png" alt="Dashboard" width="400"></td>
<td><img src="web/public/screenshot/desktop-channel.png" alt="Channel" width="400"></td>
<td><img src="web/public/screenshot/desktop-group.png" alt="Group" width="400"></td>
</tr>
<tr>
<td align="center"><b>Model Market</b></td>
<td align="center"><b>Logs</b></td>
<td align="center"><b>Settings</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-price.png" alt="Model Market" width="400"></td>
<td><img src="web/public/screenshot/desktop-log.png" alt="Logs" width="400"></td>
<td><img src="web/public/screenshot/desktop-setting.png" alt="Settings" width="400"></td>
</tr>
</table>
</div>

### 📱 Mobile

<div align="center">
<table>
<tr>
<td align="center"><b>Home</b></td>
<td align="center"><b>Channel</b></td>
<td align="center"><b>Group</b></td>
<td align="center"><b>Model Market</b></td>
<td align="center"><b>Logs</b></td>
<td align="center"><b>Settings</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/mobile-home.png" alt="Mobile Home" width="140"></td>
<td><img src="web/public/screenshot/mobile-channel.png" alt="Mobile Channel" width="140"></td>
<td><img src="web/public/screenshot/mobile-group.png" alt="Mobile Group" width="140"></td>
<td><img src="web/public/screenshot/mobile-price.png" alt="Mobile Model Market" width="140"></td>
<td><img src="web/public/screenshot/mobile-log.png" alt="Mobile Logs" width="140"></td>
<td><img src="web/public/screenshot/mobile-setting.png" alt="Mobile Settings" width="140"></td>
</tr>
</table>
</div>


## 📖 Documentation

### 🧭 Management Console Modules

The embedded management UI currently ships with these top-level modules:

| Module | What it covers |
|--------|----------------|
| Home | Version, runtime status, and high-level summaries |
| Channel | Upstream provider configuration, keys, headers, sync, and latency probing |
| Group | Model routing, load-balancing strategies, sticky sessions, group test, and AI route generation |
| Model Market | Model catalog, custom pricing, channel coverage, enabled key counts, latency, and success summaries |
| Analytics | Overview, utilization, route health, and evaluation |
| Log | Relay request history, error details, token usage, and cost records |
| Alert | Alert rules, notification channels, state, and history |
| Ops | Semantic cache, API key quota posture, system health, runtime summary, and audit trail |
| Setting | Runtime tuning, semantic cache, page order, AI route services, retry, circuit breaker, backup, and dangerous operations |
| User | Admin user management and roles |

### 📡 Channel Management

Channels are the basic configuration units for connecting to LLM providers.

**Base URL Guide:**

The program automatically appends API paths based on channel type. You only need to provide the base URL:

| Channel Type | Auto-appended Path | Base URL | Full Request URL Example |
|--------------|-------------------|----------|--------------------------|
| OpenAI Chat | `/chat/completions` | `https://api.openai.com/v1` | `https://api.openai.com/v1/chat/completions` |
| OpenAI Responses | `/responses` | `https://api.openai.com/v1` | `https://api.openai.com/v1/responses` |
| OpenAI Embeddings | `/embeddings` | `https://api.openai.com/v1` | `https://api.openai.com/v1/embeddings` |
| OpenAI Images | `/images/generations`, `/images/edits`, `/images/variations` | `https://api.openai.com/v1` | `https://api.openai.com/v1/images/generations` |
| Anthropic | `/messages` | `https://api.anthropic.com/v1` | `https://api.anthropic.com/v1/messages` |
| Gemini | `/models/:model:generateContent` | `https://generativelanguage.googleapis.com/v1beta` | `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent` |
| Volcengine | `/responses` | `https://ark.cn-beijing.volces.com/api/v3` | `https://ark.cn-beijing.volces.com/api/v3/responses` |

> 💡 **Tip**: No need to include specific API endpoint paths in the Base URL - the program handles this automatically.

### 🌐 Public Relay Endpoints

The public relay API supports both OpenAI-style and Anthropic-style clients:

- OpenAI-style clients: `Authorization: Bearer sk-octopus-...`
- Anthropic-style clients: `x-api-key: sk-octopus-...`

| Category | Paths | Notes |
|----------|-------|-------|
| OpenAI-compatible LLM | `/v1/chat/completions`, `/v1/responses`, `/v1/embeddings`, `/v1/models` | JSON request / response |
| Anthropic-compatible LLM | `/v1/messages` | Anthropic-style request / response |
| JSON media / utility | `/v1/images/generations`, `/v1/audio/speech`, `/v1/videos/generations`, `/v1/music/generations`, `/v1/search`, `/v1/rerank`, `/v1/moderations` | Uses the same group / retry / circuit-breaker pipeline |
| Multipart media | `/v1/images/edits`, `/v1/images/variations`, `/v1/audio/transcriptions` | Multipart upload forwarding |

JSON media endpoints can also proxy upstream SSE streams when the provider supports `stream=true`.

Semantic cache is currently evaluated only for non-streaming OpenAI Chat and OpenAI Responses text requests. Anthropic, embeddings, streaming, and media / utility requests bypass the cache and continue through the normal relay flow.

---

### 📁 Group Management

Groups aggregate multiple channels into a unified external model name.

**Core Concepts:**

- **Group name** is the model name exposed by the program
- When calling the API, set the `model` parameter to the group name
- **First Token Timeout**: unit in seconds, only effective for streaming responses, `0` means no limit
- **Session Keep Time**: unit in seconds, keeps using the same channel for the same API key + model within the configured session window, `0` means disabled
- **Condition (JSON)**: optional AND rules currently evaluated in the main LLM relay path; the built-in request context currently includes `model`, `api_key_id`, and `hour`

**Load Balancing Modes:**

| Mode | Description |
|------|-------------|
| 🔄 **Round Robin** | Cycles through channels sequentially for each request |
| 🎲 **Random** | Randomly selects an available channel for each request |
| 🛡️ **Failover** | Prioritizes high-priority channels, switches to lower priority only on failure |
| ⚖️ **Weighted** | Orders candidates by weight from high to low, then tries them in that order |
| 🤖 **Auto** | Explores under-sampled candidates first, then prefers the candidate with the best success rate inside the configured window |

**Auto Strategy Defaults:**

- **Minimum samples**: `10`
- **Time window**: `300` seconds
- **Sliding window size**: `100` records per channel-model pair
- **Latency weight**: `30`
- Before a candidate reaches the minimum sample count, Octopus prioritizes exploration
- After candidates are explored, Octopus sorts by success rate, then uses sample count, weight, priority, and latency tuning as tie-breakers
- Auto-strategy windows are restored from the database at startup and saved periodically plus on graceful shutdown

**AI Routing Behavior:**

- Clicking **AI Route** on the route page sends all models to AI and generates the full routing table in batch
- Existing groups with the same name only receive missing route items; existing groups are not cleared or replaced
- Clicking **AI Fill Current Group** in the edit dialog sends all models to AI and appends only the matched route items to that group
- The setting previously named AI route target group now acts as the default target group for the single-group compatibility flow only

> 💡 **Example**: Create a group named `gpt-4o`, add multiple providers' GPT-4o channels to it, then access all channels via a unified `model: gpt-4o`.

---

### 💎 Model Market & Pricing

The `Model` route is now a model market view instead of a plain price list. It combines model pricing, channel coverage, enabled key counts, average latency, and success metrics into a single page while keeping the original pricing workflows.

**Data merged on each card:**

- Custom or synced pricing from the LLM price catalog
- Channel coverage and enabled key counts from channel-model relationships
- Average latency and success / failure counts from recorded model stats

**Summary metrics:**

| Metric | Meaning |
|--------|---------|
| Models | Number of currently visible model cards |
| Coverage | Total channel-to-model coverage count in the current result set |
| Unique Channels | Distinct channels represented by the visible cards |
| Average Latency | Weighted average latency derived from model request stats |

**Data Sources:**

- The system periodically syncs model pricing data from [models.dev](https://github.com/sst/models.dev)
- When creating or syncing channels, if a model is not yet in the local catalog, Octopus automatically creates a local model-price record so the price can still be maintained manually
- Manual creation of models that exist in models.dev is also supported for custom pricing

**Price Priority:**

| Priority | Source | Description |
|:--------:|--------|-------------|
| 🥇 High | This Page | Prices set by user in the model market page |
| 🥈 Low | models.dev | Auto-synced default prices |

> 💡 **Tip**: To override a model's default price, simply set a custom price for it in the model market page.

**Operational actions preserved on the page:**

- Create a custom model price record
- Edit input / output / cache prices for an existing model
- Delete a custom model entry
- Refresh upstream pricing from the page header
- Keep the scheduled price refresh policy in the Settings `LLM Price` card

---

### 📈 Analytics

The Analytics module is a read-oriented operations view with four tabs:

| Tab | What it shows |
|-----|---------------|
| Overview | Request count, success rate, token volume, cost, provider count, API key count, model count, and fallback rate |
| Utilization | Provider, model, and API key breakdowns for the selected time range |
| Route Health | Health score, enabled / disabled item counts, and recent failure pressure for each group |
| Evaluation | Group readiness, AI route progress, group test progress, and semantic-cache effectiveness |

**Time ranges:** `1d`, `7d`, `30d`, `90d`, `ytd`, and `all`

The Evaluation tab is intentionally lightweight: it acts as an entry point into group testing, AI routing, and semantic-cache tuning instead of duplicating those full workflows.

---

### 🛠️ Ops

The Ops module focuses on runtime posture and operational diagnostics:

| Tab | What it shows |
|-----|---------------|
| Cache | Semantic-cache configured state, runtime-enabled state, TTL, threshold, hit / miss counts, and usage rate |
| Quota | API key limit posture across RPM, TPM, max-cost, and per-model quota settings |
| Health | Database reachability, cache readiness, task-runtime sanity, recent error count, and failing groups |
| System | Build metadata, database type, public API base URL, proxy, retention intervals, AI route mode, and AI route services |
| Audit | Paginated audit history for management-side write operations |

**Audit scope:**

- Covers selected management write routes such as channel / group / model / setting / API key / alert / user mutations, AI route generation, log clearing, price refresh, import, and self-update
- Does not record public `/v1/...` relay traffic

---

### ⚙️ Settings

Global system configuration.

**Statistics Save Interval (minutes):**

Since the program handles numerous statistics, writing to the database on every request would impact read/write performance. The program uses this strategy:

- Statistics are first stored in **memory**
- Periodically **batch-written** to the database at the configured interval
- Relay balancer runtime state uses the same periodic persistence pattern

**Runtime State Persistence:**

- Auto strategy windows are loaded from the database on startup
- Circuit breaker state is loaded from the database on startup
- Both are saved periodically using the same interval as statistics persistence
- Both are also saved during graceful shutdown

**Key settings cards in the current UI:**

| Card | Purpose |
|------|---------|
| System | Public API base URL, proxy URL, and general runtime settings |
| Semantic Cache | Enablement, TTL, similarity threshold, max entries, embedding base URL / API key / model / timeout |
| Page Order | Drag-and-drop ordering for top-level console pages, persisted globally in settings |
| AI Route | Default compatibility group, timeout, parallelism, and service-pool configuration |
| Retry / Auto Strategy / Circuit Breaker | Relay retry and candidate-selection tuning |
| Log / LLM Price / LLM Sync | Retention, price refresh cadence, and upstream model synchronization |
| Backup | Database export and import |
| Route Group Danger | Delete all route groups with explicit confirmation |

**Semantic Cache Scope:**

- Applies only to non-streaming OpenAI Chat and OpenAI Responses text requests
- Namespaces cache entries by `api_key_id + endpoint_family + requested_model`
- If the embedding client is not fully configured, or embedding lookup / store fails, Octopus bypasses the cache and relays the request normally
- Runtime state and effectiveness are visible in both `Analytics -> Evaluation` and `Ops -> Cache`

**Dangerous Operation in Settings:**

- The Settings page provides **Delete All Route Groups**
- The action requires a second confirmation before execution
- It deletes all groups and group items, then resets the default target group for single-group AI routing to `0` to avoid dangling references

> ⚠️ **Important**: When exiting the program, use proper shutdown methods (like `Ctrl+C` or sending `SIGTERM` signal) to ensure in-memory statistics are correctly written to the database. **Do NOT use `kill -9` or other forced termination methods**, as this may result in statistics data loss.

---

## 🔌 Client Integration

### OpenAI SDK

```python
from openai import OpenAI
import os

client = OpenAI(   
    base_url="http://127.0.0.1:8080/v1",   
    api_key="sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg", 
)
completion = client.chat.completions.create(
    model="octopus-openai",  # Use the correct group name
    messages = [
        {"role": "user", "content": "Hello"},
    ],
)
print(completion.choices[0].message.content)
```

### Claude Code

Edit `~/.claude/settings.json`

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

Edit `~/.codex/config.toml`

```toml
model = "octopus-codex" # Use the correct group name

model_provider = "octopus"

[model_providers.octopus]
name = "octopus"
base_url = "http://127.0.0.1:8080/v1"
```

Edit `~/.codex/auth.json`

```json
{
  "OPENAI_API_KEY": "sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg"
}
```

---

## 🤝 Acknowledgments

- 🙏 [looplj/axonhub](https://github.com/looplj/axonhub) - The LLM API adaptation module in this project is directly derived from this repository
- 📊 [sst/models.dev](https://github.com/sst/models.dev) - AI model database providing model pricing data

