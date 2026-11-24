package db_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestGetAgentCapabilities_OmitsDanglingOAuthInstance(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", []string{"scope"})
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	def, err := db.GetDefaultAgentConnectorInstance(t.Context(), tx, agentID, uid, connID)
	if err != nil || def == nil {
		t.Fatalf("default: %v", err)
	}
	second, err := db.CreateAgentConnectorInstance(t.Context(), tx, db.CreateAgentConnectorInstanceParams{
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
	if _, err := db.UpsertAgentConnectorCredentialByInstance(t.Context(), tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: def.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &missingOAuth,
	}); err != nil {
		t.Fatalf("bind dangling default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(t.Context(), tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: second.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &liveOAuth,
	}); err != nil {
		t.Fatalf("bind live sibling: %v", err)
	}

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 || !caps.Connectors[0].CredentialsReady {
		t.Fatalf("expected the connector to stay ready via the live instance, got %+v", caps.Connectors)
	}
	if len(caps.ConnectorInstances) != 1 {
		t.Fatalf("expected 1 visible instance, got %+v", caps.ConnectorInstances)
	}
	got := caps.ConnectorInstances[0]
	if got.ConnectorInstanceID != second.ConnectorInstanceID || !got.CredentialsReady {
		t.Fatalf("expected the live instance, got %+v", got)
	}
}

func TestGetAgentCapabilities_KeepsSoleUnboundInstance(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", []string{"scope"})
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.ConnectorInstances) != 1 {
		t.Fatalf("expected the unbound default to stay visible, got %+v", caps.ConnectorInstances)
	}
	if caps.ConnectorInstances[0].CredentialsReady {
		t.Fatal("expected credentials_ready=false")
	}
}

func TestGetAgentCapabilities_OmitsUnboundInstanceWhenSiblingIsLive(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", []string{"scope"})
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	second, err := db.CreateAgentConnectorInstance(t.Context(), tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}
	liveOAuth := testhelper.GenerateID(t, "oc_")
	testhelper.InsertOAuthConnectionFull(t, tx, liveOAuth, uid, "google", testhelper.OAuthConnectionOpts{
		Scopes: []string{"scope"},
	})
	// Default stays unbound (binding cascaded away). Sibling has the live account.
	if _, err := db.UpsertAgentConnectorCredentialByInstance(t.Context(), tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "acc_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: second.ConnectorInstanceID, ApproverID: uid, OAuthConnectionID: &liveOAuth,
	}); err != nil {
		t.Fatalf("bind sibling: %v", err)
	}

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.ConnectorInstances) != 1 || caps.ConnectorInstances[0].ConnectorInstanceID != second.ConnectorInstanceID {
		t.Fatalf("expected only the live sibling, got %+v", caps.ConnectorInstances)
	}
}
