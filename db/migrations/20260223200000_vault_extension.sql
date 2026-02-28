-- +goose Up
-- Enable Supabase Vault extension for encrypted credential storage.
-- This extension is available in Supabase environments (local dev via
-- `supabase start` and hosted projects). It is NOT available in plain
-- Postgres (CI/test) — tests use MockVaultStore instead.
--
-- The DO block checks whether the extension is available before attempting
-- to create it, so this migration is a no-op on plain Postgres.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_available_extensions WHERE name = 'supabase_vault'
    ) THEN
        CREATE EXTENSION IF NOT EXISTS supabase_vault;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- Don't drop the extension on down-migration — it may contain secrets
-- from other applications sharing the same Supabase instance.
