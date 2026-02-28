-- +goose Up
-- Allow "pending" as an audit event outcome (agent registration initiated but
-- not yet verified) and "registration_invite" as a source type.
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_outcome_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_outcome_check
    CHECK (outcome IN (
        'approved', 'denied', 'cancelled', 'auto_executed', 'registered', 'deactivated', 'pending'
    ));

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_source_type_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_source_type_check
    CHECK (source_type IN ('approval', 'standing_approval', 'agent', 'registration_invite'));

-- +goose Down
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_outcome_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_outcome_check
    CHECK (outcome IN (
        'approved', 'denied', 'cancelled', 'auto_executed', 'registered', 'deactivated'
    ));

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_source_type_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_source_type_check
    CHECK (source_type IN ('approval', 'standing_approval', 'agent'));
