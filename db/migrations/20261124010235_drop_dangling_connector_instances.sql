-- +goose Up
-- Remove connector instances left behind after an OAuth connection or static
-- credential is deleted. ON DELETE CASCADE drops the binding row but leaves
-- the agent_connectors instance, which then shows up with no display name and
-- can still be chosen as the default.

DELETE FROM agent_connector_credentials
WHERE (oauth_connection_id IS NOT NULL AND oauth_connection_id NOT IN (SELECT id FROM oauth_connections))
   OR (credential_id IS NOT NULL AND credential_id NOT IN (SELECT id FROM credentials));

UPDATE agent_connectors
SET is_default = false
WHERE is_default
  AND NOT EXISTS (
    SELECT 1 FROM agent_connector_credentials acc
    WHERE acc.agent_id = agent_connectors.agent_id
      AND acc.approver_id = agent_connectors.approver_id
      AND acc.connector_id = agent_connectors.connector_id
      AND acc.connector_instance_id = agent_connectors.connector_instance_id
  )
  AND (
    SELECT COUNT(*) FROM agent_connectors sib
    WHERE sib.agent_id = agent_connectors.agent_id
      AND sib.approver_id = agent_connectors.approver_id
      AND sib.connector_id = agent_connectors.connector_id
      AND sib.connector_instance_id <> agent_connectors.connector_instance_id
      AND EXISTS (
        SELECT 1 FROM agent_connector_credentials acc
        WHERE acc.agent_id = sib.agent_id
          AND acc.approver_id = sib.approver_id
          AND acc.connector_id = sib.connector_id
          AND acc.connector_instance_id = sib.connector_instance_id
      )
  ) = 1;

UPDATE agent_connectors
SET is_default = true
WHERE NOT is_default
  AND EXISTS (
    SELECT 1 FROM agent_connector_credentials acc
    WHERE acc.agent_id = agent_connectors.agent_id
      AND acc.approver_id = agent_connectors.approver_id
      AND acc.connector_id = agent_connectors.connector_id
      AND acc.connector_instance_id = agent_connectors.connector_instance_id
  )
  AND NOT EXISTS (
    SELECT 1 FROM agent_connectors d
    WHERE d.agent_id = agent_connectors.agent_id
      AND d.approver_id = agent_connectors.approver_id
      AND d.connector_id = agent_connectors.connector_id
      AND d.is_default
  )
  AND (
    SELECT COUNT(*) FROM agent_connectors sib
    WHERE sib.agent_id = agent_connectors.agent_id
      AND sib.approver_id = agent_connectors.approver_id
      AND sib.connector_id = agent_connectors.connector_id
      AND EXISTS (
        SELECT 1 FROM agent_connector_credentials acc
        WHERE acc.agent_id = sib.agent_id
          AND acc.approver_id = sib.approver_id
          AND acc.connector_id = sib.connector_id
          AND acc.connector_instance_id = sib.connector_instance_id
      )
  ) = 1;

UPDATE standing_approvals
SET status = 'revoked', revoked_at = now()
WHERE status = 'active'
  AND connector_instance_id IS NOT NULL
  AND EXISTS (
    SELECT 1 FROM agent_connectors ac
    WHERE ac.agent_id = standing_approvals.agent_id
      AND ac.approver_id = standing_approvals.user_id
      AND ac.connector_instance_id = standing_approvals.connector_instance_id
      AND NOT ac.is_default
      AND NOT EXISTS (
        SELECT 1 FROM agent_connector_credentials acc
        WHERE acc.agent_id = ac.agent_id
          AND acc.approver_id = ac.approver_id
          AND acc.connector_id = ac.connector_id
          AND acc.connector_instance_id = ac.connector_instance_id
      )
  );

DELETE FROM agent_connectors
WHERE NOT is_default
  AND NOT EXISTS (
    SELECT 1 FROM agent_connector_credentials acc
    WHERE acc.agent_id = agent_connectors.agent_id
      AND acc.approver_id = agent_connectors.approver_id
      AND acc.connector_id = agent_connectors.connector_id
      AND acc.connector_instance_id = agent_connectors.connector_instance_id
  );

-- +goose Down
-- Data cleanup is not reversed. Bindings are removed by ON DELETE CASCADE.
SELECT 1;
