-- +goose Up
ALTER TABLE standing_approval_requests
    ADD COLUMN connector_name text,
    ADD COLUMN connector_instance_display text;

-- +goose Down
ALTER TABLE standing_approval_requests
    DROP COLUMN IF EXISTS connector_instance_display,
    DROP COLUMN IF EXISTS connector_name;
