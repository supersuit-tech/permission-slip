-- +goose Up
-- Multi-instance connector support: surrogate instance id + label + default flag per (agent, connector).
-- Backfill: one instance per existing row; label from connectors.name; all rows are default.

ALTER TABLE agent_connectors
    ADD COLUMN connector_instance_id uuid NOT NULL DEFAULT gen_random_uuid(),
    ADD COLUMN label text,
    ADD COLUMN is_default boolean NOT NULL DEFAULT false;

UPDATE agent_connectors ac
SET label = c.name
FROM connectors c
WHERE c.id = ac.connector_id;

ALTER TABLE agent_connectors
    ALTER COLUMN label SET NOT NULL;

-- Exactly one default instance per (agent_id, connector_id) — every existing row is that default.
UPDATE agent_connectors SET is_default = true;

CREATE UNIQUE INDEX uq_agent_connectors_agent_connector_label
    ON agent_connectors (agent_id, connector_id, label);

CREATE UNIQUE INDEX uq_agent_connectors_default_per_pair
    ON agent_connectors (agent_id, connector_id)
    WHERE is_default;

-- Back-compat: allow INSERT with only (agent_id, approver_id, connector_id); fill label and default flag.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION trg_agent_connectors_before_insert()
RETURNS trigger AS $$
BEGIN
    IF NEW.label IS NULL THEN
        SELECT c.name INTO NEW.label FROM connectors c WHERE c.id = NEW.connector_id;
        IF NEW.label IS NULL THEN
            -- Match FK violation so API maps to connector_not_found (STRICT would raise P0002 / ErrNoRows).
            RAISE EXCEPTION 'insert or update on table "agent_connectors" violates foreign key constraint'
                USING ERRCODE = '23503';
        END IF;
    END IF;
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

-- agent_connector_credentials references this PK; drop FK before PK swap (re-added in next migration).
ALTER TABLE agent_connector_credentials
    DROP CONSTRAINT IF EXISTS agent_connector_credentials_agent_id_connector_id_fkey;

ALTER TABLE agent_connectors DROP CONSTRAINT agent_connectors_pkey;

ALTER TABLE agent_connectors
    ADD CONSTRAINT agent_connectors_pkey
    PRIMARY KEY (agent_id, approver_id, connector_id, connector_instance_id);

-- +goose Down

DROP TRIGGER IF EXISTS trg_agent_connectors_before_insert ON agent_connectors;
DROP FUNCTION IF EXISTS trg_agent_connectors_before_insert();

ALTER TABLE agent_connectors DROP CONSTRAINT agent_connectors_pkey;

ALTER TABLE agent_connectors
    ADD CONSTRAINT agent_connectors_pkey
    PRIMARY KEY (agent_id, connector_id);

DROP INDEX IF EXISTS uq_agent_connectors_default_per_pair;
DROP INDEX IF EXISTS uq_agent_connectors_agent_connector_label;

ALTER TABLE agent_connectors
    DROP COLUMN connector_instance_id,
    DROP COLUMN label,
    DROP COLUMN is_default;
