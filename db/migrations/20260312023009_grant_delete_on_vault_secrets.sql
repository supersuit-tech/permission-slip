-- +goose Up

-- The vault.secrets table was created by the supabase_vault extension BEFORE
-- the ALTER DEFAULT PRIVILEGES in 20260311123718 ran, so app_backend never
-- received DML grants on the existing table. This caused DELETE operations
-- (e.g. disconnecting an OAuth provider) to fail with "permission denied".
--
-- Grant SELECT, INSERT, UPDATE, DELETE explicitly on the existing table so
-- that vault operations work for both reads and writes.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'vault' AND table_name = 'secrets'
    ) THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON vault.secrets TO app_backend;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down

-- Only revoke the privileges this migration added. SELECT, INSERT, UPDATE
-- may have been granted through other mechanisms (vault functions, prior
-- migrations) and revoking them here would leave the app in a broken state.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'vault' AND table_name = 'secrets'
    ) THEN
        REVOKE DELETE ON vault.secrets FROM app_backend;
    END IF;
END
$$;
-- +goose StatementEnd
