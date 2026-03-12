-- +goose Up

-- The vault.decrypted_secrets view is SECURITY INVOKER (default) and calls
-- pgsodium._crypto_aead_det_decrypt() internally. Without EXECUTE on pgsodium
-- functions, app_backend gets "permission denied for function
-- _crypto_aead_det_decrypt (SQLSTATE 42501)" when reading decrypted secrets.
--
-- The prior migration (20260311123718) skipped this grant with a comment that
-- these functions "cannot be granted by the postgres role". That is only true
-- on Supabase's managed cloud platform; on self-hosted Postgres, the migration
-- role is superuser and can grant freely.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'pgsodium') THEN
        GRANT USAGE ON SCHEMA pgsodium TO app_backend;
        -- Only grant the specific decrypt function used by vault.decrypted_secrets.
        -- Granting ALL FUNCTIONS would expose key-management and raw crypto
        -- operations that app_backend has no business calling.
        GRANT EXECUTE ON FUNCTION pgsodium._crypto_aead_det_decrypt(bytea, bytea, bytea, bytea) TO app_backend;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'pgsodium') THEN
        REVOKE EXECUTE ON FUNCTION pgsodium._crypto_aead_det_decrypt(bytea, bytea, bytea, bytea) FROM app_backend;
        REVOKE USAGE ON SCHEMA pgsodium FROM app_backend;
    END IF;
END
$$;
-- +goose StatementEnd
