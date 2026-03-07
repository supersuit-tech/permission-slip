-- +goose Up

-- Store provider-specific data from the OAuth token response (e.g.
-- Salesforce's instance_url). Kept as JSONB so different providers can
-- store different fields without needing per-provider columns.
ALTER TABLE oauth_connections ADD COLUMN extra_data jsonb;

-- +goose Down
ALTER TABLE oauth_connections DROP COLUMN IF EXISTS extra_data;
