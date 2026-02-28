# Contributing to Permission Slip

Thanks for your interest in contributing to Permission Slip! This guide covers everything you need to develop, test, and submit changes.

## Getting Started

1. Fork the repository and clone your fork
2. Follow the [Getting Started](README.md#getting-started) section in the README to install dependencies and set up the database
3. Create a feature branch from `main`

## Development Workflow

1. **Create a branch** — use a descriptive name (e.g., `add-calendar-action`, `fix-approval-timeout`)
2. **Make your changes** — follow the code standards below
3. **Run tests** — `make test` runs both backend and frontend tests
4. **Run the build** — `make build` catches TypeScript compilation errors that tests may miss
5. **Open a pull request** — describe what you changed and why

## Environment Variables

```bash
cp .env.example .env
```

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Port the Go server listens on |
| `MODE` | `development` | Set to `development` to skip serving embedded static files (use Vite instead) |
| `DATABASE_URL` | _(empty)_ | Postgres connection string for the running app. If unset, the server starts without a database. |
| `DATABASE_URL_TEST` | _(empty)_ | Postgres connection string used by `go test` and CI. |
| `SUPABASE_URL` | _(empty)_ | Supabase project URL (e.g. `http://127.0.0.1:54321` locally, `https://xxx.supabase.co` in production). The backend derives the JWKS endpoint from this automatically. **Required for Supabase CLI v2+ (ES256 JWT signing).** |
| `SUPABASE_JWKS_URL` | _(derived from `SUPABASE_URL`)_ | Override for the JWKS endpoint. Defaults to `$SUPABASE_URL/auth/v1/.well-known/jwks.json`. |
| `SUPABASE_JWT_SECRET` | _(empty)_ | HS256 HMAC secret for legacy JWT validation (Supabase CLI v1 only). **Not needed for CLI v2+** — use `SUPABASE_URL` instead. |
| `BASE_URL` | _(empty)_ | Public base URL (e.g. `https://app.permissionslip.dev`). Used to construct invite URLs for agent registration. |
| `INVITE_HMAC_KEY` | _(empty)_ | HMAC key for hashing invite codes. If set, uses HMAC-SHA256 instead of plain SHA-256. Generate with `openssl rand -hex 32`. |
| `VAULT_SECRET_KEY` | _(empty)_ | Encryption key for Supabase Vault (AES-256-GCM). Required when running with Supabase (`supabase start`). Not needed for CI/tests (tests use MockVaultStore). Generate with `openssl rand -hex 32`. |
| `VITE_SUPABASE_URL` | _(empty)_ | Supabase project URL, used by the React frontend. |
| `VITE_SUPABASE_ANON_KEY` | _(empty)_ | Supabase anonymous/public key, used by the React frontend. |

## Setting Up Supabase Locally

The frontend uses Supabase for Auth, Storage, and Realtime. For local development, you can run the full Supabase stack using the Supabase CLI and Docker.

**Prerequisites:**

- **Docker Desktop** — [Install Docker](https://www.docker.com/products/docker-desktop/) (must be running before starting Supabase)
- **Supabase CLI** — Install via Homebrew:

```bash
brew install supabase/tap/supabase
```

**Start Supabase:**

```bash
supabase init   # only needed the first time (creates supabase/ directory)
supabase start
```

This pulls and starts the Supabase Docker containers (Postgres, Auth, Storage, Realtime, Studio, etc.). The first run takes a few minutes to download images.

**Get your local credentials:**

```bash
supabase status
```

This prints the local URLs and keys. Update your `.env` with the values:

```bash
VITE_SUPABASE_URL=http://127.0.0.1:54321
VITE_SUPABASE_ANON_KEY=<anon key from supabase status>
```

**Supabase Studio:** Open **http://127.0.0.1:54323** to access the GUI for managing your local database, viewing tables, editing RLS policies, and testing auth.

**Stop / Restart:**

```bash
supabase stop
supabase start
```

> **Tip:** If `supabase start` hangs on health checks, use `supabase start --ignore-health-check` — the services are usually functional even when health checks time out.

> **Important:** Changes to `supabase/config.toml` (auth settings, MFA, rate limits, etc.) only take effect after a Supabase restart. You do not need to restart Supabase when restarting the Vite dev server.

> **Note:** The local Supabase stack runs its own Postgres instance on port 54322. This is separate from the standalone Postgres used by the Go backend (port 5432). Both can run simultaneously.

## Code Standards

### Project Structure

```
api/              Go API route handlers (one file per domain)
db/               Database queries, migrations, test helpers
frontend/src/
  api/            Generated API client and types
  components/     Shared/reusable UI components
  components/ui/  shadcn/ui components (owned source, not a package)
  pages/          Page components organized by feature
  hooks/          Custom React hooks
  lib/            Utility functions
spec/openapi/     OpenAPI specification (source of truth for API types)
docs/             Architecture docs, ADRs, schema reference
```

### Backend (Go)

- API route handlers live in `api/` — each domain gets its own file with a `Register*Routes` function
- Database queries live in `db/` — no SQL in API handlers
- JWT validation supports both ES256 (Supabase CLI v2+) and HS256 (legacy)
- See [docs/api-scaffolding.md](docs/api-scaffolding.md) for error handling, JSON helpers, and trace ID conventions

### Frontend (React + TypeScript)

- **One component per file.** If a section of JSX needs a comment explaining what it is, extract it into its own component.
- **Pages** go in `frontend/src/pages/<feature>/`. Shared components go in `frontend/src/components/`.
- **Container vs presentational** — components either manage data (container) or render UI (presentational). Don't mix concerns.
- **Business logic** belongs in custom hooks or utility functions, not in components.
- **No `any` types.** No `as` casts without a comment explaining why the cast is safe.
- **React Query** for all server state — no `useEffect` + `useState` for API calls. Cache keys must mirror the API endpoint structure.

### Design System

The frontend uses **shadcn/ui** components built on **Radix UI** primitives, styled with **Tailwind CSS v4**, and using **Lucide React** icons. See [ADR-006](docs/adr/006-design-system.md) for the full rationale.

- **Components live in** `frontend/src/components/ui/` — these are owned source files (not a package dependency), so they can be customized freely.
- **Theme** is defined via CSS custom properties in `frontend/src/index.css` (light and dark mode).
- **Utility function** `cn()` in `frontend/src/lib/utils.ts` merges Tailwind classes (via `clsx` + `tailwind-merge`).
- **Path alias** `@/` maps to `frontend/src/` for clean imports.

**Adding new shadcn/ui components:**

```bash
cd frontend
npx shadcn@latest add <component-name>
```

Or copy the component source manually into `src/components/ui/` from the [shadcn/ui docs](https://ui.shadcn.com/docs/components).

## Adding API Endpoints

Each domain owns its routes in a separate file under `api/`. The file exposes a `Register*Routes` function that `NewRouter` calls:

```go
// api/agents.go — session-authenticated route
func RegisterAgentRoutes(mux *http.ServeMux, deps *Deps) {
    requireProfile := RequireProfile(deps)
    mux.Handle("GET /agents", requireProfile(handleListAgents(deps)))
    mux.Handle("GET /agents/{agent_id}", requireProfile(handleGetAgent(deps)))
    // ...
}
```

For routes that require a logged-in user, wrap the handler with `RequireSession` (JWT only) or `RequireProfile` (JWT + profile lookup). For agent-authenticated routes, use `RequireAgentSignature`:

```go
// api/profiles.go — requires session + profile in database
func RegisterProfileRoutes(mux *http.ServeMux, deps *Deps) {
    requireProfile := RequireProfile(deps)
    mux.Handle("GET /profile", requireProfile(handleGetProfile()))
}
```

`RequireProfile` chains `RequireSession` and looks up the user's profile. Use it when the handler needs the profile or when you want to guarantee the user has a profile row. Use `RequireSession` alone for routes that only need the user ID.

To add a new domain, create a new file (e.g., `api/webhooks.go`) with a `Register*Routes` function, then add the call in `NewRouter`. See [docs/api-scaffolding.md](docs/api-scaffolding.md) for the full router listing.

The pattern format is `METHOD /path` (Go 1.22+ routing). All `/api/*` routes are automatically proxied from Vite in development.

### API Types and Client

The OpenAPI spec (`spec/openapi/`) is the single source of truth for all API types and endpoints. The typed client lives in `frontend/src/api/client.ts` (using `openapi-fetch`), and types are generated via `openapi-typescript`.

- **Never hand-write API types or request functions.** Always generate them from the OpenAPI spec (`npm run generate:api` from `frontend/`). If a type doesn't exist in the generated output, the spec needs updating — not a manual type.
- Import all request/response types from the generated client (`api/schema.d.ts`).
- New backend API calls must use the `openapi-fetch` client instance in `api/client.ts` — never raw `fetch` for our API.
- Wrap API calls in custom hooks that handle loading, error, and empty states.

When adding or modifying endpoints:

1. Update the OpenAPI spec in `spec/openapi/`
2. Run `make generate` to regenerate TypeScript types
3. Implement the Go handler in `api/`
4. Add a custom hook in the frontend to consume it

## Adding Dependencies

**Go:**

```bash
go get github.com/some/package
```

This updates `go.mod` and `go.sum` automatically.

**Frontend:**

```bash
cd frontend
npm install some-package
```

## Testing

```bash
make test              # all tests (backend + frontend)
make test-backend      # Go tests (requires Postgres)
make test-frontend     # frontend tests (no database needed)
cd frontend && npm run test:watch   # watch mode for frontend
```

### Testing Strategy

This project follows a hybrid approach (see [ADR-004](docs/adr/004-local-testing-strategy.md)):

- **Backend tests** run against a real local Postgres database. This catches real SQL issues, validates migrations, and tests RLS policies. The test helper (`db/testhelper`) handles connection setup and table truncation between tests via `t.Cleanup()`.
- **Frontend tests** use a mocked Supabase client (`src/__mocks__/supabaseClient.ts`). Auth, Storage, and Realtime are mocked at the module boundary since they're HTTP-based wrappers. Set return values per-test as needed.

No Docker required for tests — just a local Postgres install.

> **Before running tests for the first time** (or after pulling new dependencies), make sure frontend deps are installed:
>
> ```bash
> cd frontend && npm install
> ```

## Database Migrations

Migrations use [goose](https://github.com/pressly/goose) and live in `db/migrations/` as SQL files.

```bash
make migrate-create NAME=add_users_table   # create a new migration
make migrate-up                             # run pending migrations
make migrate-down                           # roll back last migration
```

Migrations are embedded into the Go binary and run automatically on startup when `DATABASE_URL` is set. They can also be run manually via the `cmd/migrate` CLI.

For the full schema (tables, relationships, constraints, and conventions), see [docs/database-schema.md](docs/database-schema.md).

Migration files use the goose format:

```sql
-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE users;
```

## Reporting Issues

Use [GitHub Issues](https://github.com/supersuit-tech/permission-slip-web/issues) to report bugs or suggest features. Include:

- Steps to reproduce (for bugs)
- Expected vs actual behavior
- Your environment (OS, Go version, Node version)

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
