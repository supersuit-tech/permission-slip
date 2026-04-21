package db_test

import (
	"fmt"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestAgentConnectorsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.RequireColumns(t, tx, "agent_connectors", []string{
		"agent_id", "approver_id", "connector_id", "connector_instance_id",
		"is_default", "enabled_at",
	})
}

func TestAgentConnectorsIndex(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.RequireIndex(t, tx, "agent_connectors", "idx_agent_connectors_connector")
}

func TestAgentConnectorsCascadeOnAgentDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	connID := testhelper.GenerateID(t, "conn_")

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "user_"+uid[:8])
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	testhelper.RequireCascadeDeletes(t, tx,
		fmt.Sprintf("DELETE FROM agents WHERE agent_id = %d", agentID),
		[]string{"agent_connectors"},
		fmt.Sprintf("agent_id = %d", agentID),
	)
}

func TestAgentConnectorsCascadeOnConnectorDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	connID := testhelper.GenerateID(t, "conn_")

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "user_"+uid[:8])
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM connectors WHERE id = '"+connID+"'",
		[]string{"agent_connectors"},
		"connector_id = '"+connID+"'",
	)
}
