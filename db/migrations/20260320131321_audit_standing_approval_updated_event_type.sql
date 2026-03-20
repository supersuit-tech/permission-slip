-- +goose Up
-- Add standing_approval.updated and outcome 'updated' for PATCH/POST standing approval updates.
-- Go code (emitStandingApprovalUpdateAuditEvent) already emits these; production failed on DB CHECK.
--
-- Use NOT VALID + VALIDATE CONSTRAINT to shorten ACCESS EXCLUSIVE time vs a single ADD that
-- validates immediately (existing rows already satisfy the widened checks).

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_event_type_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_event_type_check CHECK (
    event_type IN (
        'approval.requested',
        'approval.approved',
        'approval.denied',
        'approval.cancelled',
        'action.executed',
        'standing_approval.executed',
        'standing_approval.updated',
        'agent.registered',
        'agent.deactivated',
        'payment_method.charged'
    )
) NOT VALID;
ALTER TABLE audit_events VALIDATE CONSTRAINT audit_events_event_type_check;

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_outcome_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_outcome_check
    CHECK (outcome IN (
        'approved', 'denied', 'cancelled', 'auto_executed', 'registered', 'deactivated', 'pending', 'expired', 'charged', 'updated'
    )) NOT VALID;
ALTER TABLE audit_events VALIDATE CONSTRAINT audit_events_outcome_check;

-- +goose Down
-- NOT VALID so rollback does not fail when rows already contain standing_approval.updated / updated.
-- New inserts/updates are still checked against the restored (narrower) rules.
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_event_type_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_event_type_check CHECK (
    event_type IN (
        'approval.requested',
        'approval.approved',
        'approval.denied',
        'approval.cancelled',
        'action.executed',
        'standing_approval.executed',
        'agent.registered',
        'agent.deactivated',
        'payment_method.charged'
    )
) NOT VALID;

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_outcome_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_outcome_check
    CHECK (outcome IN (
        'approved', 'denied', 'cancelled', 'auto_executed', 'registered', 'deactivated', 'pending', 'expired', 'charged'
    )) NOT VALID;
