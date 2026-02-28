# ADR-008: Credential Vault Testing Strategy — VaultStore Interface with Mock

**Status:** Accepted
**Date:** 2026-02-23
**Authors:** SuperSuit team

## Context

Issue #209 introduces Supabase Vault integration for encrypting credential secrets at rest. The Vault is a Postgres extension (`supabase_vault`) that provides server-side AES-256-GCM encryption, available in Supabase environments (local dev via `supabase start` and hosted projects) but **not in plain Postgres** used by CI.

This extends the hybrid testing strategy established in [ADR-004](004-local-testing-strategy.md) to cover the Vault.

---

## Decision

Define a `VaultStore` interface in Go with three methods (`CreateSecret`, `ReadSecret`, `DeleteSecret`). Provide two implementations:

| Implementation | Environment | Behavior |
|---|---|---|
| `SupabaseVaultStore` | Production, local dev (with `supabase start`) | Calls `vault.create_secret()`, reads from `vault.decrypted_secrets`, deletes from `vault.secrets` — all via SQL within the caller's transaction |
| `MockVaultStore` | CI, unit tests | In-memory `map[string][]byte`, no encryption, generates random UUIDs |

### Interface Design

```go
type VaultStore interface {
    CreateSecret(ctx context.Context, tx db.DBTX, name string, secret []byte) (string, error)
    ReadSecret(ctx context.Context, tx db.DBTX, secretID string) ([]byte, error)
    DeleteSecret(ctx context.Context, tx db.DBTX, secretID string) error
}
```

All methods accept `db.DBTX` so vault operations participate in the same database transaction as credential row operations, ensuring atomicity.

### Injection

`VaultStore` is injected via the `api.Deps` struct, following the existing dependency injection pattern. Tests create a `MockVaultStore` per test; production creates a `SupabaseVaultStore` at startup.

---

## Rationale

### Why an interface (not conditional SQL)

- **Testability.** The mock is trivial — no SQL, no extension dependency. Tests run fast and deterministic.
- **Separation of concerns.** The credential handler doesn't know whether secrets are encrypted or stored in memory. It just calls `CreateSecret` / `DeleteSecret`.
- **Same pattern as ADR-004.** Real Postgres for schema/query testing, mocked Supabase service at the boundary.

### Why the mock stores in-memory (not a second Postgres table)

- The `supabase_vault` extension is not available in plain Postgres 16. There's no way to create a `vault.secrets` table or call `vault.create_secret()` without the extension.
- A "fake vault table" approach would require maintaining a parallel schema, adding complexity without meaningful coverage.
- The mock validates the contract (interface compliance, error handling, data flow) — the actual encryption is Supabase's responsibility and is tested in integration tests with the full stack.

### Why vault operations share the caller's transaction

- Atomicity: if the credential row insert fails (e.g., unique constraint violation), the vault secret is automatically rolled back — no orphaned secrets.
- This is a key advantage of Supabase Vault over application-level encryption with an external KMS: the secret lives in the same Postgres instance, so it can participate in ACID transactions.

---

## Consequences

- **CI does not test actual encryption.** The mock stores plaintext. To verify encryption works, run integration tests locally with `supabase start`.
- **MockVaultStore must never be used in production.** The application startup initializes `SupabaseVaultStore` when `DATABASE_URL` is set. The mock is only instantiated in test code.
- **The migration conditionally enables the extension.** The `20260223200000_vault_extension.sql` migration uses a `DO` block to check if `supabase_vault` is available before creating it, so CI migrations don't fail.
- **Seed data continues to use placeholder UUIDs.** Seed data inserts static `vault_secret_id` values that don't correspond to real vault secrets — this is fine because seed data is for UI development, not vault testing.
