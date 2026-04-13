-- +goose Up

ALTER TABLE connector_actions
  ADD COLUMN operation_type text NOT NULL DEFAULT 'write'
    CHECK (operation_type IN ('read', 'write', 'delete'));

COMMENT ON COLUMN connector_actions.operation_type IS
  'Whether the action is a read, write, or delete for UI grouping. Synced from connector manifests on upsert; default write applies until the connector is re-seeded.';

-- +goose Down

ALTER TABLE connector_actions DROP COLUMN IF EXISTS operation_type;
