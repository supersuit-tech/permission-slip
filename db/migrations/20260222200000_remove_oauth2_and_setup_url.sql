-- +goose Up

-- Remove oauth2 from the auth_type CHECK constraint.
ALTER TABLE connector_required_credentials
    DROP CONSTRAINT IF EXISTS connector_required_credentials_auth_type_check;

-- Normalize any existing oauth2 auth_type values so the new CHECK constraint will succeed.
UPDATE connector_required_credentials
SET auth_type = 'custom'
WHERE auth_type = 'oauth2';
ALTER TABLE connector_required_credentials
    ADD CONSTRAINT connector_required_credentials_auth_type_check
    CHECK (auth_type IN ('api_key', 'basic', 'custom'));

-- Drop the setup_url column (unused — OAuth support not yet implemented).
ALTER TABLE connector_required_credentials DROP COLUMN IF EXISTS setup_url;

-- +goose Down

-- Re-add setup_url column.
ALTER TABLE connector_required_credentials
    ADD COLUMN setup_url text CHECK (char_length(setup_url) <= 2048);

-- Restore the auth_type CHECK constraint with oauth2.
ALTER TABLE connector_required_credentials
    DROP CONSTRAINT IF EXISTS connector_required_credentials_auth_type_check;
ALTER TABLE connector_required_credentials
    ADD CONSTRAINT connector_required_credentials_auth_type_check
    CHECK (auth_type IN ('api_key', 'oauth2', 'basic', 'custom'));
