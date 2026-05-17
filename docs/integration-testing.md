# Integration Testing

Integration tests run against a full local Supabase stack (Postgres with Vault extension, GoTrue auth, Mailpit email). They are **never** run in CI — they are opt-in for local development.

## Prerequisites

1. **Supabase CLI** installed ([docs](https://supabase.com/docs/guides/local-development/cli/getting-started))
2. **Supabase running locally:**
   ```bash
   supabase start
   ```
3. **Migrations applied** (Supabase applies them automatically on start, but if you've added new ones):
   ```bash
   make migrate-up
   ```
4. **`VAULT_SECRET_KEY`** environment variable set in your `.env` or `.env.local`. Supabase reads this from the config (`supabase/config.toml` → `[db.vault] secret_key`).

## Running the suite

### Automatically via `make test-backend`

Integration tests run automatically when Supabase is detected:

```bash
make test-backend   # runs unit tests, then integration tests if Supabase is running
```

If Supabase isn't running, the integration tests are skipped with a message — no error, no failure. This means CI (which doesn't have Supabase) skips them automatically.

### Explicitly via `make test-integration`

To run _only_ the integration tests (fails if Supabase isn't running):

```bash
make test-integration
```

This target:
1. Checks that Supabase is running (health check on port 54321) — **errors if not**
2. Runs all Go tests with the `integration` build tag against `postgresql://postgres:postgres@127.0.0.1:54322/postgres`

You can also run a specific integration test in `api`:

```bash
DATABASE_URL=postgresql://postgres:postgres@127.0.0.1:54322/postgres \
  go test -tags integration -v -run TestIntegration_ ./api/
```

## What's covered

### Credential vault (SQLite unit tests)

Connector OAuth tokens and static credentials are encrypted with **AES-256-GCM** in the `vault_secrets` table. Package **`vault`** includes `TestSQLiteVault_*` tests (round-trip, tampered ciphertext, wrong key) using a temporary SQLite database — these run in normal `go test ./...` / CI.

### Auth tests (`api/auth_integration_test.go`)

| Test | What it verifies |
|------|-----------------|
| `JWKSEndpoint_ReturnsKeys` | JWKS endpoint returns EC P-256 keys |
| `JWKSCache_FetchesRealKeys` | JWKSCache successfully fetches and caches real keys |
| `ES256_JWTValidation_ViaSignup` | Sign up via Supabase Auth, use JWT, backend validates with JWKS |
| `ExpiredJWT_Rejected` | Expired JWTs are rejected |
| `RefreshTokenRotation` | Refresh token returns new access + refresh token pair |
| `UserLifecycle_SignupCreatesAuthUser` | Supabase signup creates `auth.users` row |

## How to add new integration tests

1. **Create or edit a `*_integration_test.go` file** in the appropriate package
2. **Add the build tag** as the first line:
   ```go
   //go:build integration
   ```
3. **Name tests with the `TestIntegration_` prefix** for consistency
4. **Clean up test data** using `t.Cleanup()` — integration tests run against a shared database, not isolated transactions
5. **Use `db.Connect` + `db.Migrate`** for SQLite-backed integration scenarios, or the Supabase helpers below when testing the legacy Docker stack

## Why separate from CI?

- CI runs `go test ./...` including `vault` package tests for AES-256-GCM round-trip, tamper detection, and wrong-key rejection against a temporary SQLite file
- Optional `//go:build integration` tests may still target the Supabase Docker stack for JWT/auth flows (`api/auth_integration_test.go`)
- The Supabase stack requires Docker containers for Postgres + GoTrue + Mailpit
- Keeping heavy integration separate means CI stays fast while developers can verify integrations locally

## Ports reference

| Service | Port |
|---------|------|
| Supabase API | 54321 |
| Postgres | 54322 |
| Studio | 54323 |
| Mailpit (Inbucket) | 54324 |
