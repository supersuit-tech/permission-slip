package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestAgentCreateStandingApprovalRequest_Success(t *testing.T) {
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)
	configID := standingApprovalTestConfigID(t, tx, agentID, uid, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret, BaseURL: "https://app.example.com"}
	router := NewRouter(deps)

	reqBody := `{"action_type":"email.send","constraints":{"to":"*@example.com"},"source_action_configuration_id":"` + configID + `"}`
	r := signedJSONRequest(t, http.MethodPost, "/standing-approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp agentStandingApprovalRequestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.RequestID == "" {
		t.Error("expected request_id")
	}
	if resp.Status != "pending" {
		t.Errorf("expected pending, got %q", resp.Status)
	}
}

func TestApproveStandingApprovalRequest_HappyPath(t *testing.T) {
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	configID := standingApprovalTestConfigID(t, tx, agentID, uid, "email.send")

	requestID := testhelper.GenerateID(t, "sar_")
	constraints := []byte(`{"to":"*@example.com"}`)
	_, err := tx.Exec(t.Context(),
		`INSERT INTO standing_approval_requests
		   (request_id, agent_id, user_id, action_type, action_version, constraints, source_action_configuration_id, status)
		 VALUES ($1, $2, $3, 'email.send', '1', $4, $5, 'pending')`,
		requestID, agentID, uid, constraints, configID,
	)
	if err != nil {
		t.Fatalf("insert request: %v", err)
	}

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approval-requests/"+requestID+"/approve", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp approveStandingApprovalRequestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "approved" {
		t.Errorf("expected approved, got %q", resp.Status)
	}
	if resp.ResultingStandingApprovalID == "" {
		t.Error("expected resulting_standing_approval_id")
	}
}

func TestDenyStandingApprovalRequest(t *testing.T) {
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	configID := standingApprovalTestConfigID(t, tx, agentID, uid, "email.send")

	requestID := testhelper.GenerateID(t, "sar_")
	testhelper.InsertStandingApprovalRequest(t, tx, requestID, agentID, uid, "email.send", []byte(`{"to":"*@example.com"}`))
	_, _ = tx.Exec(t.Context(),
		`UPDATE standing_approval_requests SET source_action_configuration_id = $1 WHERE request_id = $2`,
		configID, requestID,
	)

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approval-requests/"+requestID+"/deny", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApproveStandingApprovalRequest_ForbiddenOtherUser(t *testing.T) {
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u_"+uid2[:8])
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u_"+uid1[:8])

	requestID := testhelper.GenerateID(t, "sar_")
	testhelper.InsertStandingApprovalRequest(t, tx, requestID, agentID, uid1, "email.send", []byte(`{"to":"*@example.com"}`))

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approval-requests/"+requestID+"/approve", uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for other user, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApproveStandingApprovalRequest_Idempotent(t *testing.T) {
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	configID := standingApprovalTestConfigID(t, tx, agentID, uid, "email.send")

	requestID := testhelper.GenerateID(t, "sar_")
	constraints := []byte(`{"to":"*@example.com"}`)
	_, err := tx.Exec(t.Context(),
		`INSERT INTO standing_approval_requests
		   (request_id, agent_id, user_id, action_type, action_version, constraints, source_action_configuration_id, status)
		 VALUES ($1, $2, $3, 'email.send', '1', $4, $5, 'pending')`,
		requestID, agentID, uid, constraints, configID,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)
	path := "/standing-approval-requests/" + requestID + "/approve"

	r1 := authenticatedRequest(t, http.MethodPost, path, uid)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first approve: %d %s", w1.Code, w1.Body.String())
	}

	r2 := authenticatedRequest(t, http.MethodPost, path, uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second approve (idempotent): %d %s", w2.Code, w2.Body.String())
	}
}

func TestApproveStandingApprovalRequest_AutoCreatesActionConfig(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestConnectorOnly(t, tx, "email.send")

	requestID := testhelper.GenerateID(t, "sar_")
	constraints := []byte(`{"to":"*@example.com"}`)
	_, err := tx.Exec(t.Context(),
		`INSERT INTO standing_approval_requests
		   (request_id, agent_id, user_id, action_type, action_version, constraints, status)
		 VALUES ($1, $2, $3, 'email.send', '1', $4, 'pending')`,
		requestID, agentID, uid, constraints,
	)
	if err != nil {
		t.Fatalf("insert request: %v", err)
	}

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approval-requests/"+requestID+"/approve", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp approveStandingApprovalRequestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.StandingApproval == nil || resp.StandingApproval.SourceActionConfigurationID == nil {
		t.Fatal("expected standing approval with source_action_configuration_id")
	}

	configs, err := db.ListActionConfigsByAgent(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("ListActionConfigsByAgent: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 auto-created action config, got %d", len(configs))
	}
	if configs[0].Name != autoCreatedFromRuleProposalConfigName {
		t.Errorf("expected config name %q, got %q", autoCreatedFromRuleProposalConfigName, configs[0].Name)
	}
	if configs[0].ID != *resp.StandingApproval.SourceActionConfigurationID {
		t.Errorf("expected linked config %q, got %q", configs[0].ID, *resp.StandingApproval.SourceActionConfigurationID)
	}
}

func TestApproveStandingApprovalRequest_MultipleMatchingConfigsUsesNewest(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestConnectorOnly(t, tx, "email.send")
	connectorID := "email"

	olderID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, olderID, agentID, uid, connectorID, "email.send")
	time.Sleep(10 * time.Millisecond)
	newerID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, newerID, agentID, uid, connectorID, "email.send")

	requestID := testhelper.GenerateID(t, "sar_")
	constraints := []byte(`{"to":"*@example.com"}`)
	_, err := tx.Exec(t.Context(),
		`INSERT INTO standing_approval_requests
		   (request_id, agent_id, user_id, action_type, action_version, constraints, status)
		 VALUES ($1, $2, $3, 'email.send', '1', $4, 'pending')`,
		requestID, agentID, uid, constraints,
	)
	if err != nil {
		t.Fatalf("insert request: %v", err)
	}

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approval-requests/"+requestID+"/approve", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp approveStandingApprovalRequestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.StandingApproval == nil || resp.StandingApproval.SourceActionConfigurationID == nil {
		t.Fatal("expected standing approval with source_action_configuration_id")
	}
	got := *resp.StandingApproval.SourceActionConfigurationID
	if got != newerID {
		t.Errorf("expected newest config %q, got %q (older was %q)", newerID, got, olderID)
	}
}

func TestApproveStandingApprovalRequest_UnknownActionType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	requestID := testhelper.GenerateID(t, "sar_")
	constraints := []byte(`{"anything":"goes"}`)
	_, err := tx.Exec(t.Context(),
		`INSERT INTO standing_approval_requests
		   (request_id, agent_id, user_id, action_type, action_version, constraints, status)
		 VALUES ($1, $2, $3, 'unknown.action', '1', $4, 'pending')`,
		requestID, agentID, uid, constraints,
	)
	if err != nil {
		t.Fatalf("insert request: %v", err)
	}

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approval-requests/"+requestID+"/approve", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "not registered for connector") {
		t.Errorf("expected clear error about unregistered action type, got: %s", body)
	}
}
