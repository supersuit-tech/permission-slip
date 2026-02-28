-- +goose Up

-- Add connector_id to audit_events for per-connector analytics and billing.
-- Nullable because agent lifecycle events (registered, deactivated) have no
-- associated connector.
-- No FK constraint because the connector_id is derived from the action type
-- string and used for analytics — the connector may not yet be registered
-- (or may have been removed).
ALTER TABLE audit_events
    ADD COLUMN connector_id text;

CREATE INDEX idx_audit_events_connector_created
    ON audit_events (connector_id, created_at DESC)
    WHERE connector_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_audit_events_connector_created;
ALTER TABLE audit_events DROP COLUMN IF EXISTS connector_id;
