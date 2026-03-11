-- +goose Up

-- The app_backend role needs access to the vault schema to store and
-- retrieve OAuth tokens and user-provided credentials. Without these
-- grants, vault.create_secret() / vault.decrypted_secrets queries fail
-- with a "permission denied for schema vault" error.
--
-- Wrapped in a DO block so the migration is safe on plain Postgres (CI)
-- where the supabase_vault extension is not installed.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'vault') THEN
        GRANT USAGE ON SCHEMA vault TO app_backend;
        GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA vault TO app_backend;
        GRANT SELECT, INSERT, DELETE ON ALL TABLES IN SCHEMA vault TO app_backend;
    END IF;
END
$$;

-- +goose Down

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'vault') THEN
        REVOKE SELECT, INSERT, DELETE ON ALL TABLES IN SCHEMA vault FROM app_backend;
        REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA vault FROM app_backend;
        REVOKE USAGE ON SCHEMA vault FROM app_backend;
    END IF;
END
$$;
