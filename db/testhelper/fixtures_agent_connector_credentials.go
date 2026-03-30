package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

// InsertAgentConnectorCredential binds a static credential to an agent+connector pair.
// The agent_connectors row must already exist.
func InsertAgentConnectorCredential(t *testing.T, d db.DBTX, id string, agentID int64, approverID, connectorID, credentialID string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO agent_connector_credentials (id, agent_id, connector_id, approver_id, credential_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, agentID, connectorID, approverID, credentialID)
}

// InsertAgentConnectorCredentialOAuth binds an OAuth connection to an agent+connector pair.
// The agent_connectors row must already exist.
func InsertAgentConnectorCredentialOAuth(t *testing.T, d db.DBTX, id string, agentID int64, approverID, connectorID, oauthConnectionID string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO agent_connector_credentials (id, agent_id, connector_id, approver_id, oauth_connection_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, agentID, connectorID, approverID, oauthConnectionID)
}
