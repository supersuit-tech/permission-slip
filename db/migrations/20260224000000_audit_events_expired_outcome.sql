-- +goose Up
-- Allow "expired" as an audit event outcome for agent registrations that
-- timed out before verification completed.
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_outcome_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_outcome_check
    CHECK (outcome IN (
        'approved', 'denied', 'cancelled', 'auto_executed', 'registered', 'deactivated', 'pending', 'expired'
    ));

-- +goose Down
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_outcome_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_outcome_check
    CHECK (outcome IN (
        'approved', 'denied', 'cancelled', 'auto_executed', 'registered', 'deactivated', 'pending'
    ));
