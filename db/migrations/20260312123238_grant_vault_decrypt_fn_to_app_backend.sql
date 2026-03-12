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
-- Uses BEGIN/EXCEPTION to gracefully skip when the function doesn't exist
-- (older vault version or plain Postgres without supabase_vault).

-- +goose StatementBegin
DO $$
BEGIN
    -- Vault v0.3.0+ wrapper function used by vault.decrypted_secrets view.
    GRANT EXECUTE ON FUNCTION vault._crypto_aead_det_decrypt(bytea, bytea, bigint, bytea, bytea) TO app_backend;
EXCEPTION WHEN undefined_function OR invalid_schema_name THEN
    NULL; -- vault v0.3.0 not installed; skip
END
$$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DO $$
BEGIN
    REVOKE EXECUTE ON FUNCTION vault._crypto_aead_det_decrypt(bytea, bytea, bigint, bytea, bytea) FROM app_backend;
EXCEPTION WHEN undefined_function OR invalid_schema_name THEN
    NULL; -- vault v0.3.0 not installed; skip
END
$$;
-- +goose StatementEnd
