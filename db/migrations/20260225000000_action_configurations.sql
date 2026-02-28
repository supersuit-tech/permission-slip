-- +goose Up

CREATE TABLE action_configurations (
    id              text        PRIMARY KEY CHECK (char_length(id) <= 255),
    agent_id        bigint      NOT NULL REFERENCES agents(agent_id) ON DELETE CASCADE,
    user_id         uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    connector_id    text        NOT NULL REFERENCES connectors(id) ON DELETE CASCADE
                                CHECK (char_length(connector_id) <= 255),
    action_type     text        NOT NULL CHECK (char_length(action_type) <= 255),
    credential_id   text        REFERENCES credentials(id) ON DELETE SET NULL
                                CHECK (char_length(credential_id) <= 255),
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

-- +goose Down
DROP TABLE IF EXISTS action_configurations;
