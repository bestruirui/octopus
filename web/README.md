# Octopus Web Console

This directory contains the management UI for Octopus.

## Stack

- Next.js 16
- React 19
- TypeScript
- Tailwind CSS 4
- TanStack Query
- `next-intl`

The app uses App Router as the shell entrypoint, but the actual screen switching inside the console is handled client-side in `src/components/app.tsx`.

## Commands

Install dependencies:

```bash
pnpm install
```

Run the frontend against a local backend:

```bash
NEXT_PUBLIC_API_BASE_URL="http://127.0.0.1:8080" pnpm dev
```

Lint:

```bash
pnpm lint
```

Build the static export used by the embedded management UI:

```bash
NEXT_PUBLIC_APP_VERSION="$(git describe --tags --always 2>/dev/null || printf 'dev')" pnpm build
```

## Environment Variables

- `NEXT_PUBLIC_API_BASE_URL`: Optional API base URL. Defaults to relative requests against the current origin.
- `NEXT_PUBLIC_APP_VERSION`: Version string shown in the UI. For release builds, set this to the current git tag or commit.

## Output and Embedding

`pnpm build` produces a static export in `out/`.

The Go server embeds these files from `../static/out/`. A typical local embed flow is:

```bash
pnpm install
NEXT_PUBLIC_APP_VERSION="$(git describe --tags --always 2>/dev/null || printf 'dev')" pnpm build
cd ..
mkdir -p static/out
cp -r web/out/* static/out/
```

If `static/out/_not-found/` exists but is empty, add `.keep` before running `go build` or `go run main.go start`.

The top-level `Dockerfile` already builds this frontend and copies the export into `static/out` during image build, so release images contain the matching frontend automatically.

## Key Directories

- `src/components/app.tsx`: Main application shell
- `src/components/modules/model/*`: Model Market UI, including summary strip, virtualized cards, and price-edit actions
- `src/components/modules/analytics/*`: Utilization, route-health, and evaluation surfaces, plus shared overview cards reused by Home
- `src/components/modules/ops/*`: Cache, quota, health, system, and audit surfaces
- `src/components/modules/setting/*`: Settings cards including info, appearance/nav preferences, semantic cache, AI route, API key defaults, backup, and dangerous actions
- `src/components/modules/navbar/*`: Top-level navigation state and persisted nav-order helpers
- `src/api/`: API client and endpoint hooks
- `src/route/config.tsx`: UI route registration
- `public/locale/`: Localized text resources

## Notes

- The settings module includes dangerous operations such as deleting all route groups; UI confirmation is required before executing them.
- Group mode labels and endpoint type display values are shared with backend behavior and should be updated together when adding new strategies or capabilities.
- AI routing has two entry points: the route page button generates the full routing table, while the group edit dialog button appends matched items into the current group only.
- The settings field for AI routing is now the default target group for the single-group compatibility flow, not the target for full-table generation.
- The `Model` route is now a `Model Market` view backed by `/api/v1/model/market`; it merges pricing, coverage, enabled-key counts, latency, and success metrics while preserving price-management actions.
- `Analytics` is organized into `utilization`, `route-health`, and `evaluation` tabs. The overview query remains available, but its primary UI summary cards now live on Home.
- `Ops` is organized into `cache`, `quota`, `health`, `system`, and `audit` tabs. Audit only covers selected management write routes, not public relay traffic.
- Semantic cache settings are split into configured state and runtime-enabled state. Enabling the switch alone is not enough; the embedding base URL and embedding model also need to be configured before runtime metrics turn green.
- Top-level page order is edited inside the `Appearance` card, persisted through the `nav_order` setting, and normalized against `DEFAULT_NAV_ORDER`, so missing routes are appended automatically and unknown routes are dropped.
