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

## High-level architecture

### Request surfaces
There are two distinct HTTP surfaces:
- `/api/v1/...` is the management API used by the embedded/exported web UI. It uses JWT auth for admin users.
- `/v1/...` is the public relay API. It accepts OpenAI-compatible and Anthropic-compatible requests and authenticates with Octopus API keys.

### Server composition
- `internal/server/server.go` creates the Gin engine, installs recovery/CORS/static middleware, and registers routes.
- Route registration is side-effect driven: `server.Start()` blank-imports `internal/server/handlers`, and each handler file registers its own routes in `init()` using the custom router registry in `internal/server/router/router.go`.
- Static UI serving is middleware-based. If `static.StaticFS` is nil, the API still runs but the management UI is unavailable.
- Group management includes `DELETE /api/v1/group/delete-all`, which clears all groups and group items and resets the default target group for single-group AI routing to `0`.

### Core domain model
The system revolves around a few persisted concepts in `internal/model/`:
- `Channel`: one upstream provider configuration, including provider type, base URLs, keys, model declarations, headers, proxying, and auto-sync/grouping options.
- `Group`: routing policy for a requested model. Groups hold ordered/weighted `GroupItem`s that map requested models to candidate channels.
- `APIKey`: Octopus-issued client credential for `/v1/...` requests, including supported model restrictions and cost caps.
- `Setting`: runtime tuning for retries, circuit breaking, sync intervals, log retention, etc.
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
3. The requested model is resolved to a `Group` via `internal/op`.
4. `internal/relay/balancer/` builds a candidate iterator using the group mode (round robin, random, failover, weighted, auto), sticky sessions, and circuit breaker state.
   - `auto` mode explores low-sample candidates first, then sorts by in-window success rate with sample count, weight, and priority as tie-breakers.
5. A `Channel` and channel key are selected, with retry/cooldown logic controlled by settings.
6. An outbound adapter from `internal/transformer/outbound/` converts the internal request into the target provider format and forwards it.
7. Response usage, stats, relay logs, channel-key state, and sticky/circuit-breaker data are recorded.

Important relay details:
- Inbound protocol conversion lives in `internal/transformer/inbound/`.
- Outbound provider conversion lives in `internal/transformer/outbound/`.
- `internal/relay/type.go` reads runtime retry/cooldown limits from settings.
- The special `zen/...` model prefix is interpreted in relay code to steer candidate provider types and upstream model resolution.

### Auth model
- Admin UI auth: JWT bearer tokens validated by `internal/server/middleware/auth.go` and created in `internal/server/auth/auth.go`.
- Relay auth: Octopus API keys with prefix `sk-octopus-...`. The middleware accepts `Authorization: Bearer ...` for OpenAI-style clients and `x-api-key` for Anthropic-style clients.

### Background work
`internal/task/init.go` wires periodic jobs based on DB-backed settings:
- model price refresh
- base URL latency probing
- upstream model synchronization
- stats flush
- balancer runtime state flush
- relay log flush

Channel creation/update also kicks off async helper work such as model discovery, price hydration, base URL delay probing, and auto-grouping.

## Frontend structure
- The frontend is a Next.js app in `web/` using App Router only as a shell entrypoint; the actual in-app screen switching is client-side inside `web/src/components/app.tsx`.
- Main sections are lazy-loaded from `web/src/components/modules/*` and registered in `web/src/route/config.tsx`.
- API access is centralized in `web/src/api/client.ts`; by default it uses relative base URL `.` unless `NEXT_PUBLIC_API_BASE_URL` is set.
- Production frontend output is a static export (`next.config.ts` uses `output: "export"`) that gets copied into `static/out/` for Go embedding.
- Dangerous settings actions live under `web/src/components/modules/setting/`; bulk route-group deletion uses a confirmation dialog in `RouteGroupDanger.tsx`.

## Files to inspect first for common tasks
- Startup / wiring: `main.go`, `cmd/start.go`
- Config: `internal/conf/config.go`
- DB init / migrations: `internal/db/db.go`, `internal/db/migrate/`
- Route registration: `internal/server/server.go`, `internal/server/router/router.go`, `internal/server/handlers/`
- Relay behavior: `internal/relay/relay.go`, `internal/relay/type.go`, `internal/relay/balancer/`
- Protocol adapters: `internal/transformer/inbound/`, `internal/transformer/outbound/`
- Cache-backed operations: `internal/op/`
- Frontend entry: `web/src/components/app.tsx`, `web/src/route/config.tsx`
- Embedded static serving: `static/static.go`
