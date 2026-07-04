-- +goose Up
ALTER TABLE connector_actions ADD COLUMN data_window jsonb;

COMMENT ON COLUMN connector_actions.data_window IS
  'Optional {"start_param","end_param"} pair naming the action parameters used for $data_window enforcement';

-- +goose Down
ALTER TABLE connector_actions DROP COLUMN IF EXISTS data_window;
