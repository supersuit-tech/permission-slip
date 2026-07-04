-- +goose Up
ALTER TABLE connector_actions ADD COLUMN data_window TEXT;

-- +goose Down
ALTER TABLE connector_actions DROP COLUMN data_window;
