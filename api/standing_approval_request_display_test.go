package api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestResolveStandingApprovalRequestDisplay_SingleInstance(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := "protonmail"
	testhelper.InsertConnector(t, tx, connID)
	actionType := "protonmail.read_email"
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Read email")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	display := resolveStandingApprovalRequestDisplay(ctx, tx, agentID, uid, actionType, &configID)
	if display.ConnectorName == "" {
		t.Error("expected connector name")
	}
	if display.ConnectorInstanceDisplay != "" {
		t.Errorf("single default instance should not set instance display, got %q", display.ConnectorInstanceDisplay)
	}
}

func TestResolveStandingApprovalRequestDisplay_PinnedInstance(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	actionType := connID + ".ping"
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Ping")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	inst2, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID:     agentID,
		ApproverID:  uid,
		ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("CreateAgentConnectorInstance: %v", err)
	}

	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID, uid, connID, "Sales", "00000000-0000-0000-0000-000000000099")
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: inst2.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID,
	}); err != nil {
		t.Fatalf("bind sales instance: %v", err)
	}

	configID := testhelper.GenerateID(t, "ac_")
	params, _ := json.Marshal(map[string]string{
		"connector_instance": inst2.ConnectorInstanceID,
	})
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, actionType, testhelper.ActionConfigOpts{
		Parameters: params,
		Name:       "Pinned instance",
	})

	display := resolveStandingApprovalRequestDisplay(ctx, tx, agentID, uid, actionType, &configID)
	if display.ConnectorInstanceDisplay != "Sales" {
		t.Fatalf("expected Sales, got %q", display.ConnectorInstanceDisplay)
	}
}

func TestFixedConnectorInstanceSelector(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]string{"connector_instance": "sales"})
	if got := fixedConnectorInstanceSelector(raw); got != "sales" {
		t.Fatalf("got %q", got)
	}

	wildcard, _ := json.Marshal(map[string]string{"connector_instance": "*"})
	if got := fixedConnectorInstanceSelector(wildcard); got != "" {
		t.Fatalf("wildcard should not resolve, got %q", got)
	}
}
