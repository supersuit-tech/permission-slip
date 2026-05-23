-- +goose Up
-- Agent-proposed standing approval rules awaiting human approval.

CREATE TABLE standing_approval_requests (
    request_id                      text        PRIMARY KEY CHECK (char_length(request_id) <= 255),
    agent_id                        bigint      NOT NULL REFERENCES agents(agent_id) ON DELETE CASCADE,
    user_id                         uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    action_type                     text        NOT NULL CHECK (char_length(action_type) <= 128),
    action_version                  text        NOT NULL DEFAULT '1' CHECK (char_length(action_version) <= 10),
    constraints                     jsonb       NOT NULL CHECK (pg_column_size(constraints) <= 65536),
    max_executions                  int         CHECK (max_executions IS NULL OR max_executions > 0),
    expires_in_seconds              int         CHECK (expires_in_seconds IS NULL OR expires_in_seconds > 0),
    source_action_configuration_id  text        REFERENCES action_configurations(id) ON DELETE RESTRICT,
    status                          text        NOT NULL CHECK (status IN ('pending', 'approved', 'denied', 'expired', 'cancelled')),
    decided_at                      timestamptz,
    resulting_standing_approval_id  text        REFERENCES standing_approvals(standing_approval_id) ON DELETE SET NULL,
    created_at                      timestamptz NOT NULL DEFAULT now(),
    updated_at                      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_standing_approval_requests_user_status_created
    ON standing_approval_requests (user_id, status, created_at DESC, request_id DESC);

CREATE INDEX idx_standing_approval_requests_agent_status
    ON standing_approval_requests (agent_id, status);

ALTER TABLE standing_approval_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY app_backend_all ON standing_approval_requests FOR ALL TO app_backend USING (true) WITH CHECK (true);

-- +goose Down
DROP TABLE IF EXISTS standing_approval_requests;
