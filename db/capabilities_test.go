package db_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestGetAgentCapabilities_NoConnectors(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if caps == nil {
		t.Fatal("expected non-nil result")
	}
	if len(caps.Connectors) != 0 {
		t.Errorf("expected 0 connectors, got %d", len(caps.Connectors))
	}
	if len(caps.Actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(caps.Actions))
	}
	if len(caps.StandingApprovals) != 0 {
		t.Errorf("expected 0 standing approvals, got %d", len(caps.StandingApprovals))
	}
}

func TestGetAgentCapabilities_MultipleConnectors(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Connector 1: with credentials stored and bound to this agent+connector
	conn1 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, conn1, "Gmail", "Send and manage emails")
	svc1 := testhelper.GenerateID(t, "svc_")
	testhelper.InsertConnectorRequiredCredential(t, tx, conn1, svc1, "api_key")
	cred1 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, cred1, uid, svc1)

	riskLow := "low"
	schema := json.RawMessage(`{"type":"object","required":["to","subject"]}`)
	sendDesc := "Send an email"
	testhelper.InsertConnectorActionFull(t, tx, conn1, "email.send", "Send Email", testhelper.ConnectorActionOpts{
		Description:      &sendDesc,
		RiskLevel:        &riskLow,
		ParametersSchema: schema,
	})
	readDesc := "Read emails"
	testhelper.InsertConnectorActionFull(t, tx, conn1, "email.read", "Read Email", testhelper.ConnectorActionOpts{
		Description: &readDesc,
		RiskLevel:   &riskLow,
	})

	// Connector 2: no credentials
	conn2 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, conn2, "Stripe", "Payment processing")
	svc2 := testhelper.GenerateID(t, "svc_")
	testhelper.InsertConnectorRequiredCredential(t, tx, conn2, svc2, "api_key")

	riskHigh := "high"
	chargeDesc := "Charge a payment"
	testhelper.InsertConnectorActionFull(t, tx, conn2, "payment.charge", "Charge Payment", testhelper.ConnectorActionOpts{
		Description: &chargeDesc,
		RiskLevel:   &riskHigh,
	})

	// Enable both connectors for the agent
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn1)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn2)
	testhelper.InsertAgentConnectorCredential(t, tx, testhelper.GenerateID(t, "acc_"), agentID, uid, conn1, cred1)

	// Add a standing approval for email.send
	maxExec := 100
	constraints := json.RawMessage(`{"recipient_pattern":"*@mycompany.com"}`)
	testhelper.InsertStandingApprovalFull(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:    "email.send",
		Constraints:   constraints,
		MaxExecutions: &maxExec,
	})

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}

	// Verify connectors (ordered by connector ID)
	if len(caps.Connectors) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(caps.Connectors))
	}

	// Build a map for order-independent assertions (generated IDs aren't lexically predictable)
	connByID := make(map[string]db.CapabilityConnector, 2)
	for _, c := range caps.Connectors {
		connByID[c.ID] = c
	}

	c1 := connByID[conn1]
	if c1.Name != "Gmail" {
		t.Errorf("expected name=Gmail, got %q", c1.Name)
	}
	if c1.Description == nil || *c1.Description != "Send and manage emails" {
		t.Errorf("expected description='Send and manage emails', got %v", c1.Description)
	}
	if !c1.CredentialsReady {
		t.Error("expected conn1 credentials_ready=true")
	}

	c2 := connByID[conn2]
	if c2.CredentialsReady {
		t.Error("expected conn2 credentials_ready=false")
	}

	// Verify actions
	if len(caps.Actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(caps.Actions))
	}

	// Build a map keyed by (connector_id, action_type) for order-independent assertions
	type actionKey struct{ connID, actionType string }
	actionByKey := make(map[actionKey]db.CapabilityAction, 3)
	for _, a := range caps.Actions {
		actionByKey[actionKey{a.ConnectorID, a.ActionType}] = a
	}

	emailSend := actionByKey[actionKey{conn1, "email.send"}]
	if emailSend.Name != "Send Email" {
		t.Errorf("expected name='Send Email', got %q", emailSend.Name)
	}
	if emailSend.Description == nil || *emailSend.Description != "Send an email" {
		t.Errorf("expected description='Send an email', got %v", emailSend.Description)
	}
	if emailSend.RiskLevel == nil || *emailSend.RiskLevel != "low" {
		t.Errorf("expected risk_level=low, got %v", emailSend.RiskLevel)
	}
	if emailSend.ParametersSchema == nil {
		t.Error("expected parameters_schema to be non-nil")
	}

	if _, ok := actionByKey[actionKey{conn1, "email.read"}]; !ok {
		t.Error("expected email.read action for conn1")
	}
	if _, ok := actionByKey[actionKey{conn2, "payment.charge"}]; !ok {
		t.Error("expected payment.charge action for conn2")
	}

	// Verify standing approvals
	if len(caps.StandingApprovals) != 1 {
		t.Fatalf("expected 1 standing approval, got %d", len(caps.StandingApprovals))
	}
	sa := caps.StandingApprovals[0]
	if sa.ActionType != "email.send" {
		t.Errorf("expected action_type=email.send, got %q", sa.ActionType)
	}
	if sa.MaxExecutions == nil || *sa.MaxExecutions != 100 {
		t.Errorf("expected max_executions=100, got %v", sa.MaxExecutions)
	}
	if sa.ExecutionsRemaining == nil || *sa.ExecutionsRemaining != 100 {
		t.Errorf("expected executions_remaining=100, got %v", sa.ExecutionsRemaining)
	}
	if sa.Constraints == nil {
		t.Error("expected constraints to be non-nil")
	}
}

func TestGetAgentCapabilities_StandingApprovals_ExecutionsRemaining(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	// Standing approval with max_executions=10 and 3 executions used
	maxExec := 10
	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:    "test.action",
		MaxExecutions: &maxExec,
	})
	// Record 3 executions
	for range 3 {
		testhelper.InsertStandingApprovalExecution(t, tx, saID)
	}

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.StandingApprovals) != 1 {
		t.Fatalf("expected 1 standing approval, got %d", len(caps.StandingApprovals))
	}
	sa := caps.StandingApprovals[0]
	if sa.ExecutionsRemaining == nil || *sa.ExecutionsRemaining != 7 {
		t.Errorf("expected executions_remaining=7, got %v", sa.ExecutionsRemaining)
	}
}

func TestGetAgentCapabilities_StandingApprovals_Unlimited(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	// Standing approval with no max_executions (unlimited)
	testhelper.InsertStandingApprovalFull(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:    "test.action",
		MaxExecutions: nil,
	})

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.StandingApprovals) != 1 {
		t.Fatalf("expected 1 standing approval, got %d", len(caps.StandingApprovals))
	}
	sa := caps.StandingApprovals[0]
	if sa.MaxExecutions != nil {
		t.Errorf("expected max_executions=nil (unlimited), got %v", sa.MaxExecutions)
	}
	if sa.ExecutionsRemaining != nil {
		t.Errorf("expected executions_remaining=nil (unlimited), got %v", sa.ExecutionsRemaining)
	}
}

func TestGetAgentCapabilities_ExcludesExpiredAndRevokedApprovals(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	// Active and current (should be included)
	testhelper.InsertStandingApprovalFull(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, testhelper.StandingApprovalOpts{
		ActionType: "test.action",
		Status:     "active",
	})

	// Revoked (should be excluded)
	testhelper.InsertStandingApprovalFull(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, testhelper.StandingApprovalOpts{
		ActionType: "test.action",
		Status:     "revoked",
	})

	// Expired status (should be excluded)
	testhelper.InsertStandingApprovalFull(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, testhelper.StandingApprovalOpts{
		ActionType: "test.action",
		Status:     "expired",
	})

	// Active but expires_at in the past (should be excluded)
	testhelper.InsertStandingApprovalFull(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, testhelper.StandingApprovalOpts{
		ActionType: "test.action",
		Status:     "active",
		StartsAt:   time.Now().Add(-48 * time.Hour),
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
	})

	// Active but starts_at in the future (should be excluded)
	testhelper.InsertStandingApprovalFull(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, testhelper.StandingApprovalOpts{
		ActionType: "test.action",
		Status:     "active",
		StartsAt:   time.Now().Add(1 * time.Hour),
		ExpiresAt:  time.Now().Add(48 * time.Hour),
	})

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.StandingApprovals) != 1 {
		t.Errorf("expected 1 active standing approval, got %d", len(caps.StandingApprovals))
	}
}

func TestGetAgentCapabilities_CredentialsReady_NoRequiredCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Connector with no required credentials — should be credentials_ready=true
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(caps.Connectors))
	}
	if !caps.Connectors[0].CredentialsReady {
		t.Error("expected credentials_ready=true for connector with no required credentials")
	}
}

func TestGetAgentCapabilities_CredentialsReady_UnboundGlobalCredentialNotReady(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// User has a matching global credential but no agent_connector_credentials binding.
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, connID, "API", "Needs bound credential")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "svc_a", "api_key")
	testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid, "svc_a")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(caps.Connectors))
	}
	if caps.Connectors[0].CredentialsReady {
		t.Error("expected credentials_ready=false when credential exists but is not bound to this agent+connector")
	}
}

func TestGetAgentCapabilities_CredentialsReady_BoundCredentialServiceNameMismatch(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Required row uses an internal service label; stored credential uses connector id
	// (same rule as assign-credential API: cred.service must match connector id).
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "google", "api_key")
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	testhelper.InsertAgentConnectorCredential(t, tx, testhelper.GenerateID(t, "acc_"), agentID, uid, connID, credID)

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(caps.Connectors))
	}
	if !caps.Connectors[0].CredentialsReady {
		t.Error("expected credentials_ready=true when bound credential service matches connector id while required row uses a different service label")
	}
}

func TestGetAgentCapabilities_ScopedToAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agent1 := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	agent2 := testhelper.InsertAgent(t, tx, uid) // second agent, same user

	conn1 := testhelper.GenerateID(t, "conn_")
	conn2 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, conn1)
	testhelper.InsertConnector(t, tx, conn2)
	testhelper.InsertAgentConnector(t, tx, agent1, uid, conn1)
	testhelper.InsertAgentConnector(t, tx, agent2, uid, conn2)

	// Agent 1 should only see conn1
	caps, err := db.GetAgentCapabilities(t.Context(), tx, agent1, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector for agent1, got %d", len(caps.Connectors))
	}
	if caps.Connectors[0].ID != conn1 {
		t.Errorf("expected %s, got %q", conn1, caps.Connectors[0].ID)
	}

	// Agent 2 should only see conn2
	caps, err = db.GetAgentCapabilities(t.Context(), tx, agent2, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector for agent2, got %d", len(caps.Connectors))
	}
	if caps.Connectors[0].ID != conn2 {
		t.Errorf("expected %s, got %q", conn2, caps.Connectors[0].ID)
	}
}

func TestGetAgentCapabilities_CredentialsReady_OAuth2Connected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Connector requiring OAuth2 credentials (e.g., Google)
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, connID, "Google Drive", "Upload files to Google Drive")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google_drive", "google", []string{"https://www.googleapis.com/auth/drive"})

	// User has an active OAuth connection for Google with the required scope
	ocID := testhelper.GenerateID(t, "oc_")
	testhelper.InsertOAuthConnectionFull(t, tx, ocID, uid, "google", testhelper.OAuthConnectionOpts{
		Scopes: []string{"https://www.googleapis.com/auth/drive"},
	})

	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	testhelper.InsertAgentConnectorCredentialOAuth(t, tx, testhelper.GenerateID(t, "acc_"), agentID, uid, connID, ocID)

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(caps.Connectors))
	}
	if !caps.Connectors[0].CredentialsReady {
		t.Error("expected credentials_ready=true when bound OAuth2 connection is active with required scopes")
	}
}

func TestGetAgentCapabilities_CredentialsReady_OAuth2InsufficientScopes(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Connector requires Drive scope
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google_drive", "google", []string{"https://www.googleapis.com/auth/drive"})

	// User connected Google but only with email scope (insufficient)
	ocID := testhelper.GenerateID(t, "oc_")
	testhelper.InsertOAuthConnectionFull(t, tx, ocID, uid, "google", testhelper.OAuthConnectionOpts{
		Scopes: []string{"https://www.googleapis.com/auth/gmail.readonly"},
	})

	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	testhelper.InsertAgentConnectorCredentialOAuth(t, tx, testhelper.GenerateID(t, "acc_"), agentID, uid, connID, ocID)

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(caps.Connectors))
	}
	if caps.Connectors[0].CredentialsReady {
		t.Error("expected credentials_ready=false when bound OAuth2 connection has insufficient scopes")
	}
}

func TestGetAgentCapabilities_CredentialsReady_OAuth2NotConnected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Connector requiring OAuth2 credentials, but no OAuth connection exists
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google_drive", "google", []string{"https://www.googleapis.com/auth/drive"})

	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(caps.Connectors))
	}
	if caps.Connectors[0].CredentialsReady {
		t.Error("expected credentials_ready=false when OAuth2 connection is missing")
	}
}

func TestGetAgentCapabilities_CredentialsReady_OAuth2Revoked(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Connector requiring OAuth2, user has a revoked connection
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google_drive", "google", []string{"https://www.googleapis.com/auth/drive"})

	ocID := testhelper.GenerateID(t, "oc_")
	testhelper.InsertOAuthConnectionFull(t, tx, ocID, uid, "google", testhelper.OAuthConnectionOpts{
		Status: "revoked",
		Scopes: []string{},
	})

	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	testhelper.InsertAgentConnectorCredentialOAuth(t, tx, testhelper.GenerateID(t, "acc_"), agentID, uid, connID, ocID)

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(caps.Connectors))
	}
	if caps.Connectors[0].CredentialsReady {
		t.Error("expected credentials_ready=false when bound OAuth2 connection is revoked")
	}
}

func TestGetAgentCapabilities_CrossUserIsolation(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agent1 := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	_ = testhelper.InsertUserWithAgent(t, tx, uid2, "u2_"+uid2[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agent1, uid1, connID)

	// User 2 should not see user 1's agent's connectors
	caps, err := db.GetAgentCapabilities(t.Context(), tx, agent1, uid2)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 0 {
		t.Errorf("expected 0 connectors for wrong user, got %d", len(caps.Connectors))
	}
}

func TestGetAgentCapabilities_StandingApprovalOnlyForCorrectAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agent1 := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	agent2 := testhelper.InsertAgent(t, tx, uid)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "scoped.action", "Scoped")
	testhelper.InsertAgentConnector(t, tx, agent1, uid, connID)
	testhelper.InsertAgentConnector(t, tx, agent2, uid, connID)

	// Standing approval only for agent1
	testhelper.InsertStandingApprovalFull(t, tx, testhelper.GenerateID(t, "sa_"), agent1, uid, testhelper.StandingApprovalOpts{
		ActionType: "scoped.action",
	})

	// Agent 1 should see the standing approval
	caps, err := db.GetAgentCapabilities(t.Context(), tx, agent1, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities for agent1: %v", err)
	}
	if len(caps.StandingApprovals) != 1 {
		t.Errorf("expected 1 standing approval for agent1, got %d", len(caps.StandingApprovals))
	}

	// Agent 2 should not see agent 1's standing approval
	caps, err = db.GetAgentCapabilities(t.Context(), tx, agent2, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities for agent2: %v", err)
	}
	if len(caps.StandingApprovals) != 0 {
		t.Errorf("expected 0 standing approvals for agent2, got %d", len(caps.StandingApprovals))
	}
}

func TestGetAgentCapabilities_CredentialsReady_MultiCRCAllSatisfiedViaConnectorID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Connector with two required credential rows (different services, same auth type).
	// A single bound credential whose service = connector ID should satisfy both,
	// because the query matches on (cr.service = crc.service OR cr.service = c.id).
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "svc_alpha", "api_key")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "svc_beta", "api_key")

	// Credential with service = connector ID (covers all CRC rows via the OR fallback).
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	testhelper.InsertAgentConnectorCredential(t, tx, testhelper.GenerateID(t, "acc_"), agentID, uid, connID, credID)

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(caps.Connectors))
	}
	if !caps.Connectors[0].CredentialsReady {
		t.Error("expected credentials_ready=true when a single credential (service=connector_id) satisfies multiple CRC rows")
	}
}

func TestGetAgentCapabilities_CredentialsReady_MultiCRCPartiallyUnsatisfied(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Connector with two required credential rows for different services.
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "svc_alpha", "api_key")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "svc_beta", "api_key")

	// Credential that matches only svc_alpha (not using connector ID fallback).
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, "svc_alpha")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	testhelper.InsertAgentConnectorCredential(t, tx, testhelper.GenerateID(t, "acc_"), agentID, uid, connID, credID)

	caps, err := db.GetAgentCapabilities(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("GetAgentCapabilities: %v", err)
	}
	if len(caps.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(caps.Connectors))
	}
	if caps.Connectors[0].CredentialsReady {
		t.Error("expected credentials_ready=false when bound credential only matches one of two CRC rows")
	}
}

func TestConnectorRequiredCredentials_MixedAuthTypeRejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	// Create a connector with a non-oauth2 CRC row.
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "svc_a", "api_key")

	// Attempting to insert an oauth2 CRC row for the same connector should fail.
	_, err := tx.Exec(t.Context(),
		`INSERT INTO connector_required_credentials (connector_id, service, auth_type, oauth_provider, oauth_scopes)
		 VALUES ($1, 'svc_b', 'oauth2', 'google', '{}')`,
		connID)
	if err == nil {
		t.Fatal("expected error when inserting mixed auth types for the same connector, got nil")
	}
}

func TestConnectorRequiredCredentials_MixedAuthTypeRejectedOnUpdate(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	// Create a connector with two non-oauth2 CRC rows (different services).
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "svc_a", "api_key")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "svc_b", "basic")

	// Updating one row's auth_type to oauth2 should be rejected because the
	// other row is still non-oauth2, creating a mixed-auth situation.
	_, err := tx.Exec(t.Context(),
		`UPDATE connector_required_credentials
		 SET auth_type = 'oauth2', oauth_provider = 'google', oauth_scopes = '{}'
		 WHERE connector_id = $1 AND service = 'svc_a' AND auth_type = 'api_key'`,
		connID)
	if err == nil {
		t.Fatal("expected error when updating auth_type to create mixed auth types, got nil")
	}
}
