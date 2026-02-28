-- +goose Up

-- Track whether executed actions succeeded or failed at the third-party service.
-- Only populated for action.executed and standing_approval.executed events.
ALTER TABLE audit_events
    ADD COLUMN execution_status text CHECK (execution_status IN ('success', 'failure', 'timeout', 'skipped'));

ALTER TABLE audit_events
    ADD COLUMN execution_error text;

-- +goose Down

ALTER TABLE audit_events DROP COLUMN IF EXISTS execution_error;
ALTER TABLE audit_events DROP COLUMN IF EXISTS execution_status;
