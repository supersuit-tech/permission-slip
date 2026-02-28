-- +goose Up

CREATE TABLE connectors (
    id          text        PRIMARY KEY CHECK (char_length(id) <= 255),
    name        text        NOT NULL CHECK (char_length(name) <= 255),
    description text        CHECK (char_length(description) <= 4096),
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE connector_actions (
    connector_id      text  NOT NULL REFERENCES connectors(id) ON DELETE CASCADE
                            CHECK (char_length(connector_id) <= 255),
    action_type       text  NOT NULL CHECK (char_length(action_type) <= 255),
    name              text  NOT NULL CHECK (char_length(name) <= 255),
    description       text  CHECK (char_length(description) <= 4096),
    risk_level        text  CHECK (risk_level IN ('low', 'medium', 'high')),
    parameters_schema jsonb CHECK (pg_column_size(parameters_schema) <= 65536),
    PRIMARY KEY (connector_id, action_type)
);

CREATE TABLE connector_required_credentials (
    connector_id text NOT NULL REFERENCES connectors(id) ON DELETE CASCADE
                      CHECK (char_length(connector_id) <= 255),
    service      text NOT NULL CHECK (char_length(service) <= 255),
    auth_type    text NOT NULL CHECK (auth_type IN ('api_key', 'oauth2', 'basic', 'custom')),
    setup_url    text CHECK (char_length(setup_url) <= 2048),
    PRIMARY KEY (connector_id, service)
);

-- +goose Down
DROP TABLE IF EXISTS connector_required_credentials;
DROP TABLE IF EXISTS connector_actions;
DROP TABLE IF EXISTS connectors;
