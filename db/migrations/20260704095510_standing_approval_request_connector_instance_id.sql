-- +goose Up
ALTER TABLE standing_approval_requests
    ADD COLUMN connector_instance_id text REFERENCES agent_connector_instances(connector_instance_id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE standing_approval_requests
    DROP COLUMN IF EXISTS connector_instance_id;
