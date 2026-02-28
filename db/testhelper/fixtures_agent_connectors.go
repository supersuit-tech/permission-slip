package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// InsertAgentConnector enables a connector for an agent.
// The agent and connector must already exist.
func InsertAgentConnector(t *testing.T, d db.DBTX, agentID int64, approverID, connectorID string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO agent_connectors (agent_id, approver_id, connector_id) VALUES ($1, $2, $3)`,
		agentID, approverID, connectorID)
}
