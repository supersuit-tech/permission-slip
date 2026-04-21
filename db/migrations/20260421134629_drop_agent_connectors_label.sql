-- +goose Up
-- Remove redundant per-instance label; display names come from credentials (issue #974 phase 1).

DROP TRIGGER IF EXISTS trg_agent_connectors_before_insert ON agent_connectors;
DROP FUNCTION IF EXISTS trg_agent_connectors_before_insert();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION trg_agent_connectors_before_insert()
RETURNS trigger AS $$
BEGIN
    -- First row for (agent, connector) must be the default instance (column default is false).
    IF NOT EXISTS (
        SELECT 1 FROM agent_connectors
        WHERE agent_id = NEW.agent_id
          AND connector_id = NEW.connector_id
          AND is_default
    ) THEN
        NEW.is_default := true;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_agent_connectors_before_insert
    BEFORE INSERT ON agent_connectors
    FOR EACH ROW
    EXECUTE FUNCTION trg_agent_connectors_before_insert();

DROP INDEX IF EXISTS uq_agent_connectors_agent_connector_label;

ALTER TABLE agent_connectors DROP COLUMN label;

-- +goose Down

ALTER TABLE agent_connectors
    ADD COLUMN label text;

UPDATE agent_connectors ac
SET label = c.name
FROM connectors c
WHERE c.id = ac.connector_id;

ALTER TABLE agent_connectors
    ALTER COLUMN label SET NOT NULL;

CREATE UNIQUE INDEX uq_agent_connectors_agent_connector_label
    ON agent_connectors (agent_id, connector_id, label);

DROP TRIGGER IF EXISTS trg_agent_connectors_before_insert ON agent_connectors;
DROP FUNCTION IF EXISTS trg_agent_connectors_before_insert();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION trg_agent_connectors_before_insert()
RETURNS trigger AS $$
BEGIN
    IF NEW.label IS NULL THEN
        SELECT c.name INTO NEW.label FROM connectors c WHERE c.id = NEW.connector_id;
        IF NEW.label IS NULL THEN
            RAISE EXCEPTION 'insert or update on table "agent_connectors" violates foreign key constraint'
                USING ERRCODE = '23503';
        END IF;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM agent_connectors
        WHERE agent_id = NEW.agent_id
          AND connector_id = NEW.connector_id
          AND is_default
    ) THEN
        NEW.is_default := true;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_agent_connectors_before_insert
    BEFORE INSERT ON agent_connectors
    FOR EACH ROW
    EXECUTE FUNCTION trg_agent_connectors_before_insert();
