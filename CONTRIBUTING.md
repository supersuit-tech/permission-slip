# Contributing to Permission Slip

Thanks for your interest in contributing to Permission Slip! Whether you're fixing a typo, reporting a bug, or building a new feature, we appreciate your help making this project better.

This guide covers everything you need to know to get started.

## Table of Contents

- [Ways to Contribute](#ways-to-contribute)
- [Finding an Issue to Work On](#finding-an-issue-to-work-on)
- [Development Setup](#development-setup)
- [Architecture Overview](#architecture-overview)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Adding API Endpoints](#adding-api-endpoints)
- [Database Migrations](#database-migrations)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Commit Messages](#commit-messages)
- [Reporting Bugs](#reporting-bugs)
- [Requesting Features](#requesting-features)
- [Troubleshooting](#troubleshooting)
- [Getting Help](#getting-help)
- [License](#license)

## Ways to Contribute

There are many ways to contribute beyond writing code:

- **Report bugs** — found something broken? [Open an issue](#reporting-bugs)
- **Suggest features** — have an idea? [Request a feature](#requesting-features)
- **Improve documentation** — fix typos, clarify instructions, add examples
- **Write tests** — improve coverage or add missing edge cases
- **Review pull requests** — help review open PRs with constructive feedback
- **Build connectors** — create connectors for new services (see [Creating Connectors](docs/creating-connectors.md))
- **Answer questions** — help other contributors in issues and discussions
- **Share Permission Slip** — blog about it, talk about it, star the repo

## Finding an Issue to Work On

New to the project? Look for issues labeled:

- **`good first issue`** — small, well-scoped tasks ideal for first-time contributors
- **`help wanted`** — issues where we'd especially appreciate community help
- **`documentation`** — docs improvements that don't require deep codebase knowledge

Browse open issues: [github.com/supersuit-tech/permission-slip-web/issues](https://github.com/supersuit-tech/permission-slip-web/issues)

Before starting work on an issue, leave a comment to let others know you're working on it. This avoids duplicate effort. If an issue is already assigned or has a recent comment claiming it, consider picking a different one or asking if help is still needed.

If you want to work on something that doesn't have an issue yet, open one first so we can discuss the approach before you invest time.

## Development Setup

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| [Go](https://go.dev/doc/install) | 1.24+ | Backend API server |
| [Node.js](https://nodejs.org/) | 20+ | Frontend build tooling |
| [PostgreSQL](https://www.postgresql.org/download/) | 16 | Database (required for backend tests) |
| [Docker](https://www.docker.com/products/docker-desktop/) | Latest | Required for local Supabase |
| [Supabase CLI](https://supabase.com/docs/guides/local-development/cli/getting-started) | Latest | Local auth, storage, and realtime |

### Quick Start

```bash
# 1. Fork the repo on GitHub, then clone your fork
git clone https://github.com/<your-username>/permission-slip-web.git
cd permission-slip-web

# 2. Add the upstream remote
git remote add upstream https://github.com/supersuit-tech/permission-slip-web.git

# 3. Install dependencies
make install

# 4. Copy environment variables
cp .env.example .env

# 5. Start Supabase (requires Docker running)
supabase start
supabase status   # copy the publishable key into .env as VITE_SUPABASE_PUBLISHABLE_KEY

# 6. Run database migrations
make migrate-up

# 7. Generate the typed API client
make generate

# 8. Start development servers
make dev-backend    # Go API server on port 8080 (terminal 1)
make dev-frontend   # Vite dev server on port 5173 (terminal 2)
```

Open **http://localhost:5173** — API requests to `/api/*` are automatically proxied to the Go server.

> **Tip:** You can also run `make dev` to start both servers in one terminal, but separate terminals make it easier to see logs.

### Environment Variables

Run `cp .env.example .env` and fill in the values. The essential variables for local development:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Port the Go server listens on |
| `MODE` | `development` | Set to `development` to use Vite instead of embedded static files |
| `DATABASE_URL` | _(empty)_ | Postgres connection string for the running app |
| `DATABASE_URL_TEST` | _(empty)_ | Postgres connection string for `go test` and CI |
| `SUPABASE_URL` | _(empty)_ | Supabase project URL (e.g. `http://127.0.0.1:54321` locally) |
| `VITE_SUPABASE_URL` | _(empty)_ | Supabase URL for the React frontend |
| `VITE_SUPABASE_PUBLISHABLE_KEY` | _(empty)_ | Supabase publishable key for the React frontend |
| `BASE_URL` | _(empty)_ | Public base URL — used for invite URLs |
| `INVITE_HMAC_KEY` | _(empty)_ | HMAC key for invite codes — generate with `openssl rand -hex 32` |
| `VAULT_SECRET_KEY` | _(empty)_ | Encryption key for Supabase Vault (AES-256-GCM) — generate with `openssl rand -hex 32` |

See [.env.example](.env.example) for the full list with descriptions.

### Setting Up Supabase Locally

The frontend uses Supabase for Auth, Storage, and Realtime.

```bash
supabase init       # only needed the first time
supabase start      # start the local stack (pulls Docker images on first run)
supabase status     # get local URLs and keys
```

Copy the output into your `.env`:

```bash
VITE_SUPABASE_URL=http://127.0.0.1:54321
VITE_SUPABASE_PUBLISHABLE_KEY=<publishable key from supabase status>
```

**Supabase Studio:** Open **http://127.0.0.1:54323** to manage your local database, tables, RLS policies, and auth.

> **Tip:** If `supabase start` hangs on health checks, use `supabase start --ignore-health-check` — services are usually functional.

> **Note:** The local Supabase stack runs its own Postgres on port 54322. This is separate from the standalone Postgres used for Go backend tests (port 5432). Both can run simultaneously.

## Architecture Overview

Permission Slip is a monorepo with a Go backend and React frontend:

```
permission-slip/
├── api/                 # Go API route handlers (one file per domain)
├── connectors/          # Service connectors (Slack, GitHub, custom)
├── db/                  # Database queries, migrations, test helpers
├── cmd/                 # CLI tools (migrate, VAPID keys, connector installer)
├── frontend/src/
│   ├── api/             # Generated API client and types (from OpenAPI spec)
│   ├── auth/            # Supabase auth integration
│   ├── components/      # Shared/reusable UI components
│   │   └── ui/          # shadcn/ui components (owned source)
│   ├── pages/           # Page components organized by feature
│   ├── hooks/           # Custom React hooks
│   └── lib/             # Utility functions
├── mobile/              # React Native (Expo) mobile approval app
│   ├── __tests__/       # Jest tests
│   └── assets/          # App icons and splash screen
├── spec/openapi/        # OpenAPI specification (source of truth for API types)
├── docs/                # Architecture docs, ADRs, guides
│   └── adr/             # Architecture Decision Records
├── docs-site/           # Mintlify user-facing documentation
├── main.go              # Server entry point
├── router.go            # HTTP route registration
└── Makefile             # Build orchestration
```

Key architectural concepts:

- **Actions** are the core primitive — agents submit structured, schema-validated actions (never arbitrary API calls)
- **Connectors** define what actions are available and how they execute (e.g., Slack `send_message`, GitHub `create_issue`)
- **The OpenAPI spec** is the single source of truth for all API types — TypeScript types are generated from it, never hand-written
- **Backend tests** run against real Postgres — no mocks for the database layer
- **Frontend tests** mock Supabase at the module boundary

For deeper dives, see:
- [Architecture](docs/architecture.md) — system diagrams and component overview
- [SPEC.md](SPEC.md) — protocol design and security model
- [Terminology](docs/spec/terminology.md) — core concepts and definitions
- [Architecture Decision Records](docs/adr/) — rationale behind key design choices

## Development Workflow

1. **Sync your fork** with upstream:

   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. **Create a feature branch** from `main`:

   ```bash
   git checkout -b my-feature-name
   ```

   Use a descriptive branch name — e.g., `add-calendar-connector`, `fix-approval-timeout`, `docs-update-setup`.

3. **Make your changes** — follow the [code standards](#code-standards) below.

4. **Run tests** before committing:

   ```bash
   make test                # all tests
   # Or run just what you changed:
   make test-backend        # Go changes
   make test-frontend       # frontend changes
   make mobile-test         # mobile changes
   ```

5. **Run the build** to catch TypeScript compilation errors tests may miss:

   ```bash
   make build
   ```

6. **Commit your changes** with a [clear commit message](#commit-messages).

7. **Push and open a pull request** against `main`.

### Useful Make Targets

| Command | Description |
|---------|-------------|
| `make install` | Install Go and npm dependencies |
| `make setup` | Full setup: install + generate API client |
| `make dev` | Run backend + frontend servers together |
| `make dev-backend` | Go server on port 8080 |
| `make dev-frontend` | Vite on port 5173 with HMR |
| `make test` | All tests (backend + frontend + mobile) |
| `make test-backend` | Go tests (requires Postgres) |
| `make test-frontend` | Frontend tests (no database needed) |
| `make mobile-test` | Mobile tests (no database needed) |
| `make mobile-install` | Install mobile app dependencies |
| `make mobile-start` | Start Expo development server |
| `make mobile-typecheck` | Type-check mobile app (`tsc --noEmit`) |
| `make build` | Production binary with embedded frontend |
| `make generate` | Bundle OpenAPI spec + generate TypeScript types |
| `make typecheck` | Generate API client + run `tsc --noEmit` |
| `make bundle` | Bundle multi-file OpenAPI spec |
| `make migrate-up` | Run pending database migrations |
| `make migrate-down` | Roll back the last migration |
| `make migrate-create NAME=foo` | Create a new migration file |
| `make seed` | Seed the dev database with test data |
| `make audit` | Run npm audit (frontend + mobile) + govulncheck |

## Code Standards

### Backend (Go)

- API route handlers live in `api/` — each domain gets its own file with a `Register*Routes` function
- Database queries live in `db/` — no SQL in API handlers
- JWT validation supports both ES256 (Supabase CLI v2+) and HS256 (legacy)
- Error responses use structured JSON with error codes (see `api/error_codes.go`)
- See [docs/api-scaffolding.md](docs/api-scaffolding.md) for error handling, JSON helpers, and trace ID conventions

### Frontend (React + TypeScript)

- **One component per file.** If a section of JSX needs a comment explaining what it is, extract it into its own component.
- **Pages** go in `frontend/src/pages/<feature>/`. Shared components go in `frontend/src/components/`.
- **Container vs presentational** — components either manage data (container) or render UI (presentational). Don't mix concerns.
- **Business logic** belongs in custom hooks or utility functions, not in components.
- **No `any` types.** No `as` casts without a comment explaining why the cast is safe.
- **React Query** (TanStack Query) for all server state — no `useEffect` + `useState` for API calls.
- Cache keys must mirror the API endpoint structure for predictable invalidation.
- `noUncheckedIndexedAccess` is enabled — use `.charAt()`, nullish coalescing, or type narrowing for indexed access.

### Design System

The frontend uses **shadcn/ui** components built on **Radix UI** primitives, styled with **Tailwind CSS v4**, and using **Lucide React** icons. See [ADR-006](docs/adr/006-design-system.md) for the rationale.

- Components live in `frontend/src/components/ui/` — these are owned source files, not a package dependency, so they can be customized.
- Theme is defined via CSS custom properties in `frontend/src/index.css` (light and dark mode).
- Utility function `cn()` in `frontend/src/lib/utils.ts` merges Tailwind classes (via `clsx` + `tailwind-merge`).
- Path alias `@/` maps to `frontend/src/` for clean imports.

**Adding new shadcn/ui components:**

```bash
cd frontend
npx shadcn@latest add <component-name>
```

Or copy the component source manually into `src/components/ui/` from the [shadcn/ui docs](https://ui.shadcn.com/docs/components).

### API Types and Client

The OpenAPI spec (`spec/openapi/`) is the single source of truth for all API types. The typed client uses `openapi-fetch`, and types are generated via `openapi-typescript`.

- **`frontend/src/api/schema.d.ts` and `mobile/src/api/schema.d.ts` are gitignored** — they are produced locally when you run `make generate` (or `npm run generate:api` per package). PRs that change the spec should include the updated YAML and bundled `openapi.bundle.yaml`; reviewers run `make generate` to refresh their working copies.
- **Never hand-write API types or request functions.** Generate from the spec with `make generate`. If a type doesn't exist in the generated output, the spec needs updating — not a manual type.
- Import all request/response types from the generated client (`api/schema.d.ts`).
- New API calls must use the `openapi-fetch` client in `api/client.ts` — never raw `fetch` for our API.
- Wrap API calls in custom hooks that handle loading, error, and empty states.

When adding or modifying endpoints:

1. Update the OpenAPI spec in `spec/openapi/`
2. Run `make generate` to regenerate TypeScript types
3. Implement the Go handler in `api/`
4. Add a custom hook in the frontend to consume it

See the [OpenAPI spec README](spec/openapi/README.md) for spec authoring guidelines.

## Adding API Endpoints

Each domain owns its routes in a separate file under `api/`. The file exposes a `Register*Routes` function that `NewRouter` calls:

```go
// api/agents.go — session-authenticated route
func RegisterAgentRoutes(mux *http.ServeMux, deps *Deps) {
    requireProfile := RequireProfile(deps)
    mux.Handle("GET /agents", requireProfile(handleListAgents(deps)))
    mux.Handle("GET /agents/{agent_id}", requireProfile(handleGetAgent(deps)))
}
```

For routes that require a logged-in user, wrap the handler with:
- `RequireSession` — validates JWT only (use when you only need the user ID)
- `RequireProfile` — validates JWT + looks up the user's profile (use when the handler needs profile data)
- `RequireAgentSignature` — for agent-authenticated routes

To add a new domain, create a new file (e.g., `api/webhooks.go`) with a `Register*Routes` function, then register it in `NewRouter`. The pattern format is `METHOD /path` (Go 1.22+ routing). All `/api/*` routes are automatically proxied from Vite in development.

## Database Migrations

Migrations use [goose](https://github.com/pressly/goose) and live in `db/migrations/` as SQL files.

```bash
make migrate-create NAME=add_users_table   # create a new migration
make migrate-up                             # run pending migrations
make migrate-down                           # roll back last migration
```

Migrations are embedded into the Go binary and run automatically on startup when `DATABASE_URL` is set.

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

Every migration must include both `Up` and `Down` sections. For the full schema reference, see [docs/database-schema.md](docs/database-schema.md).

## Testing

```bash
make test              # all tests (backend + frontend + mobile)
make test-backend      # Go tests (requires Postgres)
make test-frontend     # frontend tests (no database needed)
make mobile-test       # mobile tests (no database needed)
cd frontend && npm run test:watch   # watch mode for frontend
```

### Testing Strategy

This project follows a hybrid approach (see [ADR-004](docs/adr/004-local-testing-strategy.md)):

- **Backend tests** run against a real local Postgres database. This catches real SQL issues, validates migrations, and tests RLS policies. The test helper (`db/testhelper`) handles connection setup and table truncation between tests via `t.Cleanup()`.
- **Frontend tests** use a mocked Supabase client (`src/__mocks__/supabaseClient.ts`). Auth, Storage, and Realtime are mocked at the module boundary. Set return values per-test as needed.
- **Integration tests** run when Supabase is detected locally. They test the full stack including auth flows. Run explicitly with `make test-integration`.

No Docker required for unit tests — just a local Postgres install.

### What to Test

- **Changed Go code** — run `make test-backend`
- **Changed frontend code** — run `make test-frontend`
- **Changed mobile code** — run `make mobile-test`
- **Changed migrations or `db/` code** — run `go test ./db/... -v` at minimum
- **Changed non-code files (docs, markdown)** — no tests needed
- **Not sure** — run `make test`

### Before Pushing

Always run these before pushing your branch:

```bash
make test     # all tests pass
make build    # TypeScript compiles, Go builds successfully
```

## Pull Request Process

1. **Open a PR against `main`** with a clear title and description.

2. **In your PR description**, include:
   - **What** you changed and **why**
   - How to test the changes (steps, commands, or screenshots)
   - Any related issue numbers (use `Closes #123` to auto-close issues)

3. **CI must pass.** The GitHub Actions pipeline runs:
   - Backend tests (Go + real Postgres)
   - Frontend tests (Vitest)
   - OpenAPI spec bundle freshness check
   - Production build
   - Dependency vulnerability scans

4. **Keep PRs focused.** One logical change per PR. If you find something unrelated to fix, open a separate PR.

5. **Respond to review feedback.** We aim to review PRs promptly. If changes are requested, push new commits (don't force-push over review comments).

6. **Squash or rebase** is fine — we're not prescriptive about merge strategy. The maintainer merging your PR will choose the appropriate method.

### PR Size Guidelines

- **Small PRs get reviewed faster.** Aim for under 400 lines of diff when possible.
- If a feature is large, consider breaking it into smaller, reviewable chunks that each work independently.
- It's fine to open a draft PR early to get feedback on your approach before finishing.

## Commit Messages

Write clear, descriptive commit messages. We follow a simple convention:

```
<type>: <short summary>

<optional body explaining why, not what>
```

**Types:**

| Type | When to use |
|------|-------------|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `docs` | Documentation changes |
| `test` | Adding or updating tests |
| `refactor` | Code restructuring without behavior change |
| `chore` | Build, CI, dependency, or tooling changes |

**Examples:**

```
feat: add Slack channel creation connector

fix: prevent duplicate approval notifications

docs: add connector development guide

test: add edge case coverage for expired invites

refactor: extract approval validation into shared helper

chore: upgrade React Query to v5
```

Keep the summary line under 72 characters. Use the body to explain the **why** behind non-obvious changes.

## Reporting Bugs

Use [GitHub Issues](https://github.com/supersuit-tech/permission-slip-web/issues) to report bugs. A good bug report includes:

- **Summary** — a clear, concise description of the problem
- **Steps to reproduce** — numbered steps someone else can follow
- **Expected behavior** — what you expected to happen
- **Actual behavior** — what actually happened
- **Environment** — OS, browser, Go version, Node version
- **Screenshots or logs** — if applicable

## Requesting Features

Feature requests are welcome! Open an issue with:

- **Problem statement** — what problem does this solve?
- **Proposed solution** — how do you think it should work?
- **Alternatives considered** — other approaches you thought about
- **Additional context** — mockups, examples, or related issues

For large features, we recommend opening an issue to discuss the approach before writing code. This saves time and ensures alignment with the project direction.

## Adding Dependencies

Be thoughtful about new dependencies. Each one adds maintenance burden and potential security surface.

**Go:**

```bash
go get github.com/some/package
```

**Frontend:**

```bash
cd frontend
npm install some-package
```

When adding a dependency, mention it in your PR description with a brief justification for why it's needed and why you chose this package over alternatives.

## Building Connectors

Connectors define what actions agents can perform. If you want to add support for a new service:

- See [Creating Connectors](docs/creating-connectors.md) for the development guide
- See [Custom Connectors](docs/custom-connectors.md) for the external plugin system
- Look at the existing [Slack](connectors/slack/) and [GitHub](connectors/github/) connectors as reference implementations

## Adding Notification Channels

Notification channels self-register via Go's `init()` mechanism — no changes to `main.go` or the approval handler are needed when adding a new channel.

**Steps:**

1. **Implement `notify.Sender`** in a new package, e.g. `notify/slack/`:

   ```go
   // notify/slack/sender.go
   package slack

   type Sender struct { /* ... */ }
   func (s *Sender) Name() string { return "slack" }
   func (s *Sender) Send(ctx context.Context, a notify.Approval, r notify.Recipient) error { /* ... */ }
   ```

2. **Add config fields** to `notify.Config` in `notify/config.go` and load them in `LoadConfig()`.

3. **Create `notify/slack/register.go`** with the self-registration factory:

   ```go
   package slack

   import "github.com/supersuit-tech/permission-slip-web/notify"

   func init() {
       notify.RegisterSenderFactory("slack", func(ctx context.Context, bc notify.BuildContext) ([]notify.Sender, error) {
           if bc.Config.SlackWebhookURL == "" {
               return nil, nil // channel disabled; silently skipped
           }
           return []notify.Sender{New(bc.Config.SlackWebhookURL)}, nil
       })
   }
   ```

4. **Add a blank import** to `notify/all/all.go`:

   ```go
   import _ "github.com/supersuit-tech/permission-slip-web/notify/slack"
   ```

5. **Write tests** in `notify/slack/` and optionally add registry-level tests in `notify/registry_test.go`.

**`SenderFactory` contract:**
- Return `(senders, nil)` when the channel is configured and active.
- Return `(nil, nil)` when the channel is disabled (e.g. env var missing) — silently skipped.
- Return `(nil, err)` for unexpected setup failures — logged, then skipped.
- Never call `log.Fatalf` or `os.Exit` inside a factory.

## Troubleshooting

### `make test-backend` fails with connection errors

Make sure PostgreSQL is running and the test database exists:

```bash
# Check if Postgres is running
pg_isready

# Create the test database if it doesn't exist
make db-setup
```

### `make generate` fails

Ensure frontend dependencies are installed:

```bash
cd frontend && npm install
make generate
```

### `supabase start` hangs

Try with the health check flag:

```bash
supabase start --ignore-health-check
```

Make sure Docker Desktop is running.

### Frontend type errors after pulling changes

Regenerate the API client:

```bash
make generate
```

### Port conflicts

The default ports are:
- **5173** — Vite dev server
- **8080** — Go API server
- **54321** — Supabase API
- **54322** — Supabase Postgres
- **54323** — Supabase Studio

Check for conflicts with `lsof -i :<port>`.

### Tests pass locally but fail in CI

CI runs against a clean environment. Common causes:
- Missing `make generate` step (API types not regenerated)
- Bundled OpenAPI spec is stale — run `make bundle` and commit the result
- Environment-specific test assumptions (file paths, ports)

## Getting Help

- **Issues:** [github.com/supersuit-tech/permission-slip-web/issues](https://github.com/supersuit-tech/permission-slip-web/issues) — ask questions, report problems, or suggest ideas
- **Documentation:** Browse the [docs/](docs/) directory for architecture guides, ADRs, and detailed references
- **Docs site:** Run `npx mintlify dev` from `docs-site/` to preview the user-facing documentation locally

## License

By contributing to Permission Slip, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE). No CLA is required.
