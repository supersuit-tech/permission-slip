package db

import "context"

// connectorInstanceCredentialsReadySQL is true when every required credential
// for the instance is satisfied by its own binding. Connectors with no
// required-credential rows are ready. Callers must alias the instance row as ac.
const connectorInstanceCredentialsReadySQL = `NOT EXISTS (
    SELECT 1 FROM connector_required_credentials crc
    WHERE crc.connector_id = ac.connector_id
      AND NOT (
          (crc.auth_type != 'oauth2' AND EXISTS (
              SELECT 1 FROM agent_connector_credentials acc
              INNER JOIN credentials cr ON cr.id = acc.credential_id
              WHERE acc.agent_id = ac.agent_id
                AND acc.connector_id = ac.connector_id
                AND acc.connector_instance_id = ac.connector_instance_id
                AND acc.approver_id = ac.approver_id
                AND cr.user_id = ac.approver_id
                AND (cr.service = crc.service OR cr.service = ac.connector_id)
          ))
          OR (crc.auth_type = 'oauth2' AND EXISTS (
              SELECT 1 FROM agent_connector_credentials acc
              INNER JOIN oauth_connections oc ON oc.id = acc.oauth_connection_id
              WHERE acc.agent_id = ac.agent_id
                AND acc.connector_id = ac.connector_id
                AND acc.connector_instance_id = ac.connector_instance_id
                AND acc.approver_id = ac.approver_id
                AND oc.user_id = ac.approver_id
                AND oc.provider = crc.oauth_provider
                AND oc.status = 'active'
                AND (SELECT COUNT(*) = 0 FROM json_each(crc.oauth_scopes) s WHERE NOT EXISTS (SELECT 1 FROM json_each(oc.scopes) os WHERE os.value = s.value))
          ))
      )
)`

// connectorInstanceDanglingSQL is true when the instance's binding points at an
// OAuth connection or static credential row that no longer exists.
// Callers must LEFT JOIN agent_connector_credentials acc, credentials cr, and
// oauth_connections oc.
const connectorInstanceDanglingSQL = `((acc.oauth_connection_id IS NOT NULL AND oc.id IS NULL) OR (acc.credential_id IS NOT NULL AND cr.id IS NULL))`

// connectorInstanceLiveBindingSQL is true when the instance has a binding whose
// target row still exists.
const connectorInstanceLiveBindingSQL = `(acc.id IS NOT NULL AND NOT ` + connectorInstanceDanglingSQL + `)`

// ConnectorInstanceEligibility is the credential state of one agent connector instance.
type ConnectorInstanceEligibility struct {
	ConnectorInstanceID string
	CredentialsReady    bool
	Dangling            bool
	HasLiveBinding      bool
}

// ConnectorInstanceVisible reports whether an instance should be listed to
// agents and considered for automatic selection.
//
// Instances whose bound OAuth connection or static credential was deleted are
// hidden. An instance with no binding is hidden when a sibling still has a
// live binding and this instance is not itself credential-ready — those rows
// are leftovers (the settings UI already drops them as orphans). A connector
// that requires no credentials stays visible even without a binding.
func ConnectorInstanceVisible(dangling, hasLiveBinding, credentialsReady, anyLiveBinding bool) bool {
	if dangling {
		return false
	}
	if !hasLiveBinding && anyLiveBinding && !credentialsReady {
		return false
	}
	return true
}

// ListConnectorInstanceEligibility returns credential state for every instance
// of one agent connector, including instances that should not be shown.
func ListConnectorInstanceEligibility(ctx context.Context, db DBTX, agentID int64, approverID, connectorID string) ([]ConnectorInstanceEligibility, error) {
	rows, err := db.Query(ctx, `
		SELECT ac.connector_instance_id,
		       `+connectorInstanceCredentialsReadySQL+`,
		       `+connectorInstanceDanglingSQL+`,
		       `+connectorInstanceLiveBindingSQL+`
		FROM agent_connectors ac
		LEFT JOIN agent_connector_credentials acc
		       ON acc.agent_id = ac.agent_id
		      AND acc.connector_id = ac.connector_id
		      AND acc.approver_id = ac.approver_id
		      AND acc.connector_instance_id = ac.connector_instance_id
		LEFT JOIN credentials cr ON cr.id = acc.credential_id
		LEFT JOIN oauth_connections oc ON oc.id = acc.oauth_connection_id
		WHERE ac.agent_id = $1 AND ac.approver_id = $2 AND ac.connector_id = $3
		ORDER BY ac.enabled_at ASC, ac.connector_instance_id ASC`,
		agentID, approverID, connectorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ConnectorInstanceEligibility
	for rows.Next() {
		var e ConnectorInstanceEligibility
		if err := rows.Scan(&e.ConnectorInstanceID, &e.CredentialsReady, &e.Dangling, &e.HasLiveBinding); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
