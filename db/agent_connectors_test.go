package db_test

import (
	"context"
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

func TestEnableAgentConnector_Idempotent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	connID := testhelper.GenerateID(t, "conn_")

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "user_"+uid[:8])
	testhelper.InsertConnector(t, tx, connID)

	row1, err := db.EnableAgentConnector(t.Context(), tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("EnableAgentConnector first: %v", err)
	}
	row2, err := db.EnableAgentConnector(t.Context(), tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("EnableAgentConnector second: %v", err)
	}
	if row1.EnabledAt != row2.EnabledAt {
		t.Fatalf("expected same row on idempotent enable, got %v vs %v", row1.EnabledAt, row2.EnabledAt)
	}

	var n int
	if err := tx.QueryRow(t.Context(),
		`SELECT count(*) FROM agent_connectors WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3`,
		agentID, uid, connID,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 agent_connectors row, got %d", n)
	}
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
