-- +goose Up

-- Remove legacy shared test fixture connector leaked into long-lived DBs
-- (previously inserted with ON CONFLICT DO NOTHING from testhelper).
DELETE FROM standing_approvals
WHERE source_action_configuration_id IN (
    SELECT id FROM action_configurations WHERE connector_id = 'standing_approval_fixture'
);

DELETE FROM action_configurations WHERE connector_id = 'standing_approval_fixture';

DELETE FROM connector_actions WHERE connector_id = 'standing_approval_fixture';

DELETE FROM connectors WHERE id = 'standing_approval_fixture';

-- +goose Down

-- No-op
