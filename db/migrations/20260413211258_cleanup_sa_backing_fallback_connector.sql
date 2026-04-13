-- +goose Up

-- Remove migration-only sentinel connector if present and nothing references it
-- (e.g. left in long-lived CI test DBs by an earlier revision of 20260413194737).
DELETE FROM connectors c
WHERE c.id = '__migrated_sa_backing_fallback'
  AND NOT EXISTS (
      SELECT 1 FROM action_configurations ac WHERE ac.connector_id = c.id
  );

-- +goose Down

-- No-op: re-inserting the sentinel is not required for rollback semantics.
