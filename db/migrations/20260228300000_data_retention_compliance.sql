-- +goose Up

-- Change audit_events FK on user_id from RESTRICT to CASCADE so that
-- deleting a profile cascades to audit events. This is required for
-- account deletion (GDPR / data retention compliance).
ALTER TABLE audit_events
    DROP CONSTRAINT audit_events_user_id_fkey,
    ADD CONSTRAINT audit_events_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

-- Similarly, change the agent_id FK from RESTRICT to CASCADE so that
-- audit events are cleaned up when an agent's owner deletes their account
-- (agents cascade from profiles, and audit events should follow).
ALTER TABLE audit_events
    DROP CONSTRAINT audit_events_agent_id_fkey,
    ADD CONSTRAINT audit_events_agent_id_fkey
        FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE;

-- +goose Down

ALTER TABLE audit_events
    DROP CONSTRAINT audit_events_agent_id_fkey,
    ADD CONSTRAINT audit_events_agent_id_fkey
        FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE RESTRICT;

ALTER TABLE audit_events
    DROP CONSTRAINT audit_events_user_id_fkey,
    ADD CONSTRAINT audit_events_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE RESTRICT;
