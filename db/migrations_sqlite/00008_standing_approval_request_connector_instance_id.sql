-- +goose Up
ALTER TABLE standing_approval_requests ADD COLUMN connector_instance_id TEXT;

-- +goose Down
ALTER TABLE standing_approval_requests DROP COLUMN connector_instance_id;
