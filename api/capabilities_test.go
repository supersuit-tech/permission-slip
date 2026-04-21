package api

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// capabilitiesRequest creates a signed GET request for the capabilities endpoint.
func capabilitiesRequest(t *testing.T, agentID int64, privKey ed25519.PrivateKey) *http.Request {
	t.Helper()
	path := fmt.Sprintf("/agents/%d/capabilities", agentID)
	r := httptest.NewRequest(http.MethodGet, path, nil)
	SignRequest(privKey, agentID, r, nil)
	return r
}

func TestGetCapabilities_HappyPath(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	// Set up a connector with credentials ready.
	conn1 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, conn1, "Gmail", "Send and manage emails")
	svc1 := testhelper.GenerateID(t, "svc_")
	testhelper.InsertConnectorRequiredCredential(t, tx, conn1, svc1, "api_key")
	cred1 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, cred1, uid, svc1)

	riskLow := "low"
	sendDesc := "Send an email"
	schema := json.RawMessage(`{"type":"object","required":["to","subject"]}`)
	testhelper.InsertConnectorActionFull(t, tx, conn1, "email.send", "Send Email", testhelper.ConnectorActionOpts{
		Description:      &sendDesc,
		RiskLevel:        &riskLow,
		ParametersSchema: schema,
	})
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn1)
	testhelper.InsertAgentConnectorCredential(t, tx, testhelper.GenerateID(t, "acc_"), agentID, uid, conn1, cred1)

	// Add a standing approval.
	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "email.send",
		Constraints: []byte(`{"recipient_pattern":"*@mycompany.com"}`),
	})

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BaseURL: "https://app.permissionslip.dev"})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, resp.AgentID)
	}
	if resp.Approver == nil {
		t.Fatal("expected approver to be set in capabilities response")
	}
	expectedUsername := "u_" + uid[:8]
	if resp.Approver.Username != expectedUsername {
		t.Errorf("expected approver.username %q, got %q", expectedUsername, resp.Approver.Username)
	}
	if len(resp.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(resp.Connectors))
	}

	c := resp.Connectors[0]
	if c.ID != conn1 {
		t.Errorf("expected connector ID %q, got %q", conn1, c.ID)
	}
	if !c.CredentialsReady {
		t.Error("expected credentials_ready=true")
	}
	if c.CredentialsSetupURL != nil {
		t.Error("expected no credentials_setup_url when credentials are ready")
	}
	if len(c.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(c.Actions))
	}

	a := c.Actions[0]
	if a.ActionType != "email.send" {
		t.Errorf("expected action_type 'email.send', got %q", a.ActionType)
	}
	if len(a.StandingApprovals) != 1 {
		t.Fatalf("expected 1 standing approval, got %d", len(a.StandingApprovals))
	}
	if a.StandingApprovals[0].StandingApprovalID != saID {
		t.Errorf("expected standing_approval_id %q, got %q", saID, a.StandingApprovals[0].StandingApprovalID)
	}
}

func TestGetCapabilities_CredentialsNotReady(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	// Connector with missing credentials.
	conn := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, conn, "Stripe", "Payment processing")
	svc := testhelper.GenerateID(t, "svc_")
	testhelper.InsertConnectorRequiredCredential(t, tx, conn, svc, "api_key")
	// No credential inserted — credentials_ready should be false.
	testhelper.InsertConnectorAction(t, tx, conn, "payment.charge", "Charge")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BaseURL: "https://app.permissionslip.dev"})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(resp.Connectors))
	}

	c := resp.Connectors[0]
	if c.CredentialsReady {
		t.Error("expected credentials_ready=false")
	}
	if c.CredentialsSetupURL == nil {
		t.Fatal("expected credentials_setup_url when credentials not ready")
	}
	expected := fmt.Sprintf("https://app.permissionslip.dev/connect/%s", conn)
	if *c.CredentialsSetupURL != expected {
		t.Errorf("expected credentials_setup_url %q, got %q", expected, *c.CredentialsSetupURL)
	}
}

func TestGetCapabilities_NoConnectors(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, resp.AgentID)
	}
	if len(resp.Connectors) != 0 {
		t.Errorf("expected 0 connectors, got %d", len(resp.Connectors))
	}
}

func TestGetCapabilities_MissingSignature(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, _ := insertRegisteredAgentWithKey(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	// No signature header.
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/agents/%d/capabilities", agentID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetCapabilities_InvalidSignature(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, _ := insertRegisteredAgentWithKey(t, tx, uid)

	// Sign with a different key.
	_, wrongPrivKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	r := capabilitiesRequest(t, agentID, wrongPrivKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetCapabilities_AgentIDMismatch(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	// Sign with a different agent_id than the path.
	path := fmt.Sprintf("/agents/%d/capabilities", agentID)
	r := httptest.NewRequest(http.MethodGet, path, nil)
	SignRequest(privKey, agentID+999, r, nil) // wrong agent_id in header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// The signature verification uses the real agent's key but the
	// mismatch check catches that the header agent_id differs from the path.
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrAgentIDMismatch {
		t.Errorf("expected error code %q, got %q", ErrAgentIDMismatch, errResp.Error.Code)
	}
}

func TestGetCapabilities_AgentNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	_, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	// Use a non-existent agent_id.
	r := capabilitiesRequest(t, 999999, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetCapabilities_PendingAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Insert a pending agent with a real key.
	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	var agentID int64
	err = tx.QueryRow(context.Background(),
		`INSERT INTO agents (public_key, approver_id, status) VALUES ($1, $2, 'pending') RETURNING agent_id`,
		pubKeySSH, uid).Scan(&agentID)
	if err != nil {
		t.Fatalf("insert pending agent: %v", err)
	}

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for pending agent, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetCapabilities_ExpiredTimestamp(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	path := fmt.Sprintf("/agents/%d/capabilities", agentID)
	r := httptest.NewRequest(http.MethodGet, path, nil)
	// Sign with a timestamp 10 minutes in the past (beyond the 5-minute window).
	SignRequestAt(privKey, agentID, r, nil, time.Now().Add(-10*time.Minute).Unix())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrTimestampExpired {
		t.Errorf("expected error code %q, got %q", ErrTimestampExpired, errResp.Error.Code)
	}
}

func TestGetCapabilities_MultipleConnectorsAndActions(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	// Connector 1: Gmail with credentials.
	conn1 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, conn1, "Gmail", "Email service")
	svc1 := testhelper.GenerateID(t, "svc_")
	testhelper.InsertConnectorRequiredCredential(t, tx, conn1, svc1, "api_key")
	cred1 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, cred1, uid, svc1)
	testhelper.InsertConnectorAction(t, tx, conn1, "email.send", "Send Email")
	testhelper.InsertConnectorAction(t, tx, conn1, "email.read", "Read Email")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn1)
	testhelper.InsertAgentConnectorCredential(t, tx, testhelper.GenerateID(t, "acc_"), agentID, uid, conn1, cred1)

	// Connector 2: Stripe without credentials.
	conn2 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, conn2, "Stripe", "Payments")
	svc2 := testhelper.GenerateID(t, "svc_")
	testhelper.InsertConnectorRequiredCredential(t, tx, conn2, svc2, "api_key")
	testhelper.InsertConnectorAction(t, tx, conn2, "payment.charge", "Charge")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn2)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BaseURL: "https://app.permissionslip.dev"})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Connectors) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(resp.Connectors))
	}

	// Build a map for easier assertions.
	connByID := make(map[string]connectorCapability)
	for _, c := range resp.Connectors {
		connByID[c.ID] = c
	}

	// Verify connector 1.
	c1, ok := connByID[conn1]
	if !ok {
		t.Fatalf("connector %q not found in response", conn1)
	}
	if !c1.CredentialsReady {
		t.Error("conn1: expected credentials_ready=true")
	}
	if len(c1.Actions) != 2 {
		t.Errorf("conn1: expected 2 actions, got %d", len(c1.Actions))
	}

	// Verify connector 2.
	c2, ok := connByID[conn2]
	if !ok {
		t.Fatalf("connector %q not found in response", conn2)
	}
	if c2.CredentialsReady {
		t.Error("conn2: expected credentials_ready=false")
	}
	if c2.CredentialsSetupURL == nil {
		t.Error("conn2: expected credentials_setup_url")
	}
	if len(c2.Actions) != 1 {
		t.Errorf("conn2: expected 1 action, got %d", len(c2.Actions))
	}
}

func TestGetCapabilities_WithActionConfigurations(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	// Connector with credentials.
	conn := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, conn, "GitHub", "Code hosting")
	svc := testhelper.GenerateID(t, "svc_")
	testhelper.InsertConnectorRequiredCredential(t, tx, conn, svc, "api_key")
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, svc)
	testhelper.InsertConnectorAction(t, tx, conn, "github.create_issue", "Create Issue")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn)
	testhelper.InsertAgentConnectorCredential(t, tx, testhelper.GenerateID(t, "acc_"), agentID, uid, conn, credID)

	// Insert an action configuration with credential bound.
	cfgID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfigFull(t, tx, cfgID, agentID, uid, conn, "github.create_issue", testhelper.ActionConfigOpts{
		Parameters: []byte(`{"repo":"supersuit-tech/webapp","title":"*","body":"*"}`),
		Name:       "Create issues in webapp",
	})

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BaseURL: "https://app.permissionslip.dev"})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(resp.Connectors))
	}
	c := resp.Connectors[0]
	if len(c.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(c.Actions))
	}
	a := c.Actions[0]

	// Verify action configurations are included.
	if len(a.ActionConfigurations) != 1 {
		t.Fatalf("expected 1 action config, got %d", len(a.ActionConfigurations))
	}
	ac := a.ActionConfigurations[0]
	if ac.ConfigurationID != cfgID {
		t.Errorf("expected config ID %q, got %q", cfgID, ac.ConfigurationID)
	}
	if ac.ActionType != "github.create_issue" {
		t.Errorf("expected action_type 'github.create_issue', got %q", ac.ActionType)
	}
	if ac.Name != "Create issues in webapp" {
		t.Errorf("expected name 'Create issues in webapp', got %q", ac.Name)
	}
	// Verify parameters are included.
	var params map[string]interface{}
	if err := json.Unmarshal(ac.Parameters, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if params["repo"] != "supersuit-tech/webapp" {
		t.Errorf("expected repo param 'supersuit-tech/webapp', got %v", params["repo"])
	}
	if params["title"] != "*" {
		t.Errorf("expected title param '*', got %v", params["title"])
	}
}

func TestGetCapabilities_ActionConfigCredentialNotReady(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	conn := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, conn)
	testhelper.InsertConnectorAction(t, tx, conn, "slack.post_message", "Post Message")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn)

	// Action config WITHOUT credential (no credential_id).
	cfgID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, cfgID, agentID, uid, conn, "slack.post_message")

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(resp.Connectors))
	}
	a := resp.Connectors[0].Actions[0]
	if len(a.ActionConfigurations) != 1 {
		t.Fatalf("expected 1 action config, got %d", len(a.ActionConfigurations))
	}
}

func TestGetCapabilities_ActionConfigsScopedByConnector(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	// Two connectors that share the same action type.
	conn1 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, conn1, "Provider A", "First provider")
	testhelper.InsertConnectorAction(t, tx, conn1, "shared.action", "Shared Action")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn1)

	conn2 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnectorWithDescription(t, tx, conn2, "Provider B", "Second provider")
	testhelper.InsertConnectorAction(t, tx, conn2, "shared.action", "Shared Action")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn2)

	// Config for connector 1 only.
	cfg1 := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfigFull(t, tx, cfg1, agentID, uid, conn1, "shared.action", testhelper.ActionConfigOpts{
		Name:       "Config for provider A",
		Parameters: []byte(`{"source":"a"}`),
	})

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Connectors) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(resp.Connectors))
	}

	connByID := make(map[string]connectorCapability)
	for _, c := range resp.Connectors {
		connByID[c.ID] = c
	}

	// Connector 1 should have the config.
	c1 := connByID[conn1]
	if len(c1.Actions) != 1 {
		t.Fatalf("conn1: expected 1 action, got %d", len(c1.Actions))
	}
	if len(c1.Actions[0].ActionConfigurations) != 1 {
		t.Errorf("conn1: expected 1 action config, got %d", len(c1.Actions[0].ActionConfigurations))
	}

	// Connector 2 should NOT have the config (it belongs to connector 1).
	c2 := connByID[conn2]
	if len(c2.Actions) != 1 {
		t.Fatalf("conn2: expected 1 action, got %d", len(c2.Actions))
	}
	if len(c2.Actions[0].ActionConfigurations) != 0 {
		t.Errorf("conn2: expected 0 action configs (config belongs to conn1), got %d", len(c2.Actions[0].ActionConfigurations))
	}
}

func TestGetCapabilities_DisabledConfigExcluded(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	conn := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, conn)
	testhelper.InsertConnectorAction(t, tx, conn, "slack.post_message", "Post Message")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn)

	// Insert a disabled config — it should NOT appear in capabilities.
	cfgID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfigFull(t, tx, cfgID, agentID, uid, conn, "slack.post_message", testhelper.ActionConfigOpts{
		Status: "disabled",
		Name:   "Disabled config",
	})

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(resp.Connectors))
	}
	a := resp.Connectors[0].Actions[0]
	if len(a.ActionConfigurations) != 0 {
		t.Errorf("expected 0 action configs (disabled should be excluded), got %d", len(a.ActionConfigurations))
	}
}

func TestGetCapabilities_InvalidAgentID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	r := httptest.NewRequest(http.MethodGet, "/agents/notanumber/capabilities", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetCapabilities_MultiInstance_InjectsConnectorInstanceEnum(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	conn := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, conn)
	schema := json.RawMessage(`{"type":"object","properties":{"channel":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, conn, conn+".post", "Post", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn)
	ctx := context.Background()
	inst2, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: conn,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}
	defInst, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, conn)
	if err != nil || defInst == nil {
		t.Fatalf("default: %v", defInst)
	}

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(resp.Connectors))
	}
	a := resp.Connectors[0].Actions[0]
	var sch map[string]any
	if err := json.Unmarshal(a.ParametersSchema, &sch); err != nil {
		t.Fatalf("schema: %v", err)
	}
	props, _ := sch["properties"].(map[string]any)
	if props == nil {
		t.Fatal("expected properties in parameters_schema")
	}
	ci, ok := props["connector_instance"].(map[string]any)
	if !ok {
		t.Fatalf("expected connector_instance in properties, got %#v", props)
	}
	enum, _ := ci["enum"].([]any)
	if len(enum) != 2 {
		t.Fatalf("expected enum len 2, got %#v", enum)
	}
	gotEnum := make(map[string]struct{})
	for _, x := range enum {
		s, _ := x.(string)
		gotEnum[s] = struct{}{}
	}
	if _, ok := gotEnum[defInst.ConnectorInstanceID]; !ok {
		t.Errorf("enum missing default id %q: %#v", defInst.ConnectorInstanceID, enum)
	}
	if _, ok := gotEnum[inst2.ConnectorInstanceID]; !ok {
		t.Errorf("enum missing second id %q: %#v", inst2.ConnectorInstanceID, enum)
	}
	desc, _ := ci["description"].(string)
	if !strings.Contains(desc, defInst.ConnectorInstanceID) || !strings.Contains(desc, inst2.ConnectorInstanceID) {
		t.Errorf("description should list both UUIDs: %q", desc)
	}
}

func TestGetCapabilities_SingleInstance_OmitsConnectorInstanceInjection(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	conn := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, conn)
	schema := json.RawMessage(`{"type":"object","properties":{"channel":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, conn, conn+".post", "Post", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})
	testhelper.InsertAgentConnector(t, tx, agentID, uid, conn)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	r := capabilitiesRequest(t, agentID, privKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	a := resp.Connectors[0].Actions[0]
	var sch map[string]any
	if err := json.Unmarshal(a.ParametersSchema, &sch); err != nil {
		t.Fatalf("schema: %v", err)
	}
	props, _ := sch["properties"].(map[string]any)
	if props == nil {
		t.Fatal("expected properties")
	}
	if _, ok := props["connector_instance"]; ok {
		t.Error("single-instance connector should not inject connector_instance")
	}
}
