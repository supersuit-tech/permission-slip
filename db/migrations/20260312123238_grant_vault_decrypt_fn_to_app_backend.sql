-- +goose Up

-- Vault v0.3.0 moved the decrypt function from pgsodium into the vault schema:
--   vault._crypto_aead_det_decrypt(bytea, bytea, bigint, bytea, bytea)
-- The decrypted_secrets view (SECURITY INVOKER) calls this function, so
-- app_backend needs EXECUTE on it. Without this grant, reading decrypted
-- secrets fails with "permission denied for function _crypto_aead_det_decrypt".
--
-- The prior migration (20260312113621) granted pgsodium._crypto_aead_det_decrypt
-- which only covers vault v0.2.x where the view calls pgsodium directly.
-- This migration covers v0.3.0+ where the view calls a vault-schema wrapper.
--
-- Wrapped in a DO block so the migration is safe when the function doesn't
-- exist (older vault version or plain Postgres without supabase_vault).

-- +goose StatementBegin
DO $$
BEGIN
    -- Vault v0.3.0+ wrapper function used by vault.decrypted_secrets view.
    IF EXISTS (
        SELECT 1 FROM pg_proc p
        JOIN pg_namespace n ON p.pronamespace = n.oid
        WHERE n.nspname = 'vault'
          AND p.proname = '_crypto_aead_det_decrypt'
    ) THEN
        GRANT EXECUTE ON FUNCTION vault._crypto_aead_det_decrypt(bytea, bytea, bigint, bytea, bytea) TO app_backend;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_proc p
        JOIN pg_namespace n ON p.pronamespace = n.oid
        WHERE n.nspname = 'vault'
          AND p.proname = '_crypto_aead_det_decrypt'
    ) THEN
        REVOKE EXECUTE ON FUNCTION vault._crypto_aead_det_decrypt(bytea, bytea, bigint, bytea, bytea) FROM app_backend;
    END IF;
END
$$;
-- +goose StatementEnd
