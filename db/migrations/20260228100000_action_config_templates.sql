-- +goose Up

CREATE TABLE action_config_templates (
    id              text        PRIMARY KEY CHECK (char_length(id) <= 255),
    connector_id    text        NOT NULL REFERENCES connectors(id) ON DELETE CASCADE
                                CHECK (char_length(connector_id) <= 255),
    action_type     text        NOT NULL CHECK (char_length(action_type) <= 255),
    name            text        NOT NULL CHECK (char_length(name) <= 255),
    description     text        CHECK (char_length(description) <= 4096),
    parameters      jsonb       NOT NULL DEFAULT '{}'
                                CHECK (jsonb_typeof(parameters) = 'object')
                                CHECK (pg_column_size(parameters) <= 65536),
    created_at      timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (connector_id, action_type) REFERENCES connector_actions(connector_id, action_type) ON DELETE CASCADE
);

CREATE INDEX idx_action_config_templates_connector ON action_config_templates (connector_id);

-- +goose Down
DROP TABLE IF EXISTS action_config_templates;
