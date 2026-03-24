-- +goose Up

-- Purge orphaned vault secrets left behind when oauth_provider_configs was
-- dropped (migration 20260323235640). The table stored client_id_vault_id and
-- client_secret_vault_id references into vault.secrets, but the DROP TABLE did
-- not clean them up.
--
-- Strategy: delete vault secrets whose IDs are not referenced by any current
-- table (credentials.vault_secret_id, oauth_connections.access_token_vault_id,
-- oauth_connections.refresh_token_vault_id).

-- +goose StatementBegin
DO $$
BEGIN
    -- Only run if vault extension is installed (not present in plain Postgres / CI).
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'vault') THEN
        DELETE FROM vault.secrets
        WHERE id NOT IN (
            SELECT vault_secret_id FROM credentials
            UNION ALL
            SELECT access_token_vault_id FROM oauth_connections
            UNION ALL
            SELECT refresh_token_vault_id FROM oauth_connections
                WHERE refresh_token_vault_id IS NOT NULL
        );
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Cannot restore deleted secrets; this is a one-way cleanup.
