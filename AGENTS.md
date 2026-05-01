# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Runtime and build commands

### Backend
- Start the server: `go run main.go start`
- Start with an explicit config file: `go run main.go start --config /path/to/config.json`
- Build the Go binary only: `go build ./...`
- Show embedded build metadata: `go run main.go version`

### Frontend
- Install frontend deps: `cd web && pnpm install`
- Run the frontend dev server against a local backend: `cd web && NEXT_PUBLIC_API_BASE_URL="http://127.0.0.1:8080" pnpm dev`
- Lint the frontend: `cd web && pnpm lint`
- Build the frontend export: `cd web && NEXT_PUBLIC_APP_VERSION="$(git describe --tags --always 2>/dev/null || printf 'dev')" pnpm build`

### Docker
- Build the release image locally: `docker build --platform linux/amd64 -t lingyuins/octopus:<tag> --build-arg APP_VERSION=<tag> --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) --build-arg BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)" .`
- Verify embedded version metadata: `docker run --rm lingyuins/octopus:<tag> version`

### Tests
- Run all Go tests: `go test ./...`
- Run the existing transformer tests only: `go test ./internal/transformer/model/...`
- Run a single Go test: `go test ./internal/transformer/model -run TestEmbeddingInput_MarshalJSON`

### Release / packaged build
- Canonical release build script: `bash scripts/build.sh release`
- Single-platform packaged build: `bash scripts/build.sh build linux x86_64`

### Full local dev flow
1. Start backend: `go run main.go start`
2. In another shell, start frontend: `cd web && NEXT_PUBLIC_API_BASE_URL="http://127.0.0.1:8080" pnpm dev`
3. Open `http://localhost:3000`

### Building the embedded management UI
The Go server only serves the web UI when static assets exist under `static/out/` and are embedded by `static/static.go`.

Typical local sequence:
- `cd web && pnpm install`
- `cd web && NEXT_PUBLIC_APP_VERSION="$(git describe --tags --always 2>/dev/null || printf 'dev')" pnpm build`
- Copy `web/out/*` into `static/out/`
- If `static/out/_not-found/` exists but is empty, add `.keep` (the release script already handles this)
- Then run `go build` or `go run main.go start`

## Configuration and startup behavior
- Main entrypoint is `main.go`, which delegates to Cobra commands in `cmd/`.
- `start` in `cmd/start.go` loads config, initializes DB, warms caches, starts Gin, then starts background tasks.
- Config is loaded by Viper from `data/config.json` by default, with `OCTOPUS_...` env overrides (`internal/conf/config.go`).
- If `auth.jwt_secret` / `OCTOPUS_AUTH_JWT_SECRET` is unset, startup generates an ephemeral JWT secret, so web login tokens become invalid after restart.
- Initial admin can be bootstrapped with `OCTOPUS_INITIAL_ADMIN_USERNAME` and `OCTOPUS_INITIAL_ADMIN_PASSWORD`.
- Most operational tuning does not live in `config.json`; it is stored in `settings` rows and managed through the web UI / management API, including relay retries, circuit breaking, auto-strategy tuning, relay log retention, AI-route service pool settings, and semantic-cache-related switches.

## High-level architecture

### Request surfaces
There are three distinct HTTP surfaces:
- `/api/v1/bootstrap/...` is the unauthenticated first-run bootstrap surface used before any admin account exists.
- `/api/v1/...` is the management API used by the embedded/exported web UI. It uses JWT auth plus server-side RBAC (`admin` / `editor` / `viewer`).
- `/v1/...` is the public relay API. It accepts OpenAI-compatible and Anthropic-compatible requests, plus direct media/utility endpoints such as images, audio, video, search, rerank, and moderation, all authenticated with Octopus API keys.

### Server composition
- `internal/server/server.go` creates the Gin engine, installs recovery/CORS/static/audit middleware, and registers routes.
- Route registration is side-effect driven: `server.Start()` blank-imports `internal/server/handlers`, and each handler file registers its own routes in `init()` using the custom router registry in `internal/server/router/router.go`.
- Static UI serving is middleware-based. If `static.StaticFS` is nil, the API still runs but the management UI is unavailable.
- Group management includes `DELETE /api/v1/group/delete-all`, which clears all groups and group items and resets the default target group for single-group AI routing to `0`.
- Management API coverage is split by concern: channels, groups, models, settings, stats, logs, analytics, ops, audit, AI route generation, user management, and alerting each live in their own handler file under `internal/server/handlers/`.
- Public relay coverage is split between LLM-style protocol adapters (`relay.go`) and direct media/utility forwarding (`media_relay.go`).

### Core domain model
The system revolves around a few persisted concepts in `internal/model/`:
- `Channel`: one upstream provider configuration, including provider type, base URLs, keys, model declarations, headers, proxying, and auto-sync/grouping options.
- `Group`: routing policy for a requested model. Groups hold ordered/weighted `GroupItem`s, an `endpoint_type`, optional `match_regex`, optional `condition` JSON (currently enforced on the main LLM relay path), first-token timeout, and sticky-session keep time.
- `APIKey`: Octopus-issued client credential for `/v1/...` requests, including supported model restrictions, expiry, max-cost caps, RPM/TPM limits, and optional per-model quota JSON.
- `User`: the management-console login identity, including a server-enforced role (`admin`, `editor`, `viewer`).
- `AlertRule` / `AlertNotifChannel` / `AlertStateRecord` / `AlertHistory`: webhook-based alert definitions, notification targets, current alert state, and emitted history entries.
- `Setting`: runtime tuning for retries, circuit breaking, sync intervals, log retention, auto-strategy weights, AI-route service settings, semantic-cache knobs, etc.
- Stats and relay logs are stored separately and flushed from in-memory caches.

### Persistence and cache model
- `internal/db/db.go` initializes SQLite/MySQL/Postgres via GORM and runs migrations plus `AutoMigrate`.
- `internal/op/` is the main service/repository layer. It fronts most reads through in-memory caches, refreshes them at startup (`op.InitCache()`), and periodically or gracefully flushes mutable runtime state back to the DB (`op.SaveCache()`).
- Channel key usage state, stats, and relay logs are intentionally updated in memory first, then persisted in batches.
- Relay balancer runtime state is persisted separately in `internal/relay/balancer/persistence.go`: auto-strategy windows and circuit-breaker state are restored at startup, saved on a periodic task, and flushed again during graceful shutdown.

### Relay pipeline
The relay path is the most important runtime flow:
1. A `/v1/...` handler picks an inbound protocol adapter in `internal/server/handlers/relay.go`.
2. `internal/relay/relay.go` parses the inbound payload into the internal request model.
3. API-key middleware has already injected supported-model filters and runtime quota metadata (request type, API key id, supported models, RPM/TPM, per-model quota JSON).
4. The requested model is resolved to a `Group` via `internal/op`.
5. Optional group-level `condition` JSON is evaluated before candidate selection on the main LLM relay path.
6. `internal/relay/balancer/` builds a candidate iterator using the group mode (round robin, random, failover, weighted, auto), sticky sessions, and circuit breaker state.
   - `auto` mode explores low-sample candidates first, then sorts by in-window success rate with sample count, weight, priority, and latency data.
7. A `Channel` and channel key are selected, with retry/cooldown logic controlled by settings.
8. An outbound adapter from `internal/transformer/outbound/` converts the internal request into the target provider format and forwards it.
9. Response usage, stats, relay logs, channel-key state, and sticky/circuit-breaker data are recorded.

Important relay details:
- Inbound protocol conversion lives in `internal/transformer/inbound/`.
- Outbound provider conversion lives in `internal/transformer/outbound/`.
- `internal/relay/type.go` reads runtime retry/cooldown limits from settings.
- The special `zen/...` model prefix is interpreted in relay code to steer candidate provider types and upstream model resolution.
- Group regex matching is precompiled from cache rebuilds in `internal/op/group.go`.
- Semantic cache currently applies only to non-streaming OpenAI Chat / OpenAI Responses text requests; bypasses do not block normal relay forwarding.

### Media and utility relay
Not every `/v1/...` endpoint goes through the LLM transformer pipeline:
- `internal/server/handlers/media.go` exposes direct relay endpoints for `/v1/images/generations`, `/v1/images/edits`, `/v1/images/variations`, `/v1/audio/speech`, `/v1/audio/transcriptions`, `/v1/videos/generations`, `/v1/music/generations`, `/v1/search`, `/v1/rerank`, and `/v1/moderations`.
- `internal/relay/media_relay.go` handles these requests by extracting the requested model, resolving groups by `endpoint_type`, forwarding JSON or multipart payloads, and streaming JSON / SSE / binary responses back to the client.
- This direct forwarding path currently resolves groups by `endpoint_type` + requested model and does not invoke `internal/relay/condition`.
- Image edits / variations and audio transcription are multipart endpoints; image generation, speech, video, music, search, rerank, and moderation are JSON endpoints.

### Auth model
- Admin UI auth: JWT bearer tokens validated by `internal/server/middleware/auth.go` and created in `internal/server/auth/auth.go`.
- Admin permissions: `internal/server/auth/permissions.go` defines fine-grained permissions, and `internal/server/middleware/rbac.go` enforces them for route groups and route-level writes.
- Auth middleware reloads the current cached user role on each request instead of trusting only the JWT claim.
- Relay auth: Octopus API keys with prefix `sk-octopus-...`. The middleware accepts `Authorization: Bearer ...` for OpenAI-style clients and `x-api-key` for Anthropic-style clients.

### Background work
`internal/task/init.go` wires periodic jobs based on DB-backed settings:
- model price refresh
- base URL latency probing
- upstream model synchronization
- stats flush
- balancer runtime state flush
- relay log flush
- alert rule evaluation

Channel creation/update also kicks off async helper work such as model discovery, price hydration, base URL delay probing, and auto-grouping.

## Frontend structure
- The frontend is a Next.js app in `web/` using App Router only as a shell entrypoint; the actual in-app screen switching is client-side inside `web/src/components/app.tsx`.
- Main sections are lazy-loaded from `web/src/components/modules/*` and registered in `web/src/route/config.tsx`.
- API access is centralized in `web/src/api/client.ts`; by default it uses relative base URL `.` unless `NEXT_PUBLIC_API_BASE_URL` is set.
- Production frontend output is a static export (`next.config.ts` uses `output: "export"`) that gets copied into `static/out/` for Go embedding.
- Dangerous settings actions live under `web/src/components/modules/setting/`; bulk route-group deletion uses a confirmation dialog in `RouteGroupDanger.tsx`.
- Current primary modules are Home, Channel, Group, Model (Model Market), Analytics, Log, Alert, Ops, Setting, and User.
- The settings module now includes dedicated Semantic Cache controls, while top-level page order lives inside the Appearance card rather than a standalone Page Order card.

## Files to inspect first for common tasks
- Startup / wiring: `main.go`, `cmd/start.go`
- Config: `internal/conf/config.go`
- DB init / migrations: `internal/db/db.go`, `internal/db/migrate/`
- Route registration: `internal/server/server.go`, `internal/server/router/router.go`, `internal/server/handlers/`
- Relay behavior: `internal/relay/relay.go`, `internal/relay/media_relay.go`, `internal/relay/type.go`, `internal/relay/balancer/`
- RBAC / auth: `internal/server/auth/permissions.go`, `internal/server/middleware/auth.go`, `internal/server/middleware/rbac.go`, `internal/server/handlers/user.go`
- Alerting: `internal/server/handlers/alert.go`, `internal/task/alert.go`, `internal/op/alert.go`
- Analytics / Ops / Audit: `internal/server/handlers/analytics.go`, `internal/server/handlers/ops.go`, `internal/server/handlers/audit.go`, `internal/op/analytics.go`, `internal/op/ops.go`, `internal/op/audit_log.go`
- Protocol adapters: `internal/transformer/inbound/`, `internal/transformer/outbound/`
- Cache-backed operations: `internal/op/`
- Frontend entry: `web/src/components/app.tsx`, `web/src/route/config.tsx`, `web/src/components/modules/navbar/nav-order.ts`
- Embedded static serving: `static/static.go`
