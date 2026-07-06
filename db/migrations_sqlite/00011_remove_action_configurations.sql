-- +goose Up
-- +goose StatementBegin

-- 1. Add name/description and drop source_action_configuration_id from standing_approvals.
CREATE TABLE standing_approvals_new (
    standing_approval_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    action_version TEXT NOT NULL DEFAULT '1',
    constraints TEXT,
    status TEXT NOT NULL,
    starts_at TEXT NOT NULL,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    revoked_at TEXT,
    expired_at TEXT,
    agent_id INTEGER NOT NULL,
    connector_instance_id TEXT,
    name TEXT,
    description TEXT,
    CONSTRAINT standing_approvals_action_type_check CHECK (length(action_type) <= 128),
    CONSTRAINT standing_approvals_action_version_check CHECK (length(action_version) <= 10),
    CONSTRAINT standing_approvals_constraints_check CHECK (length(constraints) <= 65536),
    CONSTRAINT standing_approvals_expires_at_after_starts_at CHECK (expires_at IS NULL OR expires_at >= starts_at),
    CONSTRAINT standing_approvals_standing_approval_id_check CHECK (length(standing_approval_id) <= 255),
    CONSTRAINT standing_approvals_status_check CHECK (status IN ('active', 'expired', 'revoked')),
    CONSTRAINT standing_approvals_name_check CHECK (name IS NULL OR length(name) <= 255),
    CONSTRAINT standing_approvals_description_check CHECK (description IS NULL OR length(description) <= 4096),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);

INSERT INTO standing_approvals_new (
    standing_approval_id, user_id, action_type, action_version, constraints,
    status, starts_at, expires_at, created_at, revoked_at, expired_at,
    agent_id, connector_instance_id, name, description
)
SELECT
    sa.standing_approval_id, sa.user_id, sa.action_type, sa.action_version, sa.constraints,
    sa.status, sa.starts_at, sa.expires_at, sa.created_at, sa.revoked_at, sa.expired_at,
    sa.agent_id, sa.connector_instance_id, ac.name, ac.description
FROM standing_approvals sa
LEFT JOIN action_configurations ac ON ac.id = sa.source_action_configuration_id;

DROP TABLE standing_approvals;
ALTER TABLE standing_approvals_new RENAME TO standing_approvals;

CREATE INDEX idx_standing_approvals_agent_action_status ON standing_approvals(agent_id, action_type, status);
CREATE INDEX idx_standing_approvals_agent_action_status_connector_instance ON standing_approvals(agent_id, action_type, status, connector_instance_id);
CREATE INDEX idx_standing_approvals_agent_status_created ON standing_approvals(agent_id, status, created_at DESC, standing_approval_id DESC);
CREATE INDEX idx_standing_approvals_user_active ON standing_approvals(user_id) WHERE status = 'active';

-- 2. Drop source_action_configuration_id from standing_approval_requests.
CREATE TABLE standing_approval_requests_new (
    request_id TEXT PRIMARY KEY,
    agent_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    action_version TEXT NOT NULL DEFAULT '1',
    constraints TEXT NOT NULL,
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
    FOREIGN KEY (resulting_standing_approval_id) REFERENCES standing_approvals(standing_approval_id) ON DELETE SET NULL
);

INSERT INTO standing_approval_requests_new (
    request_id, agent_id, user_id, action_type, action_version, constraints,
    connector_name, connector_instance_display, connector_instance_id,
    status, decided_at, resulting_standing_approval_id, created_at, updated_at
)
SELECT
    request_id, agent_id, user_id, action_type, action_version, constraints,
    connector_name, connector_instance_display, connector_instance_id,
    status, decided_at, resulting_standing_approval_id, created_at, updated_at
FROM standing_approval_requests;

DROP TABLE standing_approval_requests;
ALTER TABLE standing_approval_requests_new RENAME TO standing_approval_requests;

CREATE INDEX idx_standing_approval_requests_user_status_created
    ON standing_approval_requests (user_id, status, created_at DESC, request_id DESC);
CREATE INDEX idx_standing_approval_requests_agent_status
    ON standing_approval_requests (agent_id, status);

-- 3. Create standing_approval_templates.
CREATE TABLE standing_approval_templates (
    id TEXT PRIMARY KEY,
    connector_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    constraints TEXT NOT NULL DEFAULT '{}',
    duration_days INTEGER,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CONSTRAINT standing_approval_templates_id_check CHECK (length(id) <= 255),
    CONSTRAINT standing_approval_templates_connector_id_check CHECK (length(connector_id) <= 255),
    CONSTRAINT standing_approval_templates_action_type_check CHECK (length(action_type) <= 255),
    CONSTRAINT standing_approval_templates_name_check CHECK (length(name) <= 255),
    CONSTRAINT standing_approval_templates_description_check CHECK (description IS NULL OR length(description) <= 4096),
    CONSTRAINT standing_approval_templates_constraints_check CHECK (json_type(constraints) = 'object'),
    CONSTRAINT standing_approval_templates_constraints_check1 CHECK (length(constraints) <= 65536),
    CONSTRAINT standing_approval_templates_duration_days_check CHECK (duration_days IS NULL OR duration_days > 0),
    FOREIGN KEY (connector_id) REFERENCES connectors(id) ON DELETE CASCADE,
    FOREIGN KEY (connector_id, action_type) REFERENCES connector_actions(connector_id, action_type) ON DELETE CASCADE
);

CREATE INDEX idx_standing_approval_templates_connector ON standing_approval_templates(connector_id);

-- 4. Drop action configuration tables.
DROP TABLE IF EXISTS action_config_templates;
DROP TABLE IF EXISTS action_configurations;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

CREATE TABLE action_configurations (
    id TEXT PRIMARY KEY,
    agent_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    connector_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    parameters TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CONSTRAINT action_configurations_id_check CHECK (length(id) <= 255),
    CONSTRAINT action_configurations_connector_id_check CHECK (length(connector_id) <= 255),
    CONSTRAINT action_configurations_action_type_check CHECK (length(action_type) <= 255),
    CONSTRAINT action_configurations_name_check CHECK (length(name) <= 255),
    CONSTRAINT action_configurations_description_check CHECK (description IS NULL OR length(description) <= 4096),
    CONSTRAINT action_configurations_status_check CHECK (status IN ('active', 'disabled')),
    CONSTRAINT action_configurations_parameters_check CHECK (json_type(parameters) = 'object'),
    CONSTRAINT action_configurations_parameters_check1 CHECK (length(parameters) <= 65536),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (connector_id) REFERENCES connectors(id) ON DELETE CASCADE,
    FOREIGN KEY (connector_id, action_type) REFERENCES connector_actions(connector_id, action_type) ON DELETE CASCADE
);

CREATE INDEX idx_action_configurations_agent ON action_configurations(agent_id, user_id);
CREATE INDEX idx_action_configurations_connector_action ON action_configurations(connector_id, action_type);

CREATE TABLE action_config_templates (
    id TEXT PRIMARY KEY,
    connector_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    parameters TEXT NOT NULL DEFAULT '{}',
    standing_approval_spec TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CONSTRAINT action_config_templates_id_check CHECK (length(id) <= 255),
    CONSTRAINT action_config_templates_connector_id_check CHECK (length(connector_id) <= 255),
    CONSTRAINT action_config_templates_action_type_check CHECK (length(action_type) <= 255),
    CONSTRAINT action_config_templates_name_check CHECK (length(name) <= 255),
    CONSTRAINT action_config_templates_description_check CHECK (description IS NULL OR length(description) <= 4096),
    CONSTRAINT action_config_templates_parameters_check CHECK (json_type(parameters) = 'object'),
    CONSTRAINT action_config_templates_parameters_check1 CHECK (length(parameters) <= 65536),
    CONSTRAINT action_config_templates_standing_approval_spec_check CHECK (standing_approval_spec IS NULL OR json_type(standing_approval_spec) = 'object'),
    CONSTRAINT action_config_templates_standing_approval_spec_check1 CHECK (standing_approval_spec IS NULL OR length(standing_approval_spec) <= 4096),
    FOREIGN KEY (connector_id) REFERENCES connectors(id) ON DELETE CASCADE,
    FOREIGN KEY (connector_id, action_type) REFERENCES connector_actions(connector_id, action_type) ON DELETE CASCADE
);

CREATE INDEX idx_action_config_templates_connector ON action_config_templates(connector_id);

DROP TABLE IF EXISTS standing_approval_templates;

CREATE TABLE standing_approvals_old (
    standing_approval_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    action_version TEXT NOT NULL DEFAULT '1',
    constraints TEXT,
    status TEXT NOT NULL,
    starts_at TEXT NOT NULL,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    revoked_at TEXT,
    expired_at TEXT,
    agent_id INTEGER NOT NULL,
    source_action_configuration_id TEXT NOT NULL,
    connector_instance_id TEXT,
    CONSTRAINT standing_approvals_action_type_check CHECK (length(action_type) <= 128),
    CONSTRAINT standing_approvals_action_version_check CHECK (length(action_version) <= 10),
    CONSTRAINT standing_approvals_constraints_check CHECK (length(constraints) <= 65536),
    CONSTRAINT standing_approvals_expires_at_after_starts_at CHECK (expires_at IS NULL OR expires_at >= starts_at),
    CONSTRAINT standing_approvals_standing_approval_id_check CHECK (length(standing_approval_id) <= 255),
    CONSTRAINT standing_approvals_status_check CHECK (status IN ('active', 'expired', 'revoked')),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (source_action_configuration_id) REFERENCES action_configurations(id) ON DELETE RESTRICT,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);

INSERT INTO standing_approvals_old (
    standing_approval_id, user_id, action_type, action_version, constraints,
    status, starts_at, expires_at, created_at, revoked_at, expired_at,
    agent_id, source_action_configuration_id, connector_instance_id
)
SELECT
    standing_approval_id, user_id, action_type, action_version, constraints,
    status, starts_at, expires_at, created_at, revoked_at, expired_at,
    agent_id, 'ac_migrated_placeholder', connector_instance_id
FROM standing_approvals;

DROP TABLE standing_approvals;
ALTER TABLE standing_approvals_old RENAME TO standing_approvals;

CREATE INDEX idx_standing_approvals_agent_action_status ON standing_approvals(agent_id, action_type, status);
CREATE INDEX idx_standing_approvals_agent_action_status_connector_instance ON standing_approvals(agent_id, action_type, status, connector_instance_id);
CREATE INDEX idx_standing_approvals_agent_status_created ON standing_approvals(agent_id, status, created_at DESC, standing_approval_id DESC);
CREATE INDEX idx_standing_approvals_source_config_active ON standing_approvals(source_action_configuration_id) WHERE status = 'active';
CREATE INDEX idx_standing_approvals_user_active ON standing_approvals(user_id) WHERE status = 'active';

CREATE TABLE standing_approval_requests_old (
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

INSERT INTO standing_approval_requests_old (
    request_id, agent_id, user_id, action_type, action_version, constraints,
    source_action_configuration_id, connector_name, connector_instance_display,
    connector_instance_id, status, decided_at, resulting_standing_approval_id,
    created_at, updated_at
)
SELECT
    request_id, agent_id, user_id, action_type, action_version, constraints,
    NULL, connector_name, connector_instance_display,
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
