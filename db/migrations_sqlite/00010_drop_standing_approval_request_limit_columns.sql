-- +goose Up
-- +goose StatementBegin
-- SQLite cannot drop columns that participate in CHECK constraints via ALTER TABLE;
-- recreate the table without max_executions / expires_in_seconds.
CREATE TABLE standing_approval_requests_new (
    request_id TEXT PRIMARY KEY,
    agent_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    action_version TEXT NOT NULL DEFAULT '1',
    constraints TEXT NOT NULL,
    source_action_configuration_id TEXT,
    connector_name TEXT,
    connector_instance_display TEXT,
    connector_instance_id TEXT,
    status TEXT NOT NULL,
    decided_at TEXT,
    resulting_standing_approval_id TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CONSTRAINT standing_approval_requests_action_type_check CHECK (length(action_type) <= 128),
    CONSTRAINT standing_approval_requests_action_version_check CHECK (length(action_version) <= 10),
    CONSTRAINT standing_approval_requests_constraints_check CHECK (length(constraints) <= 65536),
    CONSTRAINT standing_approval_requests_request_id_check CHECK (length(request_id) <= 255),
    CONSTRAINT standing_approval_requests_status_check CHECK (status IN ('pending', 'approved', 'denied', 'expired', 'cancelled')),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (source_action_configuration_id) REFERENCES action_configurations(id) ON DELETE RESTRICT,
    FOREIGN KEY (resulting_standing_approval_id) REFERENCES standing_approvals(standing_approval_id) ON DELETE SET NULL
);

INSERT INTO standing_approval_requests_new (
    request_id, agent_id, user_id, action_type, action_version, constraints,
    source_action_configuration_id, connector_name, connector_instance_display,
    connector_instance_id, status, decided_at, resulting_standing_approval_id,
    created_at, updated_at
)
SELECT
    request_id, agent_id, user_id, action_type, action_version, constraints,
    source_action_configuration_id, connector_name, connector_instance_display,
    connector_instance_id, status, decided_at, resulting_standing_approval_id,
    created_at, updated_at
FROM standing_approval_requests;

DROP TABLE standing_approval_requests;
ALTER TABLE standing_approval_requests_new RENAME TO standing_approval_requests;

CREATE INDEX idx_standing_approval_requests_user_status_created
    ON standing_approval_requests (user_id, status, created_at DESC, request_id DESC);

CREATE INDEX idx_standing_approval_requests_agent_status
    ON standing_approval_requests (agent_id, status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE standing_approval_requests_old (
    request_id TEXT PRIMARY KEY,
    agent_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    action_version TEXT NOT NULL DEFAULT '1',
    constraints TEXT NOT NULL,
    max_executions INTEGER,
    expires_in_seconds INTEGER,
    source_action_configuration_id TEXT,
    connector_name TEXT,
    connector_instance_display TEXT,
    connector_instance_id TEXT,
    status TEXT NOT NULL,
    decided_at TEXT,
    resulting_standing_approval_id TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CONSTRAINT standing_approval_requests_action_type_check CHECK (length(action_type) <= 128),
    CONSTRAINT standing_approval_requests_action_version_check CHECK (length(action_version) <= 10),
    CONSTRAINT standing_approval_requests_constraints_check CHECK (length(constraints) <= 65536),
    CONSTRAINT standing_approval_requests_max_executions_check CHECK (max_executions IS NULL OR max_executions > 0),
    CONSTRAINT standing_approval_requests_expires_in_seconds_check CHECK (expires_in_seconds IS NULL OR expires_in_seconds > 0),
    CONSTRAINT standing_approval_requests_request_id_check CHECK (length(request_id) <= 255),
    CONSTRAINT standing_approval_requests_status_check CHECK (status IN ('pending', 'approved', 'denied', 'expired', 'cancelled')),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (source_action_configuration_id) REFERENCES action_configurations(id) ON DELETE RESTRICT,
    FOREIGN KEY (resulting_standing_approval_id) REFERENCES standing_approvals(standing_approval_id) ON DELETE SET NULL
);

INSERT INTO standing_approval_requests_old (
    request_id, agent_id, user_id, action_type, action_version, constraints,
    max_executions, expires_in_seconds,
    source_action_configuration_id, connector_name, connector_instance_display,
    connector_instance_id, status, decided_at, resulting_standing_approval_id,
    created_at, updated_at
)
SELECT
    request_id, agent_id, user_id, action_type, action_version, constraints,
    NULL, NULL,
    source_action_configuration_id, connector_name, connector_instance_display,
    connector_instance_id, status, decided_at, resulting_standing_approval_id,
    created_at, updated_at
FROM standing_approval_requests;

DROP TABLE standing_approval_requests;
ALTER TABLE standing_approval_requests_old RENAME TO standing_approval_requests;

CREATE INDEX idx_standing_approval_requests_user_status_created
    ON standing_approval_requests (user_id, status, created_at DESC, request_id DESC);

CREATE INDEX idx_standing_approval_requests_agent_status
    ON standing_approval_requests (agent_id, status);
-- +goose StatementEnd
