-- +goose Up

-- 1. Add name and description to standing_approvals.
ALTER TABLE standing_approvals
    ADD COLUMN name text CHECK (char_length(name) <= 255),
    ADD COLUMN description text CHECK (char_length(description) <= 4096);

-- 2. Backfill from linked action configurations.
UPDATE standing_approvals sa
SET name = ac.name,
    description = ac.description
FROM action_configurations ac
WHERE sa.source_action_configuration_id = ac.id
  AND sa.name IS NULL;

-- 3. Drop source_action_configuration_id from standing_approvals.
ALTER TABLE standing_approvals
    DROP CONSTRAINT IF EXISTS standing_approvals_source_action_configuration_id_fkey;

DROP INDEX IF EXISTS idx_standing_approvals_source_config_active;

ALTER TABLE standing_approvals
    DROP COLUMN source_action_configuration_id;

-- 4. Drop source_action_configuration_id from standing_approval_requests.
ALTER TABLE standing_approval_requests
    DROP CONSTRAINT IF EXISTS standing_approval_requests_source_action_configuration_id_fkey;

ALTER TABLE standing_approval_requests
    DROP COLUMN source_action_configuration_id;

-- 5. Create standing_approval_templates.
CREATE TABLE standing_approval_templates (
    id              text        PRIMARY KEY CHECK (char_length(id) <= 255),
    connector_id    text        NOT NULL REFERENCES connectors(id) ON DELETE CASCADE
                                CHECK (char_length(connector_id) <= 255),
    action_type     text        NOT NULL CHECK (char_length(action_type) <= 255),
    name            text        NOT NULL CHECK (char_length(name) <= 255),
    description     text        CHECK (char_length(description) <= 4096),
    constraints     jsonb       NOT NULL DEFAULT '{}'
                                CHECK (jsonb_typeof(constraints) = 'object')
                                CHECK (pg_column_size(constraints) <= 65536),
    duration_days   integer     CHECK (duration_days IS NULL OR duration_days > 0),
    created_at      timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (connector_id, action_type) REFERENCES connector_actions(connector_id, action_type) ON DELETE CASCADE
);

CREATE INDEX idx_standing_approval_templates_connector ON standing_approval_templates (connector_id);

ALTER TABLE standing_approval_templates ENABLE ROW LEVEL SECURITY;
CREATE POLICY app_backend_all ON standing_approval_templates FOR ALL TO app_backend USING (true) WITH CHECK (true);

GRANT SELECT, INSERT, UPDATE, DELETE ON standing_approval_templates TO app_backend;

-- 6. Drop action configuration tables.
DROP TABLE IF EXISTS action_config_templates;
DROP TABLE IF EXISTS action_configurations;

-- +goose Down

CREATE TABLE action_configurations (
    id              text        PRIMARY KEY CHECK (char_length(id) <= 255),
    agent_id        bigint      NOT NULL REFERENCES agents(agent_id) ON DELETE CASCADE,
    user_id         uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    connector_id    text        NOT NULL REFERENCES connectors(id) ON DELETE CASCADE
                                CHECK (char_length(connector_id) <= 255),
    action_type     text        NOT NULL CHECK (char_length(action_type) <= 255),
    parameters      jsonb       NOT NULL DEFAULT '{}'
                                CHECK (pg_column_size(parameters) <= 65536),
    status          text        NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'disabled')),
    name            text        NOT NULL CHECK (char_length(name) <= 255),
    description     text        CHECK (char_length(description) <= 4096),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (connector_id, action_type) REFERENCES connector_actions(connector_id, action_type) ON DELETE CASCADE
);

CREATE INDEX idx_action_configurations_agent ON action_configurations (agent_id, user_id);
CREATE INDEX idx_action_configurations_connector_action ON action_configurations (connector_id, action_type);

CREATE TABLE action_config_templates (
    id                      text        PRIMARY KEY CHECK (char_length(id) <= 255),
    connector_id            text        NOT NULL REFERENCES connectors(id) ON DELETE CASCADE
                                        CHECK (char_length(connector_id) <= 255),
    action_type             text        NOT NULL CHECK (char_length(action_type) <= 255),
    name                    text        NOT NULL CHECK (char_length(name) <= 255),
    description             text        CHECK (char_length(description) <= 4096),
    parameters              jsonb       NOT NULL DEFAULT '{}'
                                        CHECK (jsonb_typeof(parameters) = 'object')
                                        CHECK (pg_column_size(parameters) <= 65536),
    standing_approval_spec  jsonb       CHECK (standing_approval_spec IS NULL OR jsonb_typeof(standing_approval_spec) = 'object')
                                        CHECK (standing_approval_spec IS NULL OR pg_column_size(standing_approval_spec) <= 4096),
    created_at              timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (connector_id, action_type) REFERENCES connector_actions(connector_id, action_type) ON DELETE CASCADE
);

CREATE INDEX idx_action_config_templates_connector ON action_config_templates (connector_id);

ALTER TABLE action_config_templates ENABLE ROW LEVEL SECURITY;
CREATE POLICY app_backend_all ON action_config_templates FOR ALL TO app_backend USING (true) WITH CHECK (true);

GRANT SELECT, INSERT, UPDATE, DELETE ON action_config_templates TO app_backend;
GRANT SELECT, INSERT, UPDATE, DELETE ON action_configurations TO app_backend;

DROP TABLE IF EXISTS standing_approval_templates;

ALTER TABLE standing_approval_requests
    ADD COLUMN source_action_configuration_id text;

ALTER TABLE standing_approvals
    ADD COLUMN source_action_configuration_id text;

ALTER TABLE standing_approvals
    ALTER COLUMN source_action_configuration_id SET NOT NULL;

ALTER TABLE standing_approvals
    ADD CONSTRAINT standing_approvals_source_action_configuration_id_fkey
    FOREIGN KEY (source_action_configuration_id)
    REFERENCES action_configurations(id) ON DELETE RESTRICT;

CREATE INDEX idx_standing_approvals_source_config_active ON standing_approvals (source_action_configuration_id) WHERE status = 'active';

ALTER TABLE standing_approvals
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS description;
