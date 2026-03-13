-- +goose Up
-- Supports the NOT EXISTS subquery in ListAuditEvents that deduplicates
-- approval.requested events when a resolution (approved/denied/cancelled) exists.
CREATE INDEX idx_audit_events_source_resolution
    ON audit_events (source_id, user_id, event_type)
    WHERE source_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_audit_events_source_resolution;
