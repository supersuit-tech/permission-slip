# Claude Code Instructions

## Interaction Rules

- When asked a question, just answer it. Do not write or modify code unless explicitly asked.
- Always ask for permission before writing any code.
- Do not git push without confirming first
- Never open a pull request without confirming with the user right before doing so to get their explicit approval.
- Whenever you bring up a problem, always suggest a recommendation for how to address it.
- When asked to review for improvements or issues: fix anything you're confident should be fixed (commit & push), then mention any additional findings that are more subjective or optional so the user can decide.
- If a file is getting large enough that splitting it would improve maintainability, just go ahead and split it — don't ask first.
- When you need to ask questions, just ask them in regular chat text. Do NOT use the AskUserQuestion tool — it doesn't work reliably.

## Webhook Notification

Whenever you finish a task or need user input and the user hasn't responded within 2 minutes, trigger the webhook to notify them. This uses the same workflow as the `/watch` command but without a PR URL:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh workflow run trigger-webhook.yml
```

Do not include `-f pr_url=...` — this is a general "attention needed" ping, not a PR-specific notification.

## Post-Task Review (before marking complete)

Before presenting work as done, always perform these review passes:

1. **Self-review:** flag concerns, risks, or tradeoffs worth discussing
2. **Senior engineer lens:** would a principal engineer approve this? Any security issues?
3. **Maintainability:** check for DRY violations, oversized files, test race conditions, refactor opportunities
4. **Code quality:** best practices, error handling, edge cases
5. **Documentation:** update comments, README, docs as needed

Do NOT mark a task complete until all passes are done. If any pass surfaces issues, fix them before presenting.

## Risk Assessment

After completing any code change, provide a risk score from 0 (low) to 10 (high) across three dimensions:

- **Maintainability:** How much does this change affect long-term code health? (complexity added, readability impact, coupling, DRY violations)
- **Security:** Does this change introduce or touch attack surface? (auth, input handling, data exposure, dependencies)
- **Functionality:** How likely is this change to break existing behavior? (scope of change, test coverage, edge cases, integration points)

Present the scores in this format:

```
Risk: Maintainability 2 | Security 1 | Functionality 3
```

Include a one-line explanation if any score is 5 or above.

## Testing

Always run relevant tests after making changes, before committing. Before pushing, run `make build` to catch TypeScript compilation errors that tests alone may miss (e.g. unused variables, missing interface fields).

- **All tests:** `make test`
- **Backend only:** `make test-backend` (runs `go test ./...`)
- **Frontend only:** `make test-frontend` (runs `cd frontend && npm test`)
- **Database tests only:** `go test ./db/... -v`

Database tests require a running Postgres instance. They use `DATABASE_URL_TEST` (defaults to `postgres://localhost:5432/permission_slip_test?sslmode=disable`).

### What to run

- **Changed only non-code files (e.g. markdown, docs):** skip tests — no test run needed.
- **Changed Go code:** run `make test-backend`
- **Changed frontend code:** run `make test-frontend`
- **Changed migrations or db/ code:** run `go test ./db/... -v` at minimum
- **Not sure:** run `make test`

## Creating Migrations

**ALWAYS use `make migrate-create` to create new migration files.** Never manually create migration files or invent timestamps — this has caused duplicate timestamp collisions that break goose.

```bash
make migrate-create NAME=add_users_table
```

This generates a real timestamp from `date +%Y%m%d%H%M%S` and creates the file with the correct goose boilerplate. If you need multiple migrations, run the command once for each — the second-level precision ensures unique timestamps when run sequentially.

A test (`TestMigrationTimestampsUnique` in `db/migrations_integrity_test.go`) validates that all migration timestamps are unique and in sorted order. This runs as part of `make test-backend` and will catch duplicates before CI.

## Database Seed Data

Whenever you make changes to database schema, tables, or migrations, review the seed file and update it to reflect the new schema. Add seed data for any new tables or columns so the seed remains comprehensive and stable. The seed should always be runnable against the current schema without errors.

## React & Frontend Guidelines

### File Structure & Component Organization

- **Pages go in `frontend/src/pages/<feature>/`.** A page composes feature-specific components that live alongside it. Shared/reusable components go in `frontend/src/components/`.
- **One component per file.** If a section of JSX is distinct enough to need a comment explaining what it is, it should be its own component.
- **Break up files into composable components.** Don't inline large blocks of JSX with code comments labeling sections — extract them into named components instead. Each component should have a single clear responsibility.
- **Keep App.tsx thin.** It should handle routing and auth gating, not layout or feature UI. Layout lives in `AppLayout`, page content lives in page components.
- Props interfaces live in the same file as the component unless shared across multiple components.

### Component Architecture

- Components are either **presentational** (receives props, renders UI) or **container** (manages data fetching/state, passes to presentational). Don't mix concerns in a single component.
- Keep business logic out of components — extract to custom hooks or utility functions.
- All API calls go through custom hooks (e.g., `useGetUsers()`) — components never call the API client directly.

### API Layer

The OpenAPI spec (`spec/openapi/`) is the single source of truth for all API types and endpoints. The typed client lives in `frontend/src/api/client.ts` (using `openapi-fetch`), and types are generated via `openapi-typescript`.

- **Never hand-write API types or request functions.** Always generate them from the OpenAPI spec (`npm run generate:api` from `frontend/`). If a type doesn't exist in the generated output, the spec needs updating — not a manual type.
- Import all request/response types from the generated client (`api/schema.d.ts`).
- New backend API calls must use the `openapi-fetch` client instance in `api/client.ts` — never raw `fetch` for our API. (Existing hooks like `useProfile` still use raw `fetch` until the bundled spec includes their endpoints.) Dev-only utilities hitting external services (e.g., Mailpit) are exempt.
- Wrap API calls in custom hooks that handle loading, error, and empty states.

### Type Safety

- No `any`. No `as` casts unless accompanied by a comment explaining why the cast is safe.
- API response data must be validated or type-narrowed before use in components.
- `noUncheckedIndexedAccess` is enabled in tsconfig — use `.charAt()`, nullish coalescing, or type narrowing for indexed access.

### State & Data Fetching

- Use React Query (TanStack Query) for all server state. No `useEffect` + `useState` for API calls.
- Cache keys must mirror the API endpoint structure for predictable invalidation.
- Optimistic updates must include rollback logic.

### Error Handling

- Every API hook must handle loading, error, and empty states explicitly — never assume data is present.
- Use error boundaries at the route level at minimum.
- API errors should be typed (generated from spec error schemas when available).

## Documentation

- Keep README.md updated when making changes that affect setup, usage, or project structure.
- When a section of the README grows large enough to warrant its own page, split it into a separate markdown file in `docs/` and link to it from the README.


## GitHub Issues

- When creating issues, default to using checklists (`- [ ]`) instead of bullet points for work items that can be completed independently. This makes it easy to track progress directly in the issue.
- When you encounter an issue with a checklist that is out of date (items completed but not checked off, missing items, irrelevant items), update the checklist to reflect the current state.

## PostgreSQL Setup

At the start of each session, set up PostgreSQL for database tests:

```bash
# Start PostgreSQL
sudo pg_ctlcluster 16 main start

# Create the root role (this environment runs as root)
sudo -u postgres psql -c "CREATE ROLE root WITH LOGIN SUPERUSER;" 2>/dev/null || true

# Set local auth to trust (avoids password issues)
sudo sed -i 's/^local\s\+all\s\+all\s\+peer/local all all trust/' /etc/postgresql/16/main/pg_hba.conf
sudo sed -i 's/^host\s\+all\s\+all\s\+127.0.0.1\/32\s\+scram-sha-256/host all all 127.0.0.1\/32 trust/' /etc/postgresql/16/main/pg_hba.conf
sudo pg_ctlcluster 16 main reload

# Create dev and test databases
make db-setup
```

## GitHub CLI (gh) Setup

At the start of each session, install the gh CLI if it's not already available:

```bash
# Check if gh is installed
if ! command -v gh &> /dev/null; then
  curl -sL "https://github.com/cli/cli/releases/download/v2.63.2/gh_2.63.2_linux_amd64.tar.gz" -o /tmp/gh.tar.gz
  tar -xzf /tmp/gh.tar.gz -C /tmp
  sudo cp /tmp/gh_2.63.2_linux_amd64/bin/gh /usr/local/bin/gh
fi
```

When using gh, the local git remote uses a proxy, so always set the repo explicitly:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip-web gh <command>
```
