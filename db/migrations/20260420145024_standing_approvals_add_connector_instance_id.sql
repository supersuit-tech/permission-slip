-- +goose Up
-- NULL = type-wide standing approval (legacy); non-null = instance-scoped (Phase 4).

ALTER TABLE standing_approvals
    ADD COLUMN connector_instance_id uuid;

CREATE INDEX idx_standing_approvals_agent_action_status_connector_instance
    ON standing_approvals (agent_id, action_type, status, connector_instance_id);

-- +goose Down

DROP INDEX IF EXISTS idx_standing_approvals_agent_action_status_connector_instance;

ALTER TABLE standing_approvals
    DROP COLUMN connector_instance_id;
