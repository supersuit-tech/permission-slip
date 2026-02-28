-- +goose Up
-- Add 'action.executed' to the audit_events event_type CHECK constraint.
-- This supports the POST /v1/actions/execute endpoint which emits audit events
-- when an action is executed via a one-off approval token.

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
        'agent.deactivated'
    )
);

-- +goose Down
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_event_type_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_event_type_check CHECK (
    event_type IN (
        'approval.requested',
        'approval.approved',
        'approval.denied',
        'approval.cancelled',
        'standing_approval.executed',
        'agent.registered',
        'agent.deactivated'
    )
);
