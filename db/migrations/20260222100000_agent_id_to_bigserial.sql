-- +goose Up
-- Migrate agent_id from text to auto-incrementing bigserial.
-- This is a breaking change: agents receive an ID on creation rather than providing one.

-- 1. Drop all foreign key constraints that reference agents.
ALTER TABLE agent_connectors DROP CONSTRAINT agent_connectors_agent_id_approver_id_fkey;
ALTER TABLE approvals DROP CONSTRAINT approvals_agent_id_approver_id_fkey;
ALTER TABLE request_ids DROP CONSTRAINT request_ids_agent_id_approver_id_fkey;
ALTER TABLE standing_approvals DROP CONSTRAINT standing_approvals_agent_id_user_id_fkey;

-- 2. Drop indexes that include agent_id (text).
DROP INDEX IF EXISTS idx_approvals_agent_status;
DROP INDEX IF EXISTS idx_approvals_agent_created;
DROP INDEX IF EXISTS idx_standing_approvals_agent_action_status;
DROP INDEX IF EXISTS idx_agents_approver_created;

-- 3. Drop the composite PK on agents and the agent_connectors PK.
ALTER TABLE agent_connectors DROP CONSTRAINT agent_connectors_pkey;
ALTER TABLE agents DROP CONSTRAINT agents_pkey;

-- 4. Add a bigserial column and build a mapping from old text IDs.
ALTER TABLE agents ADD COLUMN new_id BIGSERIAL;

-- Temporary mapping for backfilling dependent tables.
CREATE TEMP TABLE agent_id_map AS
    SELECT agent_id AS old_id, approver_id, new_id FROM agents;

-- 5. Replace agent_id in the agents table.
ALTER TABLE agents DROP COLUMN agent_id;
ALTER TABLE agents RENAME COLUMN new_id TO agent_id;
ALTER TABLE agents ADD PRIMARY KEY (agent_id);

-- Remove the old text CHECK constraint on agent_id if it survived.
-- (It was part of the column definition, so DROP COLUMN already removed it.)

-- 6. Replace agent_id (text → bigint) in approvals.
ALTER TABLE approvals ADD COLUMN new_agent_id BIGINT;
UPDATE approvals SET new_agent_id = m.new_id
    FROM agent_id_map m
    WHERE approvals.agent_id = m.old_id
      AND approvals.approver_id = m.approver_id;
ALTER TABLE approvals DROP COLUMN agent_id;
ALTER TABLE approvals RENAME COLUMN new_agent_id TO agent_id;
ALTER TABLE approvals ALTER COLUMN agent_id SET NOT NULL;

-- 7. Replace agent_id (text → bigint) in request_ids.
ALTER TABLE request_ids ADD COLUMN new_agent_id BIGINT;
UPDATE request_ids SET new_agent_id = m.new_id
    FROM agent_id_map m
    WHERE request_ids.agent_id = m.old_id
      AND request_ids.approver_id = m.approver_id;
ALTER TABLE request_ids DROP COLUMN agent_id;
ALTER TABLE request_ids RENAME COLUMN new_agent_id TO agent_id;
ALTER TABLE request_ids ALTER COLUMN agent_id SET NOT NULL;

-- 8. Replace agent_id (text → bigint) in standing_approvals.
ALTER TABLE standing_approvals ADD COLUMN new_agent_id BIGINT;
UPDATE standing_approvals SET new_agent_id = m.new_id
    FROM agent_id_map m
    WHERE standing_approvals.agent_id = m.old_id
      AND standing_approvals.user_id = m.approver_id;
ALTER TABLE standing_approvals DROP COLUMN agent_id;
ALTER TABLE standing_approvals RENAME COLUMN new_agent_id TO agent_id;
ALTER TABLE standing_approvals ALTER COLUMN agent_id SET NOT NULL;

-- 9. Replace agent_id (text → bigint) in agent_connectors.
ALTER TABLE agent_connectors ADD COLUMN new_agent_id BIGINT;
UPDATE agent_connectors SET new_agent_id = m.new_id
    FROM agent_id_map m
    WHERE agent_connectors.agent_id = m.old_id
      AND agent_connectors.approver_id = m.approver_id;
ALTER TABLE agent_connectors DROP COLUMN agent_id;
ALTER TABLE agent_connectors RENAME COLUMN new_agent_id TO agent_id;
ALTER TABLE agent_connectors ALTER COLUMN agent_id SET NOT NULL;

-- 10. Restore primary keys (agent_connectors simplifies to (agent_id, connector_id)
--     since agent_id is now globally unique).
ALTER TABLE agent_connectors ADD PRIMARY KEY (agent_id, connector_id);

-- 11. Restore foreign keys (single-column since agent_id is globally unique).
ALTER TABLE approvals
    ADD CONSTRAINT approvals_agent_id_fkey
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE;

ALTER TABLE request_ids
    ADD CONSTRAINT request_ids_agent_id_fkey
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE;

ALTER TABLE standing_approvals
    ADD CONSTRAINT standing_approvals_agent_id_fkey
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE;

ALTER TABLE agent_connectors
    ADD CONSTRAINT agent_connectors_agent_id_fkey
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE;

-- 12. Rebuild indexes.
CREATE INDEX idx_approvals_agent_status ON approvals (agent_id, status);
CREATE INDEX idx_approvals_agent_created ON approvals (agent_id, created_at);
CREATE INDEX idx_standing_approvals_agent_action_status ON standing_approvals (agent_id, action_type, status);
CREATE INDEX idx_agents_approver_created ON agents (approver_id, created_at DESC, agent_id DESC);

-- +goose Down
-- Reverting from bigserial back to text is destructive (the original text IDs are lost).
-- This down migration recreates the text columns but cannot restore original values.

ALTER TABLE agent_connectors DROP CONSTRAINT agent_connectors_agent_id_fkey;
ALTER TABLE standing_approvals DROP CONSTRAINT standing_approvals_agent_id_fkey;
ALTER TABLE request_ids DROP CONSTRAINT request_ids_agent_id_fkey;
ALTER TABLE approvals DROP CONSTRAINT approvals_agent_id_fkey;
DROP INDEX IF EXISTS idx_approvals_agent_status;
DROP INDEX IF EXISTS idx_standing_approvals_agent_action_status;

ALTER TABLE agent_connectors DROP CONSTRAINT agent_connectors_pkey;
ALTER TABLE agents DROP CONSTRAINT agents_pkey;

-- Restore text agent_id in agents.
ALTER TABLE agents ADD COLUMN old_agent_id TEXT;
UPDATE agents SET old_agent_id = 'agent_' || agent_id;
ALTER TABLE agents DROP COLUMN agent_id;
ALTER TABLE agents RENAME COLUMN old_agent_id TO agent_id;
ALTER TABLE agents ALTER COLUMN agent_id SET NOT NULL;
ALTER TABLE agents ADD CONSTRAINT agents_pkey PRIMARY KEY (agent_id, approver_id);

-- Restore text agent_id in dependent tables.
ALTER TABLE approvals ADD COLUMN old_agent_id TEXT;
UPDATE approvals SET old_agent_id = 'agent_' || agent_id;
ALTER TABLE approvals DROP COLUMN agent_id;
ALTER TABLE approvals RENAME COLUMN old_agent_id TO agent_id;
ALTER TABLE approvals ALTER COLUMN agent_id SET NOT NULL;

ALTER TABLE request_ids ADD COLUMN old_agent_id TEXT;
UPDATE request_ids SET old_agent_id = 'agent_' || agent_id;
ALTER TABLE request_ids DROP COLUMN agent_id;
ALTER TABLE request_ids RENAME COLUMN old_agent_id TO agent_id;
ALTER TABLE request_ids ALTER COLUMN agent_id SET NOT NULL;

ALTER TABLE standing_approvals ADD COLUMN old_agent_id TEXT;
UPDATE standing_approvals SET old_agent_id = 'agent_' || agent_id;
ALTER TABLE standing_approvals DROP COLUMN agent_id;
ALTER TABLE standing_approvals RENAME COLUMN old_agent_id TO agent_id;
ALTER TABLE standing_approvals ALTER COLUMN agent_id SET NOT NULL;

ALTER TABLE agent_connectors ADD COLUMN old_agent_id TEXT;
UPDATE agent_connectors SET old_agent_id = 'agent_' || agent_id;
ALTER TABLE agent_connectors DROP COLUMN agent_id;
ALTER TABLE agent_connectors RENAME COLUMN old_agent_id TO agent_id;
ALTER TABLE agent_connectors ALTER COLUMN agent_id SET NOT NULL;

-- Restore composite PKs and FKs.
ALTER TABLE agent_connectors ADD PRIMARY KEY (agent_id, approver_id, connector_id);

ALTER TABLE approvals
    ADD CONSTRAINT approvals_agent_id_approver_id_fkey
    FOREIGN KEY (agent_id, approver_id) REFERENCES agents(agent_id, approver_id) ON DELETE CASCADE;

ALTER TABLE request_ids
    ADD CONSTRAINT request_ids_agent_id_approver_id_fkey
    FOREIGN KEY (agent_id, approver_id) REFERENCES agents(agent_id, approver_id) ON DELETE CASCADE;

ALTER TABLE standing_approvals
    ADD CONSTRAINT standing_approvals_agent_id_user_id_fkey
    FOREIGN KEY (agent_id, user_id) REFERENCES agents(agent_id, approver_id) ON DELETE CASCADE;

ALTER TABLE agent_connectors
    ADD CONSTRAINT agent_connectors_agent_id_approver_id_fkey
    FOREIGN KEY (agent_id, approver_id) REFERENCES agents(agent_id, approver_id) ON DELETE CASCADE;

CREATE INDEX idx_approvals_agent_status ON approvals (agent_id, status);
CREATE INDEX idx_standing_approvals_agent_action_status ON standing_approvals (agent_id, action_type, status);
