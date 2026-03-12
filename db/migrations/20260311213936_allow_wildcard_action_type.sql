-- +goose Up

-- Drop composite FK so action_type = '*' (wildcard) can be stored without
-- a matching row in connector_actions.  Non-wildcard types are validated at
-- the application layer instead.
ALTER TABLE action_configurations
  DROP CONSTRAINT action_configurations_connector_id_action_type_fkey;

-- Enforce at most one wildcard config per agent+connector.
CREATE UNIQUE INDEX idx_action_config_wildcard_unique
  ON action_configurations (agent_id, connector_id)
  WHERE action_type = '*';

-- +goose Down

DROP INDEX IF EXISTS idx_action_config_wildcard_unique;

-- Remove wildcard configs that would violate the FK being restored.
DELETE FROM action_configurations WHERE action_type = '*';

ALTER TABLE action_configurations
  ADD CONSTRAINT action_configurations_connector_id_action_type_fkey
  FOREIGN KEY (connector_id, action_type)
  REFERENCES connector_actions(connector_id, action_type) ON DELETE CASCADE;
