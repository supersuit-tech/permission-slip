-- +goose Up

CREATE TABLE agent_connectors (
    agent_id     text        NOT NULL
                              CHECK (char_length(agent_id) <= 255),
    approver_id  uuid        NOT NULL,
    connector_id text        NOT NULL REFERENCES connectors(id) ON DELETE CASCADE
                              CHECK (char_length(connector_id) <= 255),
    enabled_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, approver_id, connector_id),
    FOREIGN KEY (agent_id, approver_id) REFERENCES agents(agent_id, approver_id) ON DELETE CASCADE
);

CREATE INDEX idx_agent_connectors_connector ON agent_connectors (connector_id);

-- +goose Down
DROP TABLE IF EXISTS agent_connectors;
