-- +goose Up
ALTER TABLE connector_actions ADD COLUMN display_template TEXT;

-- +goose Down
ALTER TABLE connector_actions DROP COLUMN IF EXISTS display_template;
