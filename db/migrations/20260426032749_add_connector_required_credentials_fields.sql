-- +goose Up
-- +goose StatementBegin
ALTER TABLE connector_required_credentials
    ADD COLUMN IF NOT EXISTS credential_fields JSONB NOT NULL DEFAULT '[]'::jsonb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE connector_required_credentials
    DROP COLUMN IF EXISTS credential_fields;
-- +goose StatementEnd
