-- +goose Up

-- Remove migration-only sentinel connector and rows that referenced it only
-- (left in long-lived CI test DBs by an earlier revision of 20260413194737).
DELETE FROM standing_approvals
WHERE source_action_configuration_id IN (
    SELECT id FROM action_configurations
    WHERE connector_id = '__migrated_sa_backing_fallback'
);

DELETE FROM action_configurations
WHERE connector_id = '__migrated_sa_backing_fallback';

DELETE FROM connectors WHERE id = '__migrated_sa_backing_fallback';

-- +goose Down

-- No-op: re-inserting the sentinel is not required for rollback semantics.
