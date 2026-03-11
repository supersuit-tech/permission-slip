-- +goose Up

-- The app_backend role needs access to the vault schema to store and
-- retrieve OAuth tokens and user-provided credentials. Without these
-- grants, vault.create_secret() / vault.decrypted_secrets queries fail
-- with a "permission denied for schema vault" error.
--
-- Wrapped in a DO block so the migration is safe on plain Postgres (CI)
-- where the supabase_vault extension is not installed.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'vault') THEN
        GRANT USAGE ON SCHEMA vault TO app_backend;
        GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA vault TO app_backend;
        GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA vault TO app_backend;

        -- Future-proof: grant on objects created by vault extension upgrades.
        ALTER DEFAULT PRIVILEGES IN SCHEMA vault
            GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_backend;
        ALTER DEFAULT PRIVILEGES IN SCHEMA vault
            GRANT EXECUTE ON FUNCTIONS TO app_backend;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'vault') THEN
        ALTER DEFAULT PRIVILEGES IN SCHEMA vault
            REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM app_backend;
        ALTER DEFAULT PRIVILEGES IN SCHEMA vault
            REVOKE EXECUTE ON FUNCTIONS FROM app_backend;
        REVOKE SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA vault FROM app_backend;
        REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA vault FROM app_backend;
        REVOKE USAGE ON SCHEMA vault FROM app_backend;
    END IF;
END
$$;
-- +goose StatementEnd
