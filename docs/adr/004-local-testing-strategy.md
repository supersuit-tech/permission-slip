# ADR-004: Local Testing Strategy — Hybrid Postgres + Mocked Supabase

**Status:** Accepted
**Date:** 2026-02-14
**Authors:** SuperSuit team

## Context

Permission Slip uses Supabase as its hosted Postgres provider and relies on Supabase client libraries for Auth, Storage, and Realtime in the web app (`permission-slip-web`). The Go API (`permission-slip-api`) connects to Postgres directly.

We need a local testing strategy that:

- Lets CI run without a network dependency on hosted Supabase
- Gives developers fast, offline-capable test runs
- Validates database migrations and queries against real Postgres
- Tests code that depends on Supabase Auth, Storage, and Realtime

The Supabase CLI (`supabase start`) provides a full local stack via Docker, but Docker is not available in all CI environments (e.g., sandboxed runners, lightweight containers). We need a strategy that works even when Docker is unavailable.

---

## Decision

Use a **hybrid approach**: real local Postgres for database testing, mocked Supabase client for Auth/Storage/Realtime.

| Layer | Strategy | Rationale |
|---|---|---|
| **Postgres (queries, migrations, RLS)** | Real local Postgres 16 | Catches real SQL issues, validates migrations, tests RLS policies against an actual database engine |
| **Supabase Auth** | Mock `@supabase/supabase-js` auth methods | Auth is HTTP-based; client methods (`getUser`, `signInWithPassword`, `getSession`) are straightforward to stub |
| **Supabase Storage** | Mock storage client | File upload/download calls are simple HTTP wrappers; mock at the client boundary |
| **Supabase Realtime** | Mock subscription handlers | Realtime channels are event-driven; test by emitting synthetic events against subscription callbacks |

---

## Rationale

### Why not the full Supabase local stack everywhere

- **Docker dependency.** `supabase start` requires a running Docker daemon. Many CI environments (sandboxed runners, lightweight containers, cloud-based dev environments) don't provide Docker-in-Docker.
- **Startup overhead.** The full stack (Postgres, GoTrue, PostgREST, Realtime, Storage API, Studio) takes 30–60 seconds to spin up. For unit and integration test suites that run in seconds, this is disproportionate.
- **Complexity.** Managing Docker Compose lifecycle in CI adds failure modes (port conflicts, container cleanup, image pulls) unrelated to the code under test.

### Why real Postgres (not SQLite or mocks)

- **RLS policies are Postgres-specific.** Permission Slip's security model relies on row-level security. SQLite and query mocks can't validate these.
- **Migration fidelity.** Running real migrations against real Postgres catches syntax errors, type mismatches, and extension dependencies (`pgcrypto`, `uuid-ossp`) that SQLite would silently ignore.
- **Postgres 16 is widely available.** `apt install postgresql-16` works on Ubuntu CI runners. No Docker required.

### Why mocking Supabase client services is acceptable

- **`@supabase/supabase-js` is modular.** Auth, Storage, and Realtime are separate sub-clients built on standard HTTP requests (`fetch`). Intercepting or stubbing these is well-supported by Jest, Vitest, and MSW.
- **Auth flows are the Supabase client's concern, not ours.** Permission Slip validates JWTs and session tokens — we test our validation logic against known token fixtures, not against a running GoTrue instance.
- **Storage and Realtime are thin wrappers.** Our application logic around these services is minimal in the MVP. Mocking at the boundary is sufficient coverage.

---

## Implementation

### Go API (`permission-slip-api`)

The API connects to Postgres directly via `database/sql` or a driver like `pgx`. No Supabase client is involved.

```bash
# CI setup
pg_ctlcluster 16 main start
createdb permission_slip_test
go test ./...
```

- Run migrations against `permission_slip_test` before each test suite
- Use `t.Cleanup()` to truncate tables between tests
- Test RLS policies by connecting as different Postgres roles

### React Web App (`permission-slip-web`)

Mock the Supabase client at the module boundary:

```typescript
// __mocks__/supabaseClient.ts
import { vi } from 'vitest';

export const supabase = {
  auth: {
    getUser: vi.fn(),
    getSession: vi.fn(),
    signInWithPassword: vi.fn(),
    signOut: vi.fn(),
    onAuthStateChange: vi.fn(() => ({
      data: { subscription: { unsubscribe: vi.fn() } },
    })),
  },
  from: vi.fn(() => ({
    select: vi.fn().mockReturnThis(),
    insert: vi.fn().mockReturnThis(),
    update: vi.fn().mockReturnThis(),
    delete: vi.fn().mockReturnThis(),
    eq: vi.fn().mockReturnThis(),
    single: vi.fn(),
  })),
  storage: {
    from: vi.fn(() => ({
      upload: vi.fn(),
      download: vi.fn(),
      getPublicUrl: vi.fn(),
    })),
  },
  channel: vi.fn(() => ({
    on: vi.fn().mockReturnThis(),
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
  })),
};
```

Tests set return values per-test:

```typescript
supabase.auth.getUser.mockResolvedValue({
  data: { user: { id: 'user-123', email: 'alice@example.com' } },
  error: null,
});
```

### When to use the full Supabase local stack

The full stack (`supabase start`) is recommended for:

- **Local development** on machines where Docker is available
- **End-to-end tests** that exercise the complete auth flow (sign-up, email confirmation, session management)
- **RLS policy development** that depends on Supabase's `auth.uid()` function

This is opt-in, not required. The CI pipeline must pass without Docker.

---

## Alternatives Considered

### Full Supabase local stack for all testing

Rejected. Docker is not universally available in CI, adds startup latency, and introduces failure modes unrelated to the code under test. Appropriate for local dev and e2e tests, not for the primary CI pipeline.

### SQLite for database tests

Rejected. Permission Slip relies on Postgres-specific features (RLS, `pgcrypto`, `uuid-ossp`, `JSONB` operators). SQLite cannot validate these.

### No local database — mock all queries

Rejected. Query mocks don't catch SQL syntax errors, migration issues, or RLS policy bugs. The cost of running real Postgres in CI is low; the coverage gain is high.

---

## Consequences

- **CI environments must have Postgres 16 available.** This is straightforward on Ubuntu-based runners (`apt install postgresql-16`) and most managed CI services.
- **Two testing modes coexist.** Unit/integration tests run with mocked Supabase client + real Postgres. E2e tests (optional, local-only) can use the full Supabase stack.
- **Mock fidelity is a maintenance concern.** When Supabase client APIs change, mocks must be updated. Pin `@supabase/supabase-js` versions and update mocks alongside dependency bumps.
- **RLS tests that depend on `auth.uid()` require either the full stack or Postgres role-based testing** with `SET LOCAL request.jwt.claims` to simulate Supabase's auth context.
