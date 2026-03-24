# Claude Code Instructions

## Interaction Rules

- When asked a question, just answer it. Do not write or modify code unless explicitly asked.
- Always ask for permission before writing any code.
- Do not git push without confirming first
- After making the first commit on a branch, automatically create a pull request using `gh pr create`. Use a concise title and include a descriptive body with a summary and test plan. For subsequent commits, push to the existing PR branch.
- **Link PRs to issues.** When a PR addresses a GitHub issue, include `Closes #<issue>` in the PR body so the issue is automatically closed when the PR merges. Don't wait to be asked.
- **Never open draft PRs.** Always open PRs as ready for review.
- When suggesting a PR body or GitHub issue that includes a checklist, **split the checklist into two sections** based on who should handle each item:
  - `### Developer` — items that a developer can address (e.g., add tests, fix lint, update docs, add error handling, run checks). The `/watch` command will pick these up and check them off.
  - `### Manual Testing` — items that require manual verification, design review, or hands-on testing that a developer can't perform.
- Always include the pull request URL at the end of every message where a PR already exists, formatted exactly as: `Pull request: <url>` — no bold, no markdown link syntax, just the plain text and URL so the link doesn't break.
- Whenever you bring up a problem, always suggest a recommendation for how to address it.
- When asked to review for improvements or issues: fix anything you're confident should be fixed (commit & push), then mention any additional findings that are more subjective or optional so the user can decide.
- **Check all comment types:** When asked to address PR review comments, also check normal PR comments (issue comments) for feedback worth implementing. Reviewers sometimes leave actionable suggestions in the general conversation thread instead of inline review comments — treat them the same way.
- **Train the reviewer:** When you address a PR review comment, you MUST react to it AND leave a threaded reply. Do both of these for EVERY review comment you respond to:
  1. **React** with 👍 (agree/fixed) or 👎 (disagree/won't fix) using the GitHub API:
     ```bash
     # Get the comment ID from the review comment URL or API response
     # React with thumbs up (agreed and fixed):
     GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api repos/supersuit-tech/permission-slip/pulls/comments/{comment_id}/reactions -f content="+1"
     # React with thumbs down (disagree, explain why):
     GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api repos/supersuit-tech/permission-slip/pulls/comments/{comment_id}/reactions -f content="-1"
     ```
  2. **Reply** in the thread explaining what you changed and why (or why you disagree).
  This builds context over time and helps the reviewer calibrate their feedback.
- If a file is getting large enough that splitting it would improve maintainability, just go ahead and split it — don't ask first.
- When you need to ask questions, just ask them in regular chat text. Do NOT use the AskUserQuestion tool — it doesn't work reliably.

## Post-Task Review (before marking complete)

Before presenting work as done, always perform these review passes:

1. **Self-review:** flag concerns, risks, or tradeoffs worth discussing
2. **Senior engineer lens:** would a principal engineer approve this? Any security issues?
3. **Maintainability:** check for DRY violations, oversized files, test race conditions, refactor opportunities
4. **Code quality:** best practices, error handling, edge cases
5. **Documentation:** update comments, README, docs as needed

Do NOT mark a task complete until all passes are done. If any pass surfaces issues, fix them before presenting.

## Minimizing Merge Conflicts

This codebase is worked on in parallel by multiple agents and developers. Write code with that in mind:

### File & Function Hygiene
- **Keep files small and focused.** Large files are merge conflict magnets. If a file is growing, split it proactively — don't wait until it's a problem.
- **Append, don't insert into the middle.** When adding new items to lists, maps, routes, constants, or config arrays, add them at the end rather than alphabetically or in the middle. This avoids conflicts when two branches both insert at the same position.
- **One concern per file.** Two branches are unlikely to conflict if they're editing different files. Colocating unrelated logic in one file increases collision risk.

### Structural Patterns
- **Prefer new files over editing shared files.** When adding a new feature (route handler, component, hook, migration), create a new file and wire it in with a single-line import/registration — rather than inlining everything into an existing file.
- **Use index/registry files for wiring.** If multiple branches need to register routes, components, or middleware, a simple registry pattern (e.g., an array of imports) keeps each addition to a single line, reducing overlap.
- **Avoid reformatting or reordering existing code.** Don't rearrange imports, reorder functions, or reformat code you didn't change. These cosmetic diffs touch many lines and create unnecessary conflicts with other branches.

### Data & Schema
- **Migrations are inherently conflict-prone.** Keep each migration small and scoped to one concern. Never combine unrelated schema changes in a single migration file.
- **Seed data additions should be append-only.** Add new seed entries at the end of insert blocks rather than interleaving with existing data.

### General Practices
- **Keep diffs minimal.** Only touch lines directly related to your task. Resist the urge to fix nearby style issues, rename unrelated variables, or "clean up while you're in there" — save those for a dedicated cleanup PR.
- **Avoid touching shared configuration files unnecessarily.** Files like `package.json`, `go.mod`, `tsconfig.json`, and CI configs are edited by almost every branch. Only modify them when your task genuinely requires it.
- **When you must edit a hot file, make surgical changes.** If you need to add a route to a central router or a column to a shared type, add it in as few lines as possible and avoid reformatting surrounding code.

## Testing

Always run relevant tests after making changes, before committing. Before pushing, run `make build` to catch TypeScript compilation errors that tests alone may miss (e.g. unused variables, missing interface fields).

**Important:** Ensure the Go environment variables from the "Go Toolchain Setup" section are exported before running any `make` or `go` commands. If you get network-related errors from Go (e.g., "CONNECT tunnel failed", "module lookup disabled"), the env vars are not set — fix that first, don't give up.

- **All tests:** `make test`
- **Backend only:** `make test-backend` (runs `go test ./...`)
- **Frontend only:** `make test-frontend` (runs `cd frontend && npm test`)
- **Mobile only:** `make mobile-test` (runs `cd mobile && npm test -- --ci`)
- **Database tests only:** `go test ./db/... -v`

Database tests require a running Postgres instance. They use `DATABASE_URL_TEST` (defaults to `postgres://localhost:5432/permission_slip_test?sslmode=disable`).

### What to run

- **Changed only non-code files (e.g. markdown, docs):** skip tests — no test run needed.
- **Changed Go code:** run `make test-backend`
- **Changed frontend code:** run `make test-frontend`
- **Changed mobile code:** run `make mobile-test`
- **Changed migrations or db/ code:** run `go test ./db/... -v` at minimum
- **Not sure:** run `make test`

## Creating Migrations

**ALWAYS use `make migrate-create` to create new migration files.** Never manually create migration files or invent timestamps — this has caused duplicate timestamp collisions that break goose.

```bash
make migrate-create NAME=add_users_table
```

This generates a real timestamp from `date +%Y%m%d%H%M%S` and creates the file with the correct goose boilerplate. If you need multiple migrations, run the command once for each — the second-level precision ensures unique timestamps when run sequentially.

A test (`TestMigrationTimestampsUnique` in `db/migrations_integrity_test.go`) validates that all migration timestamps are unique and in sorted order. This runs as part of `make test-backend` and will catch duplicates before CI.

**Extension-managed objects need explicit grants.** `ALTER DEFAULT PRIVILEGES` only applies to tables created *after* the grant runs — it never covers tables created by extensions (e.g., `vault.secrets`, pgcrypto functions). Whenever a migration grants access to an extension-managed schema, always add explicit `GRANT ... ON <table> TO app_backend` for every object the app touches, not just the functions and views.

## app_backend Role Permissions

The Go backend runs as the `app_backend` PostgreSQL role (non-superuser). Every database object the backend touches needs an explicit grant. This section documents what's granted and the principles to follow when adding new objects.

### What's Currently Granted

| Schema | Object | Grant | Migration |
|--------|--------|-------|-----------|
| `public` | All existing tables | SELECT, INSERT, UPDATE, DELETE | `20260309003720` |
| `public` | All sequences | USAGE, SELECT | `20260309003720` |
| `public` | Future tables/sequences | ALTER DEFAULT PRIVILEGES | `20260309003720` |
| `public` | All RLS-enabled tables | `app_backend_all` policy (FOR ALL) | `20260310111634` |
| `auth` | Schema | USAGE | `20260310102034` |
| `auth` | `auth.users` | SELECT only | `20260309212920` |
| `vault` | Schema | USAGE | `20260311123718` |
| `vault` | `vault.decrypted_secrets` | SELECT | `20260311123718` |
| `vault` | `vault.secrets` | SELECT, INSERT, UPDATE, DELETE | `20260312023009` |
| `vault` | `vault.create_secret()` | EXECUTE | `20260311123718` |
| `vault` | `vault.update_secret()` | EXECUTE | `20260311123718` |
| `vault` | `vault._crypto_aead_det_decrypt()` | EXECUTE (vault v0.3.0+) | `20260312123238` |
| `pgsodium` | Schema | USAGE | `20260312113621` |
| `pgsodium` | `pgsodium._crypto_aead_det_decrypt()` | EXECUTE (vault v0.2.x) | `20260312113621` |

### When Adding New Grants — Checklist

Follow this checklist whenever a migration introduces a new table, uses a new extension, or accesses a new schema:

1. **New public table?** `ALTER DEFAULT PRIVILEGES` covers it automatically for DML. But you MUST also:
   - Enable RLS: `ALTER TABLE <name> ENABLE ROW LEVEL SECURITY;`
   - Add policy: `CREATE POLICY app_backend_all ON <name> FOR ALL TO app_backend USING (true) WITH CHECK (true);`
   - The RLS test (`TestAppBackendHasPoliciesOnAllRLSTables`) will catch this if you forget.

2. **New extension table/view?** Extensions create objects *before* `ALTER DEFAULT PRIVILEGES` runs, so they are NEVER covered. Add an explicit `GRANT SELECT/INSERT/UPDATE/DELETE ON <schema>.<table> TO app_backend;` for each object the backend touches.

3. **New extension function?** Grant EXECUTE on the exact function signature. Don't grant on ALL FUNCTIONS — follow least-privilege.

4. **New schema?** Grant `USAGE ON SCHEMA <name> TO app_backend;` — without USAGE, no objects in the schema are accessible regardless of object-level grants.

5. **SECURITY INVOKER views?** Views are SECURITY INVOKER by default. If a view calls functions or reads from tables in other schemas, `app_backend` needs grants on **every object the view touches**, not just SELECT on the view itself. This is the root cause of the `vault.decrypted_secrets` + `pgsodium._crypto_aead_det_decrypt` issue — the view runs as the caller (app_backend), so it needs EXECUTE on internal decrypt functions.

6. **SECURITY DEFINER functions?** These run as the function owner (typically superuser). No additional grants needed for objects they access internally. `vault.create_secret()` and `vault.update_secret()` are SECURITY DEFINER — that's why they work without extra grants on pgsodium internals.

7. **Extension upgrades?** Supabase extensions evolve across versions (e.g., vault v0.2.8 → v0.3.0 moved `_crypto_aead_det_decrypt` from pgsodium to vault schema). When granting on extension functions, consider version differences. Use a `BEGIN / EXCEPTION WHEN undefined_function OR invalid_schema_name THEN NULL; END;` block (see item 8) to handle both old and new signatures gracefully — this is tighter than a `pg_proc` name-only check because it validates the exact signature at grant time.

8. **Wrap grants in IF EXISTS guards.** Extension schemas (vault, pgsodium) don't exist in plain Postgres (CI/test). For schema-level or table-level grants, check schema existence: `DO $$ IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = '...') THEN ... END IF; $$`. For function-level grants, use a `BEGIN / EXCEPTION WHEN undefined_function THEN NULL; END;` block to silently skip when the function doesn't exist — this is tighter than a name-only pg_proc check and handles signature changes across extension versions.

## Database Seed Data

Whenever you make changes to database schema, tables, or migrations, review the seed file and update it to reflect the new schema. Add seed data for any new tables or columns so the seed remains comprehensive and stable. The seed should always be runnable against the current schema without errors.

## Storybook ↔ Production Parity

**Storybook stories must always reflect the real component implementation.** Never let stories diverge from production code:

- When updating a component's styling, layout, or behavior, update both the real component AND any corresponding stories in the same commit.
- Stories that duplicate component markup (e.g., `ApprovalDialog.stories.tsx`) must mirror the exact same JSX structure, class names, and design patterns as the real component (`ReviewApprovalDialog.tsx`).
- If you change agent avatars, preview cards, toggles, or any visual element in the real component, apply the same change to all story files that render that element.
- Design concept stories (e.g., `ApprovalDialogConcepts.stories.tsx`) are for exploration only. Once a concept is selected and applied to the real components, delete or archive the unused concept stories.
- **Never introduce new styling in a story without also applying it to the real component**, and vice versa.

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

- **Ask clarifying questions before creating the issue.** If a task has open design decisions, ambiguous requirements, or choices that need human judgment, ask the user directly in chat first. Resolve all decisions upfront — the issue should be actionable from the start with no "Option A vs Option B" or "TBD" items. Never put unresolved questions or option comparisons in an issue.
- When creating issues, default to using checklists (`- [ ]`) instead of bullet points for work items that can be completed independently. This makes it easy to track progress directly in the issue.
- When you encounter an issue with a checklist that is out of date (items completed but not checked off, missing items, irrelevant items), update the checklist to reflect the current state.
- **Never assume issue scope.** If you can't access an issue (e.g., a link doesn't work or returns an error), try fetching it with the `gh` CLI (`GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh issue view <number>`). If you still can't access it, ask the user for the issue details — do not guess or infer what the issue is about.

## Go Toolchain Setup

At the start of each session, set these environment variables for **every shell that runs Go commands**. The sandbox blocks most outbound network access (GCS, sum DB, direct git), so Go must use only the local toolchain and pre-cached modules:

```bash
export GOTOOLCHAIN=local
export GONOSUMDB='*'
export GONOSUMCHECK='*'
export GOPROXY=off
export GOFLAGS='-mod=mod'

# Patch go.mod to match the local Go version (sandbox can't download toolchains)
GO_VERSION=$(go env GOVERSION | sed 's/^go//')
sed -i "s/^go .*/go ${GO_VERSION}/" go.mod
git update-index --assume-unchanged go.mod go.sum

# Fix checksum mismatches for modules cached from GitHub (not the Go proxy)
# The klauspost/compress module was manually cached and has a different zip hash.
sed -i '/klauspost\/compress/d' go.sum

# The root package embeds frontend/dist — create it if missing
mkdir -p frontend/dist && touch frontend/dist/.gitkeep
```

**Why each variable matters:**
- `GOTOOLCHAIN=local` — prevents Go from downloading a newer toolchain (blocked by sandbox)
- `GONOSUMDB='*'` / `GONOSUMCHECK='*'` — skips checksum verification against sum.golang.org (blocked by sandbox)
- `GOPROXY=off` — disables module proxy lookups entirely; all modules must already be in the local cache. The proxy metadata endpoint works but the GCS zip download endpoint is blocked, causing confusing partial failures.
- `GOFLAGS='-mod=mod'` — allows go.sum to be updated when the go.mod version patch changes expected checksums

**Do NOT** say "tests can't run in this environment" — they can and should. If a module is missing from the cache, download it from GitHub and place it in `$GOPATH/pkg/mod/` manually rather than giving up.

## Frontend & Mobile Setup

At the start of each session, install npm dependencies for frontend and mobile (node_modules are not checked in):

```bash
cd frontend && npm install && cd ..
cd mobile && npm install && cd ..
```

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
