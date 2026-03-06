-- +goose Up

-- Re-add oauth2 to the auth_type CHECK constraint (it was removed in
-- 20260222200000 when OAuth wasn't yet implemented).
ALTER TABLE connector_required_credentials
    DROP CONSTRAINT IF EXISTS connector_required_credentials_auth_type_check;
ALTER TABLE connector_required_credentials
    ADD CONSTRAINT connector_required_credentials_auth_type_check
    CHECK (auth_type IN ('api_key', 'basic', 'custom', 'oauth2'));

-- Add OAuth-specific columns for connectors that use oauth2 auth.
ALTER TABLE connector_required_credentials
    ADD COLUMN oauth_provider text CHECK (char_length(oauth_provider) <= 255),
    ADD COLUMN oauth_scopes   text[] DEFAULT '{}';

-- Enforce: oauth_provider is required when auth_type is oauth2,
-- and must be NULL for other auth types.
ALTER TABLE connector_required_credentials
    ADD CONSTRAINT chk_oauth_provider_required
    CHECK (
        (auth_type = 'oauth2' AND oauth_provider IS NOT NULL)
        OR (auth_type != 'oauth2' AND oauth_provider IS NULL)
    );

-- +goose Down

ALTER TABLE connector_required_credentials
    DROP CONSTRAINT IF EXISTS chk_oauth_provider_required;

ALTER TABLE connector_required_credentials
    DROP COLUMN IF EXISTS oauth_scopes,
    DROP COLUMN IF EXISTS oauth_provider;

ALTER TABLE connector_required_credentials
    DROP CONSTRAINT IF EXISTS connector_required_credentials_auth_type_check;
ALTER TABLE connector_required_credentials
    ADD CONSTRAINT connector_required_credentials_auth_type_check
    CHECK (auth_type IN ('api_key', 'basic', 'custom'));
