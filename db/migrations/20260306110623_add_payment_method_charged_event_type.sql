-- +goose Up
-- Add 'payment_method.charged' to the audit_events event_type and outcome CHECK constraints.
-- This supports audit logging when a stored payment method is used by a connector action.
--
-- The action JSON for payment events contains only safe display metadata (brand, last4,
-- opaque payment_method_id, amount, currency). Raw card numbers, CVV, and full expiry
-- dates are NEVER stored — they remain in Stripe's PCI-compliant vault.
--
-- Also adds 'payment_method_transaction' to the source_type constraint so payment audit
-- events can reference the payment_method_transactions table as their source.

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
);

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_outcome_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_outcome_check
    CHECK (outcome IN (
        'approved', 'denied', 'cancelled', 'auto_executed', 'registered', 'deactivated', 'pending', 'expired', 'charged'
    ));

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_source_type_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_source_type_check
    CHECK (source_type IN ('approval', 'standing_approval', 'agent', 'registration_invite', 'payment_method_transaction'));

-- +goose Down
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

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_outcome_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_outcome_check
    CHECK (outcome IN (
        'approved', 'denied', 'cancelled', 'auto_executed', 'registered', 'deactivated', 'pending', 'expired'
    ));

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_source_type_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_source_type_check
    CHECK (source_type IN ('approval', 'standing_approval', 'agent', 'registration_invite'));
