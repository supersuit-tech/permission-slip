package api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestApplyConnectorInstance_SkipsDanglingDefault(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", []string{"scope"})
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
	testhelper.InsertOAuthConnectionFull(t, tx, liveOAuth, uid, "google", testhelper.OAuthConnectionOpts{
		Scopes: []string{"scope"},
	})
	missingOAuth := "oc_deleted_connection"
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

	actionObj := map[string]json.RawMessage{"parameters": json.RawMessage(`{}`)}
	got, err := applyConnectorInstanceToAction(ctx, tx, &db.Agent{AgentID: agentID, ApproverID: uid}, connID+".search", actionObj)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if got != second.ConnectorInstanceID {
		t.Fatalf("expected ready instance %s, got %s", second.ConnectorInstanceID, got)
	}
}

func TestApplyConnectorInstance_ReadyDefaultStillWins(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", []string{"scope"})
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

	firstOAuth := testhelper.GenerateID(t, "oc_")
	secondOAuth := testhelper.GenerateID(t, "oc_")
	testhelper.InsertOAuthConnectionFull(t, tx, firstOAuth, uid, "google", testhelper.OAuthConnectionOpts{Scopes: []string{"scope"}})
	testhelper.InsertOAuthConnectionFull(t, tx, secondOAuth, uid, "google", testhelper.OAuthConnectionOpts{Scopes: []string{"scope"}})
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: def.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &firstOAuth,
	}); err != nil {
		t.Fatalf("bind default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: second.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &secondOAuth,
	}); err != nil {
		t.Fatalf("bind second: %v", err)
	}

	actionObj := map[string]json.RawMessage{"parameters": json.RawMessage(`{}`)}
	got, err := applyConnectorInstanceToAction(ctx, tx, &db.Agent{AgentID: agentID, ApproverID: uid}, connID+".search", actionObj)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if got != def.ConnectorInstanceID {
		t.Fatalf("expected ready default %s, got %s", def.ConnectorInstanceID, got)
	}
}
