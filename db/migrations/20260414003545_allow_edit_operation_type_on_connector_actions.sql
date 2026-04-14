-- +goose Up

ALTER TABLE connector_actions DROP CONSTRAINT IF EXISTS connector_actions_operation_type_check;

ALTER TABLE connector_actions
  ADD CONSTRAINT connector_actions_operation_type_check
  CHECK (operation_type IN ('read', 'write', 'edit', 'delete'));

COMMENT ON COLUMN connector_actions.operation_type IS
  'Whether the action is read, write, edit, or delete for UI grouping. Synced from connector manifests on upsert; default write applies until the connector is re-seeded.';

-- +goose Down

ALTER TABLE connector_actions DROP CONSTRAINT IF EXISTS connector_actions_operation_type_check;

-- Demote any existing 'edit' rows to 'write' before re-adding the narrower
-- constraint, otherwise ADD CONSTRAINT would fail on any rows created while
-- the Up migration was live.
UPDATE connector_actions SET operation_type = 'write' WHERE operation_type = 'edit';

ALTER TABLE connector_actions
  ADD CONSTRAINT connector_actions_operation_type_check
  CHECK (operation_type IN ('read', 'write', 'delete'));

COMMENT ON COLUMN connector_actions.operation_type IS
  'Whether the action is a read, write, or delete for UI grouping. Synced from connector manifests on upsert; default write applies until the connector is re-seeded.';
