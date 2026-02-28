-- +goose Up
-- Scope agent_id per approver so two different users can each register
-- an agent with the same ID (e.g. "claude-code").

-- 1. Drop foreign keys that reference the old single-column PK.
ALTER TABLE approvals DROP CONSTRAINT approvals_agent_id_fkey;
ALTER TABLE request_ids DROP CONSTRAINT request_ids_agent_id_fkey;
ALTER TABLE standing_approvals DROP CONSTRAINT standing_approvals_agent_id_fkey;

-- 2. Replace single-column PK with composite PK.
ALTER TABLE agents DROP CONSTRAINT agents_pkey;
ALTER TABLE agents ADD PRIMARY KEY (agent_id, approver_id);

-- 3. Add approver_id to request_ids and backfill from agents.
--    Safe because agent_id is currently unique (1:1 with approver_id).
ALTER TABLE request_ids ADD COLUMN approver_id uuid;
UPDATE request_ids SET approver_id = agents.approver_id
  FROM agents WHERE request_ids.agent_id = agents.agent_id;
ALTER TABLE request_ids ALTER COLUMN approver_id SET NOT NULL;

-- 4. Re-add foreign keys as composite references.
ALTER TABLE approvals
  ADD CONSTRAINT approvals_agent_id_approver_id_fkey
  FOREIGN KEY (agent_id, approver_id) REFERENCES agents(agent_id, approver_id) ON DELETE CASCADE;

ALTER TABLE request_ids
  ADD CONSTRAINT request_ids_agent_id_approver_id_fkey
  FOREIGN KEY (agent_id, approver_id) REFERENCES agents(agent_id, approver_id) ON DELETE CASCADE;

-- standing_approvals uses user_id which is semantically the same as approver_id.
ALTER TABLE standing_approvals
  ADD CONSTRAINT standing_approvals_agent_id_user_id_fkey
  FOREIGN KEY (agent_id, user_id) REFERENCES agents(agent_id, approver_id) ON DELETE CASCADE;

-- +goose Down

-- Drop composite foreign keys.
ALTER TABLE standing_approvals DROP CONSTRAINT standing_approvals_agent_id_user_id_fkey;
ALTER TABLE request_ids DROP CONSTRAINT request_ids_agent_id_approver_id_fkey;
ALTER TABLE approvals DROP CONSTRAINT approvals_agent_id_approver_id_fkey;

-- Remove approver_id from request_ids.
ALTER TABLE request_ids DROP COLUMN approver_id;

-- Restore single-column PK.
ALTER TABLE agents DROP CONSTRAINT agents_pkey;
ALTER TABLE agents ADD PRIMARY KEY (agent_id);

-- Restore original single-column foreign keys.
ALTER TABLE approvals
  ADD CONSTRAINT approvals_agent_id_fkey
  FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE;

ALTER TABLE request_ids
  ADD CONSTRAINT request_ids_agent_id_fkey
  FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE;

ALTER TABLE standing_approvals
  ADD CONSTRAINT standing_approvals_agent_id_fkey
  FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE;
