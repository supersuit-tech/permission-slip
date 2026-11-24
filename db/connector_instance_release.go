package db

import (
	"context"
	"fmt"
)

// bindingColumn identifies which agent_connector_credentials column a release
// targets. Only these two values are accepted; they are interpolated into SQL.
const (
	bindingColumnOAuth      = "oauth_connection_id"
	bindingColumnCredential = "credential_id"
)

type connectorBindingRef struct {
	agentID     int64
	approverID  string
	connectorID string
	instanceID  string
	isDefault   bool
}

// releaseConnectorInstancesForBinding detaches agent connector instances from
// a credential or OAuth connection that is about to be deleted.
//
// Non-default instances bound to the target are deleted (their standing
// approvals are revoked). The default instance is unbound. When that leaves
// exactly one sibling with a live binding, that sibling becomes the default
// and the empty former default is removed.
//
// Call this before deleting the target row, inside the same transaction, so
// the binding is still visible. SQLite has no foreign key from the binding to
// oauth_connections or credentials, so the database will not do this itself.
func releaseConnectorInstancesForBinding(ctx context.Context, d DBTX, column, targetID string) error {
	if column != bindingColumnOAuth && column != bindingColumnCredential {
		return fmt.Errorf("release connector instances: unsupported column %q", column)
	}

	rows, err := d.Query(ctx, `
		SELECT acc.agent_id, acc.approver_id, acc.connector_id, acc.connector_instance_id, ac.is_default
		FROM agent_connector_credentials acc
		JOIN agent_connectors ac
		  ON ac.agent_id = acc.agent_id
		 AND ac.approver_id = acc.approver_id
		 AND ac.connector_id = acc.connector_id
		 AND ac.connector_instance_id = acc.connector_instance_id
		WHERE acc.`+column+` = $1`, targetID)
	if err != nil {
		return fmt.Errorf("list connector instances for %s: %w", column, err)
	}

	var refs []connectorBindingRef
	for rows.Next() {
		var ref connectorBindingRef
		if err := rows.Scan(&ref.agentID, &ref.approverID, &ref.connectorID, &ref.instanceID, &ref.isDefault); err != nil {
			rows.Close()
			return fmt.Errorf("scan connector instance for %s: %w", column, err)
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	// Close before further queries. SQLite will not run the next statement on
	// this transaction while this result set is still open.
	if err := rows.Close(); err != nil {
		return err
	}

	for _, ref := range refs {
		if ref.isDefault {
			continue
		}
		if err := DeleteAgentConnectorInstance(ctx, d, ref.agentID, ref.approverID, ref.connectorID, ref.instanceID); err != nil {
			return fmt.Errorf("delete connector instance %s: %w", ref.instanceID, err)
		}
	}

	var defaults []connectorBindingRef
	for _, ref := range refs {
		if !ref.isDefault {
			continue
		}
		if _, err := DeleteAgentConnectorCredentialByInstance(ctx, d, ref.agentID, ref.approverID, ref.connectorID, ref.instanceID); err != nil {
			return fmt.Errorf("unbind default connector instance %s: %w", ref.instanceID, err)
		}
		defaults = append(defaults, ref)
	}

	for _, ref := range defaults {
		if err := promoteSoleBoundSibling(ctx, d, ref); err != nil {
			return err
		}
	}
	return nil
}

// promoteSoleBoundSibling makes the only remaining bound sibling the default
// and deletes the unbound former default. No-op when zero or multiple siblings
// still have a live binding.
func promoteSoleBoundSibling(ctx context.Context, d DBTX, former connectorBindingRef) error {
	rows, err := d.Query(ctx, `
		SELECT ac.connector_instance_id,
		       CASE WHEN EXISTS (
		           SELECT 1 FROM agent_connector_credentials acc
		           LEFT JOIN oauth_connections oc ON oc.id = acc.oauth_connection_id
		           LEFT JOIN credentials cr ON cr.id = acc.credential_id
		           WHERE acc.agent_id = ac.agent_id
		             AND acc.approver_id = ac.approver_id
		             AND acc.connector_id = ac.connector_id
		             AND acc.connector_instance_id = ac.connector_instance_id
		             AND (
		                 (acc.oauth_connection_id IS NOT NULL AND oc.id IS NOT NULL)
		                 OR (acc.credential_id IS NOT NULL AND cr.id IS NOT NULL)
		             )
		       ) THEN 1 ELSE 0 END
		FROM agent_connectors ac
		WHERE ac.agent_id = $1 AND ac.approver_id = $2 AND ac.connector_id = $3
		  AND ac.connector_instance_id <> $4`,
		former.agentID, former.approverID, former.connectorID, former.instanceID,
	)
	if err != nil {
		return fmt.Errorf("list sibling connector instances: %w", err)
	}

	var live []string
	for rows.Next() {
		var id string
		var isLive int
		if err := rows.Scan(&id, &isLive); err != nil {
			rows.Close()
			return fmt.Errorf("scan sibling connector instance: %w", err)
		}
		if isLive == 1 {
			live = append(live, id)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(live) != 1 {
		return nil
	}

	promoted, err := SetDefaultAgentConnectorInstance(ctx, d, former.agentID, former.approverID, former.connectorID, live[0])
	if err != nil {
		return fmt.Errorf("promote connector instance %s: %w", live[0], err)
	}
	if promoted == nil {
		return fmt.Errorf("promote connector instance %s: not found", live[0])
	}
	if err := DeleteAgentConnectorInstance(ctx, d, former.agentID, former.approverID, former.connectorID, former.instanceID); err != nil {
		return fmt.Errorf("delete unbound former default %s: %w", former.instanceID, err)
	}
	return nil
}
