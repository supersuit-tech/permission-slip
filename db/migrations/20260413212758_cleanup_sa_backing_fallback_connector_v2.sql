-- +goose Up

-- Second pass: CI may have applied 20260413211258 before it deleted dependent rows.
DELETE FROM standing_approvals
WHERE source_action_configuration_id IN (
    SELECT id FROM action_configurations
    WHERE connector_id = '__migrated_sa_backing_fallback'
);

DELETE FROM action_configurations
WHERE connector_id = '__migrated_sa_backing_fallback';

DELETE FROM connectors WHERE id = '__migrated_sa_backing_fallback';

-- +goose Down

-- No-op
