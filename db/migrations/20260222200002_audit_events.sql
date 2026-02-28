-- +goose Up

CREATE TABLE audit_events (
    id            bigserial    PRIMARY KEY,
    user_id       uuid         NOT NULL REFERENCES profiles(id) ON DELETE RESTRICT,
    agent_id      bigint       NOT NULL REFERENCES agents(agent_id) ON DELETE RESTRICT,
    event_type    text         NOT NULL CHECK (event_type IN (
        'approval.approved', 'approval.denied', 'approval.cancelled',
        'standing_approval.executed',
        'agent.registered', 'agent.deactivated'
    )),
    outcome       text         NOT NULL CHECK (outcome IN (
        'approved', 'denied', 'cancelled', 'auto_executed', 'registered', 'deactivated'
    )),
    source_id     text,
    source_type   text         CHECK (source_type IN ('approval', 'standing_approval', 'agent')),
    agent_meta    jsonb,
    action        jsonb,
    created_at    timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_events_user_created ON audit_events (user_id, created_at DESC, id DESC);
CREATE INDEX idx_audit_events_agent ON audit_events (agent_id, created_at DESC, id DESC);

-- Backfill from existing resolved approvals.
INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
SELECT
    a.approver_id,
    a.agent_id,
    CASE a.status
        WHEN 'approved' THEN 'approval.approved'
        WHEN 'denied' THEN 'approval.denied'
        WHEN 'cancelled' THEN 'approval.cancelled'
    END,
    a.status,
    a.approval_id,
    'approval',
    ag.metadata,
    a.action,
    COALESCE(a.approved_at, a.denied_at, a.cancelled_at)
FROM approvals a
JOIN agents ag ON ag.agent_id = a.agent_id
WHERE a.status IN ('approved', 'denied', 'cancelled')
  AND COALESCE(a.approved_at, a.denied_at, a.cancelled_at) IS NOT NULL;

-- Backfill from existing agent registrations.
INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
SELECT
    ag.approver_id,
    ag.agent_id,
    'agent.registered',
    'registered',
    'ar:' || ag.agent_id::text,
    'agent',
    ag.metadata,
    NULL,
    ag.registered_at
FROM agents ag
WHERE ag.registered_at IS NOT NULL;

-- Backfill from existing agent deactivations.
INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
SELECT
    ag.approver_id,
    ag.agent_id,
    'agent.deactivated',
    'deactivated',
    'ad:' || ag.agent_id::text,
    'agent',
    ag.metadata,
    NULL,
    ag.deactivated_at
FROM agents ag
WHERE ag.deactivated_at IS NOT NULL;

-- Backfill from existing standing approval executions.
INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
SELECT
    sa.user_id,
    sa.agent_id,
    'standing_approval.executed',
    'auto_executed',
    'sae:' || sae.id::text,
    'standing_approval',
    ag.metadata,
    jsonb_build_object(
        'type', sa.action_type,
        'version', sa.action_version,
        'parameters', sae.parameters,
        'constraints', sa.constraints
    ),
    sae.executed_at
FROM standing_approval_executions sae
JOIN standing_approvals sa ON sa.standing_approval_id = sae.standing_approval_id
JOIN agents ag ON ag.agent_id = sa.agent_id AND ag.approver_id = sa.user_id;

-- +goose Down
DROP TABLE IF EXISTS audit_events;
