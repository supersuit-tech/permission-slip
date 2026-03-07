-- +goose Up

-- Allow multiple auth types (e.g. oauth2 + api_key) for the same service
-- within a connector. Previously the PK was (connector_id, service) which
-- only allowed one auth type per service per connector.
ALTER TABLE connector_required_credentials
    DROP CONSTRAINT connector_required_credentials_pkey;

ALTER TABLE connector_required_credentials
    ADD PRIMARY KEY (connector_id, service, auth_type);

-- +goose Down

ALTER TABLE connector_required_credentials
    DROP CONSTRAINT connector_required_credentials_pkey;

ALTER TABLE connector_required_credentials
    ADD PRIMARY KEY (connector_id, service);
