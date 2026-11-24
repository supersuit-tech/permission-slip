package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestRepairDanglingConnectorInstances_PromotesSoleBoundSibling(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	def, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || def == nil {
		t.Fatalf("default: %v", err)
	}
	second, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}

	liveOAuth := testhelper.GenerateID(t, "oc_")
	testhelper.InsertOAuthConnection(t, tx, liveOAuth, uid, "google")
	missingOAuth := "oc_missing_not_in_table"
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: def.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &missingOAuth,
	}); err != nil {
		t.Fatalf("bind dangling default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: second.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &liveOAuth,
	}); err != nil {
		t.Fatalf("bind live sibling: %v", err)
	}

	if _, err := db.RepairDanglingConnectorInstances(ctx, tx); err != nil {
		t.Fatalf("repair: %v", err)
	}

	instances, err := db.ListAgentConnectorInstances(ctx, tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected 1 instance after repair, got %d", len(instances))
	}
	if instances[0].ConnectorInstanceID != second.ConnectorInstanceID {
		t.Fatalf("expected surviving instance %s, got %s", second.ConnectorInstanceID, instances[0].ConnectorInstanceID)
	}
	if !instances[0].IsDefault {
		t.Fatal("expected the live sibling to become the default")
	}
}

func TestRepairDanglingConnectorInstances_DropsStaleNonDefault(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	def, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || def == nil {
		t.Fatalf("default: %v", err)
	}
	second, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}

	liveOAuth := testhelper.GenerateID(t, "oc_")
	testhelper.InsertOAuthConnection(t, tx, liveOAuth, uid, "google")
	missingOAuth := "oc_missing_non_default"
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: def.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &liveOAuth,
	}); err != nil {
		t.Fatalf("bind default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: second.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &missingOAuth,
	}); err != nil {
		t.Fatalf("bind dangling sibling: %v", err)
	}

	if _, err := db.RepairDanglingConnectorInstances(ctx, tx); err != nil {
		t.Fatalf("repair: %v", err)
	}

	instances, err := db.ListAgentConnectorInstances(ctx, tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(instances) != 1 || instances[0].ConnectorInstanceID != def.ConnectorInstanceID || !instances[0].IsDefault {
		t.Fatalf("expected only the live default, got %+v", instances)
	}
}

func TestRepairDanglingConnectorInstances_KeepsSoleUnboundDefault(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	if _, err := db.RepairDanglingConnectorInstances(ctx, tx); err != nil {
		t.Fatalf("repair: %v", err)
	}
	instances, err := db.ListAgentConnectorInstances(ctx, tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(instances) != 1 || !instances[0].IsDefault {
		t.Fatalf("expected the unbound default to remain, got %+v", instances)
	}
}

func TestDeleteOAuthConnection_RemovesStaleInstanceAndPromotesSibling(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	def, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || def == nil {
		t.Fatalf("default: %v", err)
	}
	second, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}

	oldOAuth := testhelper.GenerateID(t, "oc_")
	newOAuth := testhelper.GenerateID(t, "oc_")
	testhelper.InsertOAuthConnection(t, tx, oldOAuth, uid, "google")
	testhelper.InsertOAuthConnection(t, tx, newOAuth, uid, "google")
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: def.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &oldOAuth,
	}); err != nil {
		t.Fatalf("bind old default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: second.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &newOAuth,
	}); err != nil {
		t.Fatalf("bind new sibling: %v", err)
	}

	if _, err := db.DeleteOAuthConnectionByID(ctx, tx, uid, oldOAuth); err != nil {
		t.Fatalf("delete oauth: %v", err)
	}

	gone, err := db.GetOAuthConnectionByID(ctx, tx, oldOAuth)
	if err != nil {
		t.Fatalf("lookup deleted oauth: %v", err)
	}
	if gone != nil {
		t.Fatal("expected old oauth connection to be deleted")
	}

	instances, err := db.ListAgentConnectorInstances(ctx, tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d (%+v)", len(instances), instances)
	}
	if instances[0].ConnectorInstanceID != second.ConnectorInstanceID || !instances[0].IsDefault {
		t.Fatalf("expected promoted sibling, got %+v", instances[0])
	}
	binding, err := db.GetAgentConnectorCredentialByInstance(ctx, tx, agentID, connID, second.ConnectorInstanceID)
	if err != nil || binding == nil || binding.OAuthConnectionID == nil || *binding.OAuthConnectionID != newOAuth {
		t.Fatalf("expected sibling to stay bound to the live connection, got %+v err=%v", binding, err)
	}
}

func TestDeleteOAuthConnection_DropsNonDefaultInstance(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	def, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || def == nil {
		t.Fatalf("default: %v", err)
	}
	second, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}

	keepOAuth := testhelper.GenerateID(t, "oc_")
	dropOAuth := testhelper.GenerateID(t, "oc_")
	testhelper.InsertOAuthConnection(t, tx, keepOAuth, uid, "google")
	testhelper.InsertOAuthConnection(t, tx, dropOAuth, uid, "google")
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: def.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &keepOAuth,
	}); err != nil {
		t.Fatalf("bind default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: second.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &dropOAuth,
	}); err != nil {
		t.Fatalf("bind second: %v", err)
	}

	if _, err := db.DeleteOAuthConnectionByID(ctx, tx, uid, dropOAuth); err != nil {
		t.Fatalf("delete oauth: %v", err)
	}

	instances, err := db.ListAgentConnectorInstances(ctx, tx, agentID, uid, connID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(instances) != 1 || instances[0].ConnectorInstanceID != def.ConnectorInstanceID || !instances[0].IsDefault {
		t.Fatalf("expected the original default to remain, got %+v", instances)
	}
}

func TestOAuthDeleteTrigger_ClearsBinding(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	second, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}
	oauthID := testhelper.GenerateID(t, "oc_")
	testhelper.InsertOAuthConnection(t, tx, oauthID, uid, "google")
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: second.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &oauthID,
	}); err != nil {
		t.Fatalf("bind: %v", err)
	}

	// Raw delete bypasses DeleteOAuthConnectionByID. The trigger must still
	// drop the binding so the instance cannot keep a dead oauth id.
	testhelper.MustExec(t, tx, `DELETE FROM oauth_connections WHERE id = $1`, oauthID)

	binding, err := db.GetAgentConnectorCredentialByInstance(ctx, tx, agentID, connID, second.ConnectorInstanceID)
	if err != nil {
		t.Fatalf("lookup binding: %v", err)
	}
	if binding != nil {
		t.Fatalf("expected trigger to delete the binding, got %+v", binding)
	}
}
