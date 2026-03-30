package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/vault"
)

// ── Connector execution during standing approval execute ─────────────────────

// mockAction records calls and returns a configurable result or error.
type mockAction struct {
	called  bool
	request connectors.ActionRequest
	result  *connectors.ActionResult
	err     error
}

func (a *mockAction) Execute(_ context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	a.called = true
	a.request = req
	if a.err != nil {
		return nil, a.err
	}
	return a.result, nil
}

type mockConnector struct {
	id              string
	actions         map[string]connectors.Action
	validateCredsErr error
}

func (c *mockConnector) ID() string                         { return c.id }
func (c *mockConnector) Actions() map[string]connectors.Action { return c.actions }
func (c *mockConnector) ValidateCredentials(_ context.Context, _ connectors.Credentials) error {
	return c.validateCredsErr
}


func TestExecuteStandingApproval_ConnectorExecution(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "testconn.do_thing")

	// Set up connector metadata in DB.
	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorAction(t, tx, "testconn", "testconn.do_thing", "Do Thing")
	testhelper.InsertConnectorRequiredCredential(t, tx, "testconn", "testconn_service", "api_key")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, "testconn")

	// Store a credential the mock vault can decrypt.
	mockVault := vault.NewMockVaultStore()
	credID := "cred_test1"
	secretID, err := mockVault.CreateSecret(context.Background(), tx, "test-cred", []byte(`{"api_key":"secret_value_123"}`))
	if err != nil {
		t.Fatalf("failed to create mock vault secret: %v", err)
	}
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, "testconn_service", secretID)

	// Create explicit credential binding.
	bindingID := testhelper.GenerateID(t, "accr_")
	_, bindErr := db.UpsertAgentConnectorCredential(t.Context(), tx, db.UpsertAgentConnectorCredentialParams{
		ID: bindingID, AgentID: agentID, ConnectorID: "testconn",
		ApproverID: uid, CredentialID: &credID,
	})
	if bindErr != nil {
		t.Fatalf("upsert binding: %v", bindErr)
	}

	// Set up mock connector action.
	resultData := json.RawMessage(`{"issue_id":42}`)
	action := &mockAction{
		result: &connectors.ActionResult{Data: resultData},
	}
	registry := connectors.NewRegistry()
	registry.Register(&mockConnector{
		id:      "testconn",
		actions: map[string]connectors.Action{"testconn.do_thing": action},
	})

	deps := &Deps{DB: tx, Vault: mockVault, Connectors: registry, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{"parameters":{"repo":"test"}}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp executeStandingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.StandingApprovalID != saID {
		t.Errorf("expected standing_approval_id %q, got %q", saID, resp.StandingApprovalID)
	}
	if resp.ExecutionID == 0 {
		t.Error("expected non-zero execution_id")
	}
	if resp.ActionResult == nil {
		t.Fatal("expected action_result to be present")
	}
	if string(*resp.ActionResult) != `{"issue_id":42}` {
		t.Errorf("expected action_result %q, got %q", `{"issue_id":42}`, string(*resp.ActionResult))
	}

	// Verify the action received the correct credentials.
	if !action.called {
		t.Fatal("expected action.Execute to be called")
	}
	apiKey, ok := action.request.Credentials.Get("api_key")
	if !ok {
		t.Fatal("expected credentials to contain 'api_key'")
	}
	if apiKey != "secret_value_123" {
		t.Errorf("expected api_key 'secret_value_123', got %q", apiKey)
	}

	// Verify parameters were passed through.
	if string(action.request.Parameters) != `{"repo":"test"}` {
		t.Errorf("expected parameters %q, got %q", `{"repo":"test"}`, string(action.request.Parameters))
	}
	if action.request.ActionType != "testconn.do_thing" {
		t.Errorf("expected action type 'testconn.do_thing', got %q", action.request.ActionType)
	}
}

func TestExecuteStandingApproval_NoConnectorRegistered(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid) // action_type = "test.action"

	// Empty registry — no connector for "test.action".
	registry := connectors.NewRegistry()
	deps := &Deps{DB: tx, Connectors: registry, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (graceful degradation), got %d: %s", w.Code, w.Body.String())
	}

	var resp executeStandingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.ActionResult != nil {
		t.Errorf("expected no action_result when no connector registered, got %s", string(*resp.ActionResult))
	}
}

func TestExecuteStandingApproval_NilRegistry(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	// No Connectors field set at all.
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (nil registry), got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_CredentialValidationFails(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "testconn.cred_val")

	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorAction(t, tx, "testconn", "testconn.cred_val", "Cred Val")

	action := &mockAction{
		result: &connectors.ActionResult{Data: json.RawMessage(`{}`)},
	}
	registry := connectors.NewRegistry()
	registry.Register(&mockConnector{
		id:               "testconn",
		actions:          map[string]connectors.Action{"testconn.cred_val": action},
		validateCredsErr: &connectors.ValidationError{Message: "bot_token must start with \"xoxb-\""},
	})

	deps := &Deps{DB: tx, Connectors: registry, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// ValidateCredentials failure should return 400.
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for credential validation failure, got %d: %s", w.Code, w.Body.String())
	}

	// Action should not have been called.
	if action.called {
		t.Error("expected action.Execute not to be called when credential validation fails")
	}
}

func TestExecuteStandingApproval_ConnectorValidationError(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "testconn.validate_err")

	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorAction(t, tx, "testconn", "testconn.validate_err", "Validate Err")

	// No required credentials for this connector — action will be called directly.
	action := &mockAction{
		err: &connectors.ValidationError{Message: "missing required field: title"},
	}
	registry := connectors.NewRegistry()
	registry.Register(&mockConnector{
		id:      "testconn",
		actions: map[string]connectors.Action{"testconn.validate_err": action},
	})

	deps := &Deps{DB: tx, Connectors: registry, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for ValidationError, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_ConnectorExternalError(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "testconn.ext_err")

	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorAction(t, tx, "testconn", "testconn.ext_err", "External Err")

	action := &mockAction{
		err: &connectors.ExternalError{StatusCode: 500, Message: "GitHub API unavailable"},
	}
	registry := connectors.NewRegistry()
	registry.Register(&mockConnector{
		id:      "testconn",
		actions: map[string]connectors.Action{"testconn.ext_err": action},
	})

	deps := &Deps{DB: tx, Connectors: registry, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for ExternalError, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_ConnectorAuthError(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "testconn.auth_err")

	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorAction(t, tx, "testconn", "testconn.auth_err", "Auth Err")

	action := &mockAction{
		err: &connectors.AuthError{Message: "token expired"},
	}
	registry := connectors.NewRegistry()
	registry.Register(&mockConnector{
		id:      "testconn",
		actions: map[string]connectors.Action{"testconn.auth_err": action},
	})

	deps := &Deps{DB: tx, Connectors: registry, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for AuthError, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_ConnectorRateLimitError(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "testconn.rate_err")

	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorAction(t, tx, "testconn", "testconn.rate_err", "Rate Err")

	action := &mockAction{
		err: &connectors.RateLimitError{Message: "rate limited", RetryAfter: 30 * time.Second},
	}
	registry := connectors.NewRegistry()
	registry.Register(&mockConnector{
		id:      "testconn",
		actions: map[string]connectors.Action{"testconn.rate_err": action},
	})

	deps := &Deps{DB: tx, Connectors: registry, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for RateLimitError, got %d: %s", w.Code, w.Body.String())
	}

	// Verify retry_after is in the response.
	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.RetryAfter != 30 {
		t.Errorf("expected retry_after 30, got %d", errResp.Error.RetryAfter)
	}
}

func TestExecuteStandingApproval_ConnectorTimeoutError(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "testconn.timeout_err")

	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorAction(t, tx, "testconn", "testconn.timeout_err", "Timeout Err")

	action := &mockAction{
		err: &connectors.TimeoutError{Message: "request timed out"},
	}
	registry := connectors.NewRegistry()
	registry.Register(&mockConnector{
		id:      "testconn",
		actions: map[string]connectors.Action{"testconn.timeout_err": action},
	})

	deps := &Deps{DB: tx, Connectors: registry, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504 for TimeoutError, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_MissingCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "testconn.need_creds")

	// Connector requires credentials, but no credential binding was created for the agent.
	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorAction(t, tx, "testconn", "testconn.need_creds", "Need Creds")
	testhelper.InsertConnectorRequiredCredential(t, tx, "testconn", "testconn_service", "api_key")

	action := &mockAction{
		result: &connectors.ActionResult{Data: json.RawMessage(`{}`)},
	}
	registry := connectors.NewRegistry()
	registry.Register(&mockConnector{
		id:      "testconn",
		actions: map[string]connectors.Action{"testconn.need_creds": action},
	})

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, Connectors: registry, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// Missing credentials maps to ValidationError → 400.
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing credentials, got %d: %s", w.Code, w.Body.String())
	}

	// Action should not have been called.
	if action.called {
		t.Error("expected action.Execute not to be called when credentials are missing")
	}
}

func TestExecuteStandingApproval_NoRequiredCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "testconn.no_creds")

	// Connector has no required credentials.
	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorAction(t, tx, "testconn", "testconn.no_creds", "No Creds")

	resultData := json.RawMessage(`{"status":"ok"}`)
	action := &mockAction{
		result: &connectors.ActionResult{Data: resultData},
	}
	registry := connectors.NewRegistry()
	registry.Register(&mockConnector{
		id:      "testconn",
		actions: map[string]connectors.Action{"testconn.no_creds": action},
	})

	deps := &Deps{DB: tx, Connectors: registry, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp executeStandingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.ActionResult == nil {
		t.Fatal("expected action_result to be present")
	}
	if string(*resp.ActionResult) != `{"status":"ok"}` {
		t.Errorf("expected action_result %q, got %q", `{"status":"ok"}`, string(*resp.ActionResult))
	}

	// Action should have received empty credentials.
	if !action.called {
		t.Fatal("expected action.Execute to be called")
	}
	if len(action.request.Credentials.Keys()) != 0 {
		t.Errorf("expected empty credentials, got keys: %v", action.request.Credentials.Keys())
	}
}
