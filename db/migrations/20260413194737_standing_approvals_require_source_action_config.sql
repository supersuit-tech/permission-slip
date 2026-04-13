-- +goose Up

-- Clear dangling references (deleted configs) so later steps can backfill.
UPDATE standing_approvals sa
SET source_action_configuration_id = NULL
WHERE sa.source_action_configuration_id IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM action_configurations ac WHERE ac.id = sa.source_action_configuration_id
  );

-- Link legacy rows with NULL source_action_configuration_id when a matching active config exists.
UPDATE standing_approvals sa
SET source_action_configuration_id = sub.config_id
FROM (
    SELECT sa2.standing_approval_id,
           (
               SELECT ac.id
               FROM action_configurations ac
               WHERE ac.agent_id = sa2.agent_id
                 AND ac.user_id = sa2.user_id
                 AND ac.action_type = sa2.action_type
                 AND ac.status = 'active'
               ORDER BY ac.created_at DESC
               LIMIT 1
           ) AS config_id
    FROM standing_approvals sa2
    WHERE sa2.source_action_configuration_id IS NULL
) sub
WHERE sa.standing_approval_id = sub.standing_approval_id
  AND sub.config_id IS NOT NULL;

-- Remaining rows: create one backing disabled config per standing approval.
-- +goose StatementBegin
DO $$
DECLARE
    r RECORD;
    new_id text;
    resolved_connector text;
    fallback_connector text;
BEGIN
    SELECT id INTO fallback_connector FROM connectors ORDER BY id LIMIT 1;

    -- Do not INSERT into connectors here: CI uses a long-lived test DB and any
    -- extra row would break tests that expect an empty connectors table. If
    -- orphans remain with no connector to attach, fail loudly (requires manual
    -- remediation: seed a connector or clean orphan standing approvals).
    IF fallback_connector IS NULL AND EXISTS (
        SELECT 1 FROM standing_approvals WHERE source_action_configuration_id IS NULL
    ) THEN
        RAISE EXCEPTION 'Cannot backfill standing approvals: connectors table is empty but orphan standing approvals remain';
    END IF;

    FOR r IN
        SELECT standing_approval_id, agent_id, user_id, action_type
        FROM standing_approvals
        WHERE source_action_configuration_id IS NULL
    LOOP
        new_id := 'ac_' || encode(gen_random_bytes(16), 'hex');

        IF r.action_type = '*' THEN
            resolved_connector := fallback_connector;
        ELSE
            SELECT ca.connector_id INTO resolved_connector
            FROM connector_actions ca
            WHERE ca.action_type = r.action_type
            LIMIT 1;
        END IF;

        IF resolved_connector IS NULL THEN
            resolved_connector := fallback_connector;
        END IF;

        INSERT INTO action_configurations (
            id, agent_id, user_id, connector_id, action_type,
            parameters, status, name
        )
        VALUES (
            new_id, r.agent_id, r.user_id, resolved_connector, r.action_type,
            '{}'::jsonb, 'disabled',
            'Migrated standing approval backing (auto-created)'
        );

        UPDATE standing_approvals
        SET source_action_configuration_id = new_id
        WHERE standing_approval_id = r.standing_approval_id;
    END LOOP;
END $$;
-- +goose StatementEnd

ALTER TABLE standing_approvals
    ALTER COLUMN source_action_configuration_id SET NOT NULL;

ALTER TABLE standing_approvals
    ADD CONSTRAINT standing_approvals_source_action_configuration_id_fkey
    FOREIGN KEY (source_action_configuration_id)
    REFERENCES action_configurations(id) ON DELETE RESTRICT;

-- +goose Down

ALTER TABLE standing_approvals
    DROP CONSTRAINT IF EXISTS standing_approvals_source_action_configuration_id_fkey;

ALTER TABLE standing_approvals
    ALTER COLUMN source_action_configuration_id DROP NOT NULL;
