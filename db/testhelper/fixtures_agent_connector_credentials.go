package testhelper

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

func defaultConnectorInstanceID(t *testing.T, d db.DBTX, agentID int64, approverID, connectorID string) string {
	t.Helper()
	var instID string
	err := d.QueryRow(context.Background(),
		`SELECT connector_instance_id FROM agent_connectors
		 WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND is_default = 1
		 LIMIT 1`,
		agentID, approverID, connectorID,
	).Scan(&instID)
	if err != nil {
		t.Fatalf("lookup default connector instance: %v", err)
	}
	return instID
}

// InsertAgentConnectorCredential binds a static credential to an agent+connector pair.
// The agent_connectors row must already exist.
func InsertAgentConnectorCredential(t *testing.T, d db.DBTX, id string, agentID int64, approverID, connectorID, credentialID string) {
	t.Helper()
	instID := defaultConnectorInstanceID(t, d, agentID, approverID, connectorID)
	mustExec(t, d,
		`INSERT INTO agent_connector_credentials (id, agent_id, connector_id, approver_id, credential_id, connector_instance_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, agentID, connectorID, approverID, credentialID, instID)
}

// InsertAgentConnectorCredentialOAuth binds an OAuth connection to an agent+connector pair.
// The agent_connectors row must already exist.
func InsertAgentConnectorCredentialOAuth(t *testing.T, d db.DBTX, id string, agentID int64, approverID, connectorID, oauthConnectionID string) {
	t.Helper()
	instID := defaultConnectorInstanceID(t, d, agentID, approverID, connectorID)
	mustExec(t, d,
		`INSERT INTO agent_connector_credentials (id, agent_id, connector_id, approver_id, oauth_connection_id, connector_instance_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, agentID, connectorID, approverID, oauthConnectionID, instID)
}
