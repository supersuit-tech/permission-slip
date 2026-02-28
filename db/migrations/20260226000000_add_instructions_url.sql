-- +goose Up
ALTER TABLE connector_required_credentials
    ADD COLUMN instructions_url text CHECK (char_length(instructions_url) <= 2048);

-- +goose Down
ALTER TABLE connector_required_credentials
    DROP COLUMN IF EXISTS instructions_url;
