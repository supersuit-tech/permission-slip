-- +goose Up
ALTER TABLE standing_approval_requests ADD COLUMN connector_name TEXT;
ALTER TABLE standing_approval_requests ADD COLUMN connector_instance_display TEXT;

-- +goose Down
ALTER TABLE standing_approval_requests DROP COLUMN connector_instance_display;
ALTER TABLE standing_approval_requests DROP COLUMN connector_name;
