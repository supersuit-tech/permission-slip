package api

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/vault"
)

const standingApprovalRequestTestConstraints = `{"scope":"test-scope","ping":"*"}`

type standingApprovalRequestMISetup struct {
	tx          db.DBTX
	deps        *Deps
	router      http.Handler
	agentID     int64
	privKey     ed25519.PrivateKey
	uid         string
	connID      string
	actionType  string
	instDefault *db.AgentConnectorInstance
	instOther   *db.AgentConnectorInstance
	configID    string
}

func setupStandingApprovalRequestMultiInstance(t *testing.T) standingApprovalRequestMISetup {
	t.Helper()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	connID := testhelper.GenerateID(t, "conn_")
	actionType := connID + ".ping"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Ping")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, connID, "api_key")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	instOther, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("CreateAgentConnectorInstance: %v", err)
	}

	v := vault.NewMockVaultStore()
	credJSON1, _ := json.Marshal(map[string]string{"api_key": "token-default"})
	v1, err := v.CreateSecret(ctx, tx, "c1", credJSON1)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID1 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID1, uid, connID, "Default", v1)

	credJSON2, _ := json.Marshal(map[string]string{"api_key": "token-sales"})
	v2, err := v.CreateSecret(ctx, tx, "c2", credJSON2)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID2 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID2, uid, connID, "Sales", v2)

	instDefault, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || instDefault == nil {
		t.Fatalf("default instance: %v", instDefault)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: instDefault.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID1,
	}); err != nil {
		t.Fatalf("bind default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: instOther.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID2,
	}); err != nil {
		t.Fatalf("bind sales: %v", err)
	}

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, actionType))
	deps := &Deps{DB: tx, Vault: v, Connectors: reg, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	return standingApprovalRequestMISetup{
		tx: tx, deps: deps, router: router, agentID: agentID, privKey: privKey,
		uid: uid, connID: connID, actionType: actionType,
		instDefault: instDefault, instOther: instOther, configID: configID,
	}
}

func postStandingApprovalRequest(t *testing.T, s standingApprovalRequestMISetup, body string) (int, agentStandingApprovalRequestResponse) {
	t.Helper()
	r := signedJSONRequest(t, http.MethodPost, "/standing-approvals/request", body, s.privKey, s.agentID)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	var resp agentStandingApprovalRequestResponse
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
	}
	return w.Code, resp
}

func getStandingApprovalRequest(t *testing.T, s standingApprovalRequestMISetup, requestID string) standingApprovalRequestResponse {
	t.Helper()
	r := authenticatedRequest(t, http.MethodGet, "/standing-approval-requests/"+requestID, s.uid)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("get request: %d %s", w.Code, w.Body.String())
	}
	var resp standingApprovalRequestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp
}

func TestAgentCreateStandingApprovalRequest_ExplicitInstanceUUID(t *testing.T) {
	t.Parallel()
	s := setupStandingApprovalRequestMultiInstance(t)

	body := `{"action_type":"` + s.actionType + `","constraints":` + standingApprovalRequestTestConstraints + `,"source_action_configuration_id":"` + s.configID + `","connector_instance":"` + s.instOther.ConnectorInstanceID + `"}`
	code, createResp := postStandingApprovalRequest(t, s, body)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}

	got := getStandingApprovalRequest(t, s, createResp.RequestID)
	if got.ConnectorInstanceID == nil || *got.ConnectorInstanceID != s.instOther.ConnectorInstanceID {
		t.Fatalf("connector_instance_id: want %q got %v", s.instOther.ConnectorInstanceID, got.ConnectorInstanceID)
	}
	if got.ConnectorInstanceDisplay == nil || *got.ConnectorInstanceDisplay != "Sales" {
		t.Fatalf("connector_instance_display: want Sales got %v", got.ConnectorInstanceDisplay)
	}
}

func TestAgentCreateStandingApprovalRequest_DisplayNameSelector(t *testing.T) {
	t.Parallel()
	s := setupStandingApprovalRequestMultiInstance(t)

	body := `{"action_type":"` + s.actionType + `","constraints":` + standingApprovalRequestTestConstraints + `,"connector_instance":"Sales"}`
	code, createResp := postStandingApprovalRequest(t, s, body)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}

	got := getStandingApprovalRequest(t, s, createResp.RequestID)
	if got.ConnectorInstanceID == nil || *got.ConnectorInstanceID != s.instOther.ConnectorInstanceID {
		t.Fatalf("connector_instance_id: want %q got %v", s.instOther.ConnectorInstanceID, got.ConnectorInstanceID)
	}
}

func TestAgentCreateStandingApprovalRequest_AmbiguousDisplayName(t *testing.T) {
	t.Parallel()
	s := setupStandingApprovalRequestMultiInstance(t)
	ctx := context.Background()

	sharedLabel := "SharedLabel"
	credJSON, _ := json.Marshal(map[string]string{"api_key": "x"})
	vSecret, err := s.deps.Vault.CreateSecret(ctx, s.tx, "shared", credJSON)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, s.tx, credID, s.uid, s.connID, sharedLabel, vSecret)
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, s.tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: s.agentID, ConnectorID: s.connID,
		ConnectorInstanceID: s.instDefault.ConnectorInstanceID, ApproverID: s.uid, CredentialID: &credID,
	}); err != nil {
		t.Fatalf("rebind default with shared label: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, s.tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: s.agentID, ConnectorID: s.connID,
		ConnectorInstanceID: s.instOther.ConnectorInstanceID, ApproverID: s.uid, CredentialID: &credID,
	}); err != nil {
		t.Fatalf("rebind other with shared label: %v", err)
	}

	body := `{"action_type":"` + s.actionType + `","constraints":` + standingApprovalRequestTestConstraints + `,"connector_instance":"` + sharedLabel + `"}`
	r := signedJSONRequest(t, http.MethodPost, "/standing-approvals/request", body, s.privKey, s.agentID)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), string(ErrConnectorInstanceAmbiguous)) {
		t.Errorf("expected ambiguous error, got: %s", w.Body.String())
	}
}

func TestAgentCreateStandingApprovalRequest_ConflictsWithPinnedConfig(t *testing.T) {
	t.Parallel()
	s := setupStandingApprovalRequestMultiInstance(t)

	pinnedConfigID := testhelper.GenerateID(t, "ac_")
	params, _ := json.Marshal(map[string]string{
		"connector_instance": s.instDefault.ConnectorInstanceID,
	})
	testhelper.InsertActionConfigFull(t, s.tx, pinnedConfigID, s.agentID, s.uid, s.connID, s.actionType, testhelper.ActionConfigOpts{
		Parameters: params,
		Name:       "Pinned default",
	})

	body := `{"action_type":"` + s.actionType + `","constraints":` + standingApprovalRequestTestConstraints + `,"source_action_configuration_id":"` + pinnedConfigID + `","connector_instance":"` + s.instOther.ConnectorInstanceID + `"}`
	r := signedJSONRequest(t, http.MethodPost, "/standing-approvals/request", body, s.privKey, s.agentID)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "conflicts with the pinned instance") {
		t.Errorf("expected conflict error, got: %s", w.Body.String())
	}
}

func TestAgentCreateStandingApprovalRequest_OmittedSelectorWithMultipleInstances(t *testing.T) {
	t.Parallel()
	s := setupStandingApprovalRequestMultiInstance(t)

	body := `{"action_type":"` + s.actionType + `","constraints":` + standingApprovalRequestTestConstraints + `,"source_action_configuration_id":"` + s.configID + `"}`
	code, createResp := postStandingApprovalRequest(t, s, body)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}

	got := getStandingApprovalRequest(t, s, createResp.RequestID)
	if got.ConnectorInstanceID != nil {
		t.Errorf("expected nil connector_instance_id when selector omitted, got %v", got.ConnectorInstanceID)
	}
	if got.ConnectorInstanceDisplay != nil && *got.ConnectorInstanceDisplay != "" {
		t.Errorf("expected no instance display when selector omitted, got %v", got.ConnectorInstanceDisplay)
	}
}

func TestApproveStandingApprovalRequest_PropagatesConnectorInstanceID(t *testing.T) {
	t.Parallel()
	s := setupStandingApprovalRequestMultiInstance(t)

	body := `{"action_type":"` + s.actionType + `","constraints":` + standingApprovalRequestTestConstraints + `,"connector_instance":"` + s.instOther.ConnectorInstanceID + `"}`
	code, createResp := postStandingApprovalRequest(t, s, body)
	if code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d", code)
	}

	r := authenticatedRequest(t, http.MethodPost, "/standing-approval-requests/"+createResp.RequestID+"/approve", s.uid)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var approveResp approveStandingApprovalRequestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &approveResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if approveResp.StandingApproval == nil {
		t.Fatal("expected standing_approval in response")
	}
	sa, err := db.GetStandingApprovalByIDAndUser(t.Context(), s.tx, approveResp.ResultingStandingApprovalID, s.uid)
	if err != nil {
		t.Fatalf("GetStandingApprovalByIDAndUser: %v", err)
	}
	if sa == nil {
		t.Fatal("standing approval not found")
	}
	if sa.ConnectorInstanceID == nil || *sa.ConnectorInstanceID != s.instOther.ConnectorInstanceID {
		t.Fatalf("standing approval connector_instance_id: want %q got %v", s.instOther.ConnectorInstanceID, sa.ConnectorInstanceID)
	}
}

func TestStandingApprovalRequest_InstanceScopedAutoApprove_EndToEnd(t *testing.T) {
	t.Parallel()
	s := setupStandingApprovalRequestMultiInstance(t)

	body := `{"action_type":"` + s.actionType + `","constraints":` + standingApprovalRequestTestConstraints + `,"connector_instance":"Sales"}`
	code, createResp := postStandingApprovalRequest(t, s, body)
	if code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d", code)
	}

	r := authenticatedRequest(t, http.MethodPost, "/standing-approval-requests/"+createResp.RequestID+"/approve", s.uid)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var approveResp approveStandingApprovalRequestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &approveResp); err != nil {
		t.Fatalf("unmarshal approve: %v", err)
	}
	saID := approveResp.ResultingStandingApprovalID

	matchBody := `{"request_id":"req_sa_match","action":{"type":"` + s.actionType + `","parameters":{"connector_instance":"Sales","scope":"test-scope"}},"context":{"description":"match"}}`
	rMatch := signedJSONRequest(t, http.MethodPost, "/approvals/request", matchBody, s.privKey, s.agentID)
	wMatch := httptest.NewRecorder()
	s.router.ServeHTTP(wMatch, rMatch)
	if wMatch.Code != http.StatusOK {
		t.Fatalf("matching auto-approve: expected 200, got %d: %s", wMatch.Code, wMatch.Body.String())
	}
	var matchResp agentRequestApprovalResponse
	if err := json.Unmarshal(wMatch.Body.Bytes(), &matchResp); err != nil {
		t.Fatalf("unmarshal match: %v", err)
	}
	if matchResp.Status != "approved" {
		t.Errorf("matching instance: expected approved, got %q", matchResp.Status)
	}
	if matchResp.StandingApprovalID != saID {
		t.Errorf("matching instance: expected standing_approval_id %q, got %q", saID, matchResp.StandingApprovalID)
	}
	testhelper.RequireStandingApprovalExecutionCount(t, s.tx, saID, 1)

	otherBody := `{"request_id":"req_sa_other","action":{"type":"` + s.actionType + `","parameters":{"connector_instance":"Default","scope":"test-scope"}},"context":{"description":"other"}}`
	rOther := signedJSONRequest(t, http.MethodPost, "/approvals/request", otherBody, s.privKey, s.agentID)
	wOther := httptest.NewRecorder()
	s.router.ServeHTTP(wOther, rOther)
	if wOther.Code != http.StatusOK {
		t.Fatalf("other instance: expected 200, got %d: %s", wOther.Code, wOther.Body.String())
	}
	var otherResp agentRequestApprovalResponse
	if err := json.Unmarshal(wOther.Body.Bytes(), &otherResp); err != nil {
		t.Fatalf("unmarshal other: %v", err)
	}
	if otherResp.Status != "pending" {
		t.Errorf("other instance: expected pending, got %q", otherResp.Status)
	}
	if otherResp.ApprovalID == "" {
		t.Error("other instance: expected pending approval_id")
	}
	testhelper.RequireStandingApprovalExecutionCount(t, s.tx, saID, 1)
}
