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

You can also run a specific test:
```bash
DATABASE_URL=postgresql://postgres:postgres@127.0.0.1:54322/postgres \
  go test -tags integration -v -run TestIntegration_VaultCreateSecret ./api/
```

## What's covered

### Vault tests (`api/vault_integration_test.go`)

| Test | What it verifies |
|------|-----------------|
| `VaultCreateSecret_ReturnsValidUUID` | `vault.create_secret()` returns a valid UUID |
| `VaultDecryptedSecretsView` | `vault.decrypted_secrets` view returns original plaintext |
| `VaultDeleteSecret` | `DELETE FROM vault.secrets` removes the row |
| `VaultStoreAndReadRoundTrip` | Store credential via API, read vault secret, verify JSON match |
| `VaultStoreAndDeleteRoundTrip` | Store credential, delete via API, verify vault secret gone |
| `VaultTransactionAtomicity` | Vault insert rolled back with transaction rollback |
| `GetDecryptedCredentials` | Full decrypt pipeline with real Supabase Vault |
| `VaultSecretEncrypted` | Raw `vault.secrets` column contains ciphertext, not plaintext |

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
5. **Use `setupIntegrationPool(t)`** to get a connection to the Supabase Postgres instance
6. **Use `setupIntegrationUser(t, pool)`** to create a test user with proper auth.users + profiles rows

## Why separate from CI?

- CI runs against plain Postgres (fast, no Docker dependency)
- The Supabase stack requires Docker containers for Postgres + GoTrue + Vault + Mailpit
- Integration tests verify real encryption, real JWT signing, and real auth flows that mocks cannot cover
- Keeping them separate means CI stays fast and reliable while developers can verify integrations locally

## Ports reference

| Service | Port |
|---------|------|
| Supabase API | 54321 |
| Postgres | 54322 |
| Studio | 54323 |
| Mailpit (Inbucket) | 54324 |
