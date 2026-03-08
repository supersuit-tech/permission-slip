-- +goose Up

-- Allow multiple auth types (e.g. oauth2 + api_key) for the same service
-- within a connector. Previously the PK was (connector_id, service) which
-- only allowed one auth type per service per connector.
ALTER TABLE connector_required_credentials
    DROP CONSTRAINT connector_required_credentials_pkey;

ALTER TABLE connector_required_credentials
    ADD PRIMARY KEY (connector_id, service, auth_type);

-- +goose Down

-- Before restoring the old PK on (connector_id, service), ensure there is at
-- most one row per (connector_id, service) pair. Keep one deterministic row
-- (the one with the smallest auth_type) and delete the rest.
DELETE FROM connector_required_credentials crc
USING (
    SELECT
        connector_id,
        service,
        MIN(auth_type) AS keep_auth_type
    FROM connector_required_credentials
    GROUP BY connector_id, service
    HAVING COUNT(*) > 1
) d
WHERE crc.connector_id = d.connector_id
  AND crc.service = d.service
  AND crc.auth_type <> d.keep_auth_type;

ALTER TABLE connector_required_credentials
    DROP CONSTRAINT connector_required_credentials_pkey;

ALTER TABLE connector_required_credentials
    ADD PRIMARY KEY (connector_id, service);
