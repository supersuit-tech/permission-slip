-- +goose Up

-- New join table for agent-connector credential bindings.
-- Uses a join table (not columns on agent_connectors) so the schema can
-- grow to support credential sharing across agents in the future.
-- For now, a unique index enforces one binding per agent+connector.
CREATE TABLE agent_connector_credentials (
    id                  text        PRIMARY KEY CHECK (char_length(id) <= 255),
    agent_id            bigint      NOT NULL,
    connector_id        text        NOT NULL CHECK (char_length(connector_id) <= 255),
    approver_id         uuid        NOT NULL,
    -- Exactly one of these must be non-null.
    credential_id       text        REFERENCES credentials(id) ON DELETE CASCADE,
    oauth_connection_id text        REFERENCES oauth_connections(id) ON DELETE CASCADE,
    created_at          timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (agent_id, connector_id)
        REFERENCES agent_connectors(agent_id, connector_id) ON DELETE CASCADE,
    CHECK (
        (credential_id IS NOT NULL AND oauth_connection_id IS NULL) OR
        (credential_id IS NULL AND oauth_connection_id IS NOT NULL)
    )
);

-- One credential binding per agent+connector for now.
CREATE UNIQUE INDEX idx_agent_connector_credentials_unique
    ON agent_connector_credentials (agent_id, connector_id);

CREATE INDEX idx_agent_connector_credentials_cred
    ON agent_connector_credentials (credential_id);
CREATE INDEX idx_agent_connector_credentials_oauth
    ON agent_connector_credentials (oauth_connection_id);

-- RLS + grant for app_backend role
ALTER TABLE agent_connector_credentials ENABLE ROW LEVEL SECURITY;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_backend') THEN
        CREATE POLICY app_backend_all ON agent_connector_credentials
            FOR ALL TO app_backend USING (true) WITH CHECK (true);
        GRANT SELECT, INSERT, UPDATE, DELETE ON agent_connector_credentials TO app_backend;
    END IF;
END $$;
-- +goose StatementEnd

-- Drop credential_id from action_configurations (no longer needed).
ALTER TABLE action_configurations DROP COLUMN IF EXISTS credential_id;

-- +goose Down

ALTER TABLE action_configurations
    ADD COLUMN credential_id text REFERENCES credentials(id) ON DELETE SET NULL
                             CHECK (char_length(credential_id) <= 255);

DROP TABLE IF EXISTS agent_connector_credentials;
