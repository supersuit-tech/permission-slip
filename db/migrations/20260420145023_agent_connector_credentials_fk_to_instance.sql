-- +goose Up
-- Bind credentials to a specific connector instance (not just agent+connector).

ALTER TABLE agent_connector_credentials
    ADD COLUMN connector_instance_id uuid;

UPDATE agent_connector_credentials acc
SET connector_instance_id = ac.connector_instance_id
FROM agent_connectors ac
WHERE acc.agent_id = ac.agent_id
  AND acc.connector_id = ac.connector_id
  AND acc.approver_id = ac.approver_id;

ALTER TABLE agent_connector_credentials
    ALTER COLUMN connector_instance_id SET NOT NULL;

-- Legacy FK already dropped in add_connector_instance_id_to_agent_connectors (required before PK swap).

DROP INDEX IF EXISTS idx_agent_connector_credentials_unique;

ALTER TABLE agent_connector_credentials
    ADD CONSTRAINT agent_connector_credentials_agent_instance_fkey
    FOREIGN KEY (agent_id, approver_id, connector_id, connector_instance_id)
    REFERENCES agent_connectors (agent_id, approver_id, connector_id, connector_instance_id)
    ON DELETE CASCADE;

CREATE UNIQUE INDEX idx_agent_connector_credentials_unique
    ON agent_connector_credentials (agent_id, connector_id, connector_instance_id);

-- Back-compat: INSERT may omit connector_instance_id; resolve to the default instance for the pair.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION trg_agent_connector_credentials_before_insert()
RETURNS trigger AS $$
BEGIN
    IF NEW.connector_instance_id IS NULL THEN
        SELECT connector_instance_id INTO STRICT NEW.connector_instance_id
        FROM agent_connectors
        WHERE agent_id = NEW.agent_id
          AND approver_id = NEW.approver_id
          AND connector_id = NEW.connector_id
          AND is_default;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_agent_connector_credentials_before_insert
    BEFORE INSERT ON agent_connector_credentials
    FOR EACH ROW
    EXECUTE FUNCTION trg_agent_connector_credentials_before_insert();

-- +goose Down

DROP TRIGGER IF EXISTS trg_agent_connector_credentials_before_insert ON agent_connector_credentials;
DROP FUNCTION IF EXISTS trg_agent_connector_credentials_before_insert();

DROP INDEX IF EXISTS idx_agent_connector_credentials_unique;

ALTER TABLE agent_connector_credentials
    DROP CONSTRAINT IF EXISTS agent_connector_credentials_agent_instance_fkey;

CREATE UNIQUE INDEX idx_agent_connector_credentials_unique
    ON agent_connector_credentials (agent_id, connector_id);

ALTER TABLE agent_connector_credentials
    ADD CONSTRAINT agent_connector_credentials_agent_id_connector_id_fkey
    FOREIGN KEY (agent_id, connector_id)
    REFERENCES agent_connectors (agent_id, connector_id)
    ON DELETE CASCADE;

ALTER TABLE agent_connector_credentials
    DROP COLUMN connector_instance_id;
