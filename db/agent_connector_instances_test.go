package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestListAgentConnectorInstances_DefaultOnly(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	instances, err := db.ListAgentConnectorInstances(t.Context(), tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("ListAgentConnectorInstances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(instances))
	}
	if !instances[0].IsDefault {
		t.Error("expected first instance to be default")
	}
}

func TestCreateAgentConnectorInstance_SecondInstance(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	inst2, err := db.CreateAgentConnectorInstance(t.Context(), tx, db.CreateAgentConnectorInstanceParams{
		AgentID:     agentID,
		ApproverID:  uid,
		ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("CreateAgentConnectorInstance: %v", err)
	}
	if inst2.IsDefault {
		t.Error("second instance should not be default")
	}

	instances, err := db.ListAgentConnectorInstances(t.Context(), tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("ListAgentConnectorInstances: %v", err)
	}
	if len(instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(instances))
	}
}

func TestCreateAgentConnectorInstance_RequiresConnectorEnabled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	// Connector exists in catalog but is not enabled for the agent.

	_, err := db.CreateAgentConnectorInstance(t.Context(), tx, db.CreateAgentConnectorInstanceParams{
		AgentID:     agentID,
		ApproverID:  uid,
		ConnectorID: connID,
	})
	var acErr *db.AgentConnectorError
	if !errors.As(err, &acErr) || acErr.Code != db.AgentConnectorErrConnectorNotEnabled {
		t.Fatalf("expected connector_not_enabled, got %v", err)
	}
}

func TestSetDefaultAgentConnectorInstance_SwitchesDefault(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	inst2, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID:     agentID,
		ApproverID:  uid,
		ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("CreateAgentConnectorInstance: %v", err)
	}

	def1, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || def1 == nil {
		t.Fatalf("default before: err=%v inst=%v", err, def1)
	}
	if !def1.IsDefault {
		t.Error("expected def1 is_default")
	}
	if def1.ConnectorInstanceID == inst2.ConnectorInstanceID {
		t.Fatal("second instance should not start as default")
	}

	updated, err := db.SetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID, inst2.ConnectorInstanceID)
	if err != nil {
		t.Fatalf("SetDefaultAgentConnectorInstance: %v", err)
	}
	if updated == nil || !updated.IsDefault {
		t.Fatalf("expected updated default: %+v", updated)
	}

	def2, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || def2 == nil {
		t.Fatalf("default after: err=%v inst=%v", err, def2)
	}
	if def2.ConnectorInstanceID != inst2.ConnectorInstanceID {
		t.Fatalf("expected second instance to become default, got %s", def2.ConnectorInstanceID)
	}
}

func TestResolveAgentConnectorInstance_AmbiguousDisplay(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	inst2, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("CreateAgentConnectorInstance: %v", err)
	}

	defInst, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || defInst == nil {
		t.Fatalf("default: %v", defInst)
	}

	cred1 := testhelper.GenerateID(t, "cred_")
	cred2 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, cred1, uid, connID, "Same", "00000000-0000-0000-0000-0000000000a1")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, cred2, uid, connID, "Same", "00000000-0000-0000-0000-0000000000a2")

	_, err = db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: defInst.ConnectorInstanceID, ApproverID: uid, CredentialID: &cred1,
	})
	if err != nil {
		t.Fatalf("bind default: %v", err)
	}
	_, err = db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: inst2.ConnectorInstanceID, ApproverID: uid, CredentialID: &cred2,
	})
	if err != nil {
		t.Fatalf("bind second: %v", err)
	}

	_, err = db.ResolveAgentConnectorInstance(ctx, tx, agentID, uid, connID, "Same")
	var instErr *db.AgentConnectorInstanceError
	if !errors.As(err, &instErr) || instErr.Code != db.AgentConnectorInstanceErrAmbiguousDisplay {
		t.Fatalf("expected ambiguous display error, got %v", err)
	}
	if len(instErr.AmbiguousInstanceIDs) != 2 {
		t.Fatalf("expected 2 ids, got %v", instErr.AmbiguousInstanceIDs)
	}
}

func TestFindActiveStandingApprovalsForAgent_FiltersByInstance(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	actionType := "instfilt.action"
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Act")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	inst2, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID:     agentID,
		ApproverID:  uid,
		ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("CreateAgentConnectorInstance: %v", err)
	}

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	saID := testhelper.GenerateID(t, "sa_")
	_, err = tx.Exec(ctx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, source_action_configuration_id, connector_instance_id, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, 'active', $5, $6::uuid, now(), now() + interval '30 days')`,
		saID, agentID, uid, actionType, configID, inst2.ConnectorInstanceID,
	)
	if err != nil {
		t.Fatalf("insert standing approval: %v", err)
	}

	all, err := db.FindActiveStandingApprovalsForAgent(ctx, tx, agentID, actionType, "")
	if err != nil {
		t.Fatalf("Find (no filter): %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 without instance filter, got %d", len(all))
	}

	wrong, err := db.FindActiveStandingApprovalsForAgent(ctx, tx, agentID, actionType, "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("Find (wrong instance): %v", err)
	}
	if len(wrong) != 0 {
		t.Fatalf("expected 0 for wrong instance, got %d", len(wrong))
	}

	match, err := db.FindActiveStandingApprovalsForAgent(ctx, tx, agentID, actionType, inst2.ConnectorInstanceID)
	if err != nil {
		t.Fatalf("Find (matching instance): %v", err)
	}
	if len(match) != 1 {
		t.Fatalf("expected 1 for matching instance, got %d", len(match))
	}
}

func TestDeleteAgentConnectorInstance_RevokesInstanceScopedStandingApproval(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	actionType := "delinst.action"
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Act")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	inst2, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID:     agentID,
		ApproverID:  uid,
		ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("CreateAgentConnectorInstance: %v", err)
	}

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	saID := testhelper.GenerateID(t, "sa_")
	_, err = tx.Exec(ctx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, source_action_configuration_id, connector_instance_id, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, 'active', $5, $6::uuid, now(), now() + interval '30 days')`,
		saID, agentID, uid, actionType, configID, inst2.ConnectorInstanceID,
	)
	if err != nil {
		t.Fatalf("insert standing approval: %v", err)
	}

	if err := db.DeleteAgentConnectorInstance(ctx, tx, agentID, uid, connID, inst2.ConnectorInstanceID); err != nil {
		t.Fatalf("DeleteAgentConnectorInstance: %v", err)
	}

	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM standing_approvals WHERE standing_approval_id = $1`, saID).Scan(&status)
	if err != nil {
		t.Fatalf("query sa: %v", err)
	}
	if status != "revoked" {
		t.Errorf("expected revoked, got %q", status)
	}
}
