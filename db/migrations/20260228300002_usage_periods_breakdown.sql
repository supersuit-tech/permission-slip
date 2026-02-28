-- +goose Up

-- Add JSONB breakdown column for per-agent, per-connector, and per-action-type
-- usage analytics within a billing period.
-- Structure: { "by_agent": { "<agent_id>": N }, "by_connector": { "<connector_id>": N }, "by_action_type": { "<type>": N } }
ALTER TABLE usage_periods
    ADD COLUMN breakdown jsonb NOT NULL DEFAULT '{}';

-- +goose Down

ALTER TABLE usage_periods DROP COLUMN IF EXISTS breakdown;
