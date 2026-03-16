-- +goose Up
ALTER TABLE connector_actions ADD COLUMN preview jsonb;

-- +goose Down
ALTER TABLE connector_actions DROP COLUMN IF EXISTS preview;
