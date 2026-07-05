package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestUpdateStandingApprovalsConnectorInstanceBySourceConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	actionType := connID + ".scope"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Scope")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:                  actionType,
		SourceActionConfigurationID: &configID,
		Constraints:                 []byte(`{"ping":"*"}`),
	})

	inst, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("CreateAgentConnectorInstance: %v", err)
	}

	updated, err := db.UpdateStandingApprovalsConnectorInstanceBySourceConfig(
		ctx, tx, uid, configID, &inst.ConnectorInstanceID,
	)
	if err != nil {
		t.Fatalf("bulk update: %v", err)
	}
	if len(updated) != 1 {
		t.Fatalf("expected 1 updated row, got %d", len(updated))
	}
	if updated[0].ConnectorInstanceID == nil || *updated[0].ConnectorInstanceID != inst.ConnectorInstanceID {
		t.Fatalf("expected connector_instance_id %q, got %v", inst.ConnectorInstanceID, updated[0].ConnectorInstanceID)
	}

	updated, err = db.UpdateStandingApprovalsConnectorInstanceBySourceConfig(ctx, tx, uid, configID, nil)
	if err != nil {
		t.Fatalf("bulk clear: %v", err)
	}
	if len(updated) != 1 {
		t.Fatalf("expected 1 updated row on clear, got %d", len(updated))
	}
	if updated[0].ConnectorInstanceID != nil {
		t.Fatalf("expected nil connector_instance_id, got %v", updated[0].ConnectorInstanceID)
	}
}
