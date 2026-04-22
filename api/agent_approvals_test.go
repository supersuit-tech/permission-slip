package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/connectors/slack"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/vault"
)

// testDepsForDB creates a Deps suitable for tests that only need a DB and JWT secret.
func testDepsForDB(t *testing.T, tx db.DBTX) *Deps {
	t.Helper()
	return &Deps{
		DB:                tx,
		SupabaseJWTSecret: testJWTSecret,
	}
}

// ── POST /approvals/request ────────────────────────────────────────────────

func TestAgentRequestApproval_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_001","action":{"type":"email.send","parameters":{"to":"alice@example.com"}},"context":{"description":"Send email to Alice"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ApprovalID == "" {
		t.Error("expected non-empty approval_id")
	}
	if resp.ApprovalURL == "" {
		t.Error("expected non-empty approval_url")
	}
	if resp.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", resp.Status)
	}
	if resp.ExpiresAt.IsZero() {
		t.Error("expected expires_at to be set")
	}
	if resp.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestAgentRequestApproval_UnknownActionType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Set up a connector registry with only "google" connector registered.
	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector("google", "google.list_emails"))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, Connectors: reg}
	router := NewRouter(deps)

	// Use "gmail.list_messages" — a plausible but unregistered action type.
	reqBody := `{"request_id":"req_bad","action":{"type":"gmail.list_messages","parameters":{}},"context":{"description":"List emails"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown action type, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error.Message == "" {
		t.Error("expected error message in response")
	}
	if resp.Error.Code != ErrUnsupportedActionType {
		t.Errorf("expected error code %q, got %q", ErrUnsupportedActionType, resp.Error.Code)
	}
}

func TestAgentRequestApproval_KnownActionType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Set up a connector registry with "google" connector registered.
	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector("google", "google.list_emails"))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, Connectors: reg}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_known","action":{"type":"google.list_emails","parameters":{}},"context":{"description":"List emails"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for known action type, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ApprovalID == "" {
		t.Error("expected non-empty approval_id")
	}
	if resp.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", resp.Status)
	}
}

func TestAgentRequestApproval_CustomExpiresIn(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_002","action":{"type":"email.send"},"context":{"description":"test"},"expires_in":3600}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Check that expires_at is roughly 1 hour from now.
	expectedExpiry := time.Now().Add(time.Hour)
	diff := resp.ExpiresAt.Sub(expectedExpiry)
	if diff < 0 {
		diff = -diff
	}
	if diff > 30*time.Second {
		t.Errorf("expected expires_at ~1h from now, got %v (diff: %v)", resp.ExpiresAt, diff)
	}
}

func TestAgentRequestApproval_DuplicateRequestID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_dup_test","action":{"type":"email.send"},"context":{"description":"test"}}`

	// First request — should succeed.
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Second request with same request_id — should fail with 409.
	r2 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate request: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrDuplicateRequestID {
		t.Errorf("expected error code %q, got %q", ErrDuplicateRequestID, errResp.Error.Code)
	}
}

func TestAgentRequestApproval_MissingFields(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	tests := []struct {
		name string
		body string
	}{
		{"missing request_id", `{"action":{"type":"test"},"context":{"description":"test"}}`},
		{"missing action", `{"request_id":"req_1","context":{"description":"test"}}`},
		{"missing context", `{"request_id":"req_2","action":{"type":"test"}}`},
		{"null action", `{"request_id":"req_3","action":null,"context":{"description":"test"}}`},
		{"null context", `{"request_id":"req_4","action":{"type":"test"},"context":null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := signedJSONRequest(t, http.MethodPost, "/approvals/request", tt.body, privKey, agentID)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAgentRequestApproval_MissingActionType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_no_type","action":{"parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentRequestApproval_InvalidExpiresIn(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// expires_in too small (< 60 seconds).
	reqBody := `{"request_id":"req_bad_expiry","action":{"type":"test"},"context":{"description":"test"},"expires_in":10}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentRequestApproval_Unauthenticated(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentRequestApproval_ApprovalAppearsInDashboardList(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Create an approval as the agent.
	reqBody := `{"request_id":"req_list_test","action":{"type":"email.send"},"context":{"description":"Send email"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var createResp agentRequestApprovalResponse
	json.Unmarshal(w.Body.Bytes(), &createResp)

	// Verify it appears in the dashboard list.
	r2 := authenticatedRequest(t, http.MethodGet, "/approvals", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var list approvalListResponse
	json.Unmarshal(w2.Body.Bytes(), &list)

	found := false
	for _, a := range list.Data {
		if a.ApprovalID == createResp.ApprovalID {
			found = true
			if a.AgentID != agentID {
				t.Errorf("expected agent_id %d, got %d", agentID, a.AgentID)
			}
			if a.Status != "pending" {
				t.Errorf("expected status 'pending', got %q", a.Status)
			}
			break
		}
	}
	if !found {
		t.Error("agent-created approval not found in dashboard list")
	}
}

// ── POST /approvals/{approval_id}/cancel ───────────────────────────────────

func TestAgentCancelApproval_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)
	apprID := testhelper.GenerateID(t, "appr_")
	testhelper.InsertApproval(t, tx, apprID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/cancel", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentCancelApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ApprovalID != apprID {
		t.Errorf("expected approval_id %q, got %q", apprID, resp.ApprovalID)
	}
	if resp.Status != "cancelled" {
		t.Errorf("expected status 'cancelled', got %q", resp.Status)
	}
	if resp.CancelledAt.IsZero() {
		t.Error("expected cancelled_at to be set")
	}

	// Verify it no longer appears in the pending list.
	r2 := authenticatedRequest(t, http.MethodGet, "/approvals", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	list := decodeApprovalList(t, w2.Body.Bytes())
	for _, a := range list.Data {
		if a.ApprovalID == apprID {
			t.Error("cancelled approval should not appear in pending list")
		}
	}
}

func TestAgentCancelApproval_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := signedJSONRequest(t, http.MethodPost, "/approvals/appr_nonexistent/cancel", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentCancelApproval_AlreadyApproved(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)
	apprID := testhelper.GenerateID(t, "appr_")
	testhelper.InsertApprovalWithStatus(t, tx, apprID, agentID, uid, "approved")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/cancel", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != ErrApprovalAlreadyResolved {
		t.Errorf("expected error code %q, got %q", ErrApprovalAlreadyResolved, errResp.Error.Code)
	}
}

func TestAgentCancelApproval_Expired(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)
	apprID := testhelper.GenerateID(t, "appr_")
	testhelper.InsertApprovalWithExpiresAt(t, tx, apprID, agentID, uid, time.Now().Add(-1*time.Hour))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/cancel", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentCancelApproval_OtherAgentsApproval(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH1, _, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agent1 := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH1)

	pubKeySSH2, privKey2, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agent2 := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH2)

	apprID := testhelper.GenerateID(t, "appr_")
	testhelper.InsertApproval(t, tx, apprID, agent1, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Agent 2 tries to cancel agent 1's approval — should get 404.
	r := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/cancel", "", privKey2, agent2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentCancelApproval_Unauthenticated(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/approvals/appr_xyz/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ── Full flow: request → cancel ────────────────────────────────────────────

func TestAgentApproval_RequestThenCancel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// 1. Agent requests approval.
	reqBody := `{"request_id":"req_cancel_flow","action":{"type":"test.action"},"context":{"description":"test"}}`
	r1 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)

	if w1.Code != http.StatusOK {
		t.Fatalf("request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}

	var createResp agentRequestApprovalResponse
	json.Unmarshal(w1.Body.Bytes(), &createResp)

	// 2. Agent cancels the approval.
	r2 := signedJSONRequest(t, http.MethodPost, "/approvals/"+createResp.ApprovalID+"/cancel", "", privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("cancel: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var cancelResp agentCancelApprovalResponse
	json.Unmarshal(w2.Body.Bytes(), &cancelResp)
	if cancelResp.Status != "cancelled" {
		t.Errorf("expected status 'cancelled', got %q", cancelResp.Status)
	}

	// 3. User cannot approve a cancelled approval.
	r3 := authenticatedRequest(t, http.MethodPost, "/approvals/"+createResp.ApprovalID+"/approve", uid)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, r3)

	if w3.Code != http.StatusConflict {
		t.Fatalf("approve after cancel: expected 409, got %d: %s", w3.Code, w3.Body.String())
	}
}

// ── Audit event verification ───────────────────────────────────────────────

func TestAgentRequestApproval_EmitsAuditEvent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_audit","action":{"type":"email.send"},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check that an audit event was created.
	var count int
	err = tx.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_events WHERE user_id = $1 AND event_type = 'approval.requested'`,
		uid,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query audit events: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 approval.requested audit event, got %d", count)
	}
}

// ── GET /approvals/{approval_id}/status ──────────────────────────────────

func TestAgentApprovalStatus_Pending(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	// 1. Create an approval.
	reqBody := `{"request_id":"req_status_pending","action":{"type":"test.action"},"context":{"description":"test"}}`
	r1 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	var createResp agentRequestApprovalResponse
	json.Unmarshal(w1.Body.Bytes(), &createResp)

	// 2. Poll status — should be pending with no execution fields.
	r2 := signedJSONRequest(t, http.MethodGet, "/approvals/"+createResp.ApprovalID+"/status", "", privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var statusResp agentApprovalStatusResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if statusResp.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", statusResp.Status)
	}
	if statusResp.ExecutionStatus != nil {
		t.Errorf("expected nil execution_status for pending approval, got %v", *statusResp.ExecutionStatus)
	}
	if statusResp.ExecutionResult != nil {
		t.Error("expected nil execution_result for pending approval")
	}
}

func TestAgentApprovalStatus_ApprovedWithExecution(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Insert an approved approval with execution result.
	approvalID := "appr_status_exec"
	testhelper.InsertApprovalWithStatus(t, tx, approvalID, agentID, uid, "approved")
	_, err = tx.Exec(context.Background(),
		`UPDATE approvals SET execution_status = 'success', execution_result = '{"data":"ok"}', executed_at = now() WHERE approval_id = $1`,
		approvalID,
	)
	if err != nil {
		t.Fatalf("set execution: %v", err)
	}

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	r := signedJSONRequest(t, http.MethodGet, "/approvals/"+approvalID+"/status", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var statusResp agentApprovalStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if statusResp.Status != "approved" {
		t.Errorf("expected status 'approved', got %q", statusResp.Status)
	}
	if statusResp.ExecutionStatus == nil || *statusResp.ExecutionStatus != "success" {
		t.Errorf("expected execution_status 'success', got %v", statusResp.ExecutionStatus)
	}
	if statusResp.ExecutionResult == nil {
		t.Fatal("expected execution_result to be present")
	}
}

func TestAgentApprovalStatus_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	r := signedJSONRequest(t, http.MethodGet, "/approvals/appr_nonexistent/status", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentApprovalStatus_Denied(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	approvalID := "appr_status_denied"
	testhelper.InsertApprovalWithStatus(t, tx, approvalID, agentID, uid, "denied")

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	r := signedJSONRequest(t, http.MethodGet, "/approvals/"+approvalID+"/status", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var statusResp agentApprovalStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if statusResp.Status != "denied" {
		t.Errorf("expected status 'denied', got %q", statusResp.Status)
	}
	if statusResp.ExecutionStatus != nil {
		t.Errorf("expected nil execution_status for denied approval, got %v", *statusResp.ExecutionStatus)
	}
	if statusResp.ExecutionResult != nil {
		t.Error("expected nil execution_result for denied approval")
	}
}

func TestAgentApprovalStatus_FullApproveFlow(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	// 1. Agent requests approval.
	reqBody := `{"request_id":"req_full_flow","action":{"type":"test.action"},"context":{"description":"test"}}`
	r1 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	var createResp agentRequestApprovalResponse
	json.Unmarshal(w1.Body.Bytes(), &createResp)

	// 2. User approves the approval (execution happens inline).
	r2 := authenticatedRequest(t, http.MethodPost, "/approvals/"+createResp.ApprovalID+"/approve", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// 3. Agent polls for status — should see approved with execution result.
	r3 := signedJSONRequest(t, http.MethodGet, "/approvals/"+createResp.ApprovalID+"/status", "", privKey, agentID)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, r3)
	if w3.Code != http.StatusOK {
		t.Fatalf("status: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}

	var statusResp agentApprovalStatusResponse
	if err := json.Unmarshal(w3.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if statusResp.Status != "approved" {
		t.Errorf("expected status 'approved', got %q", statusResp.Status)
	}
	// No connector in test, so execution_status should be "error".
	if statusResp.ExecutionStatus == nil {
		t.Fatal("expected execution_status to be set after approval")
	}
	if *statusResp.ExecutionStatus != "error" {
		t.Errorf("expected execution_status 'error' (no connector), got %q", *statusResp.ExecutionStatus)
	}
	if statusResp.ExecutionResult == nil {
		t.Fatal("expected execution_result to be set after approval")
	}
}

func TestAgentApprovalStatus_OtherAgentAccess(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	// Create two users, each with their own agent.
	uid1 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u1_"+uid1[:8])
	pubKeySSH1, _, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key1: %v", err)
	}
	agentID1 := testhelper.InsertAgentWithPublicKey(t, tx, uid1, "registered", pubKeySSH1)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:8])
	pubKeySSH2, privKey2, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key2: %v", err)
	}
	agentID2 := testhelper.InsertAgentWithPublicKey(t, tx, uid2, "registered", pubKeySSH2)
	_ = agentID2

	// Create an approval for agent1.
	approvalID := "appr_other_agent"
	testhelper.InsertApproval(t, tx, approvalID, agentID1, uid1)

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	// Agent2 tries to access agent1's approval — should get 404.
	r := signedJSONRequest(t, http.MethodGet, "/approvals/"+approvalID+"/status", "", privKey2, agentID2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for other agent's approval, got %d: %s", w.Code, w.Body.String())
	}
}

// ── RequestValidator early rejection ──────────────────────────────────────

func TestAgentRequestApproval_RequestValidatorRejectsInvalidChannel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Register the Slack connector so RequestValidator is exercised.
	registry := connectors.NewRegistry()
	registry.Register(slack.New())
	deps := &Deps{
		DB:                tx,
		SupabaseJWTSecret: testJWTSecret,
		Connectors:        registry,
	}
	router := NewRouter(deps)

	// Agent sends a user ID (U prefix) instead of a channel ID (C/G/D prefix).
	reqBody := `{"request_id":"req_bad_channel","action":{"type":"slack.read_channel_messages","parameters":{"channel":"U0AM0RW432Q"}},"context":{"description":"Read messages"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid channel ID, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrInvalidRequest {
		t.Errorf("expected error code %q, got %q", ErrInvalidRequest, errResp.Error.Code)
	}

	// Verify the error message mentions the channel ID issue.
	if errResp.Error.Message == "" {
		t.Error("expected error message to be non-empty")
	}
}

func TestAgentRequestApproval_RequestValidatorAllowsValidChannel(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	registry := connectors.NewRegistry()
	registry.Register(slack.New())
	deps := &Deps{
		DB:                tx,
		SupabaseJWTSecret: testJWTSecret,
		Connectors:        registry,
	}
	router := NewRouter(deps)

	// Valid channel ID (C prefix) should be accepted.
	reqBody := `{"request_id":"req_good_channel","action":{"type":"slack.read_channel_messages","parameters":{"channel":"C01234567"}},"context":{"description":"Read messages"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid channel ID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentRequestApproval_MultiInstance_UsesDefaultWhenUnset(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	connID := "mi_test_conn"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".ping", "Ping")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	ctx := context.Background()
	if _, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	}); err != nil {
		t.Fatalf("second instance: %v", err)
	}

	v := vault.NewMockVaultStore()
	credJSON, _ := json.Marshal(map[string]string{"api_key": "k1"})
	v1, err := v.CreateSecret(ctx, tx, "c1", credJSON)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID1 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID1, uid, connID, "Alpha", v1)

	credJSON2, _ := json.Marshal(map[string]string{"api_key": "k2"})
	v2, err := v.CreateSecret(ctx, tx, "c2", credJSON2)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID2 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID2, uid, connID, "Beta", v2)

	defInst, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || defInst == nil {
		t.Fatalf("default instance: %v", defInst)
	}
	instances, err := db.ListAgentConnectorInstances(ctx, tx, agentID, uid, connID)
	if err != nil || len(instances) != 2 {
		t.Fatalf("instances: %v len=%d", err, len(instances))
	}
	var secondID string
	for _, inst := range instances {
		if inst.ConnectorInstanceID != defInst.ConnectorInstanceID {
			secondID = inst.ConnectorInstanceID
			break
		}
	}
	if secondID == "" {
		t.Fatal("expected two distinct instances")
	}

	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: defInst.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID1,
	}); err != nil {
		t.Fatalf("bind default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: secondID, ApproverID: uid, CredentialID: &credID2,
	}); err != nil {
		t.Fatalf("bind second: %v", err)
	}

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, connID+".ping"))
	deps := &Deps{DB: tx, Vault: v, SupabaseJWTSecret: testJWTSecret, Connectors: reg}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_mi_need_inst","action":{"type":"` + connID + `.ping","parameters":{}},"context":{"description":"x"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (default instance), got %d: %s", w.Code, w.Body.String())
	}
	var createResp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	appr, err := db.GetApprovalByIDAndAgent(t.Context(), tx, createResp.ApprovalID, agentID)
	if err != nil || appr == nil {
		t.Fatalf("get approval: %v", err)
	}
	var actionObj map[string]json.RawMessage
	if err := json.Unmarshal(appr.Action, &actionObj); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}
	var gotID string
	_ = json.Unmarshal(actionObj["_connector_instance_id"], &gotID)
	if gotID != defInst.ConnectorInstanceID {
		t.Errorf("expected frozen instance %q, got %q", defInst.ConnectorInstanceID, gotID)
	}
}

func TestAgentRequestApproval_MultiInstance_NoDefault_RequiresConnectorInstance(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	connID := "mi_test_conn_nd"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".ping", "Ping")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	ctx := context.Background()
	inst2, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}

	v := vault.NewMockVaultStore()
	credJSON, _ := json.Marshal(map[string]string{"api_key": "k1"})
	v1, err := v.CreateSecret(ctx, tx, "c1", credJSON)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID1 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID1, uid, connID, "Alpha", v1)

	credJSON2, _ := json.Marshal(map[string]string{"api_key": "k2"})
	v2, err := v.CreateSecret(ctx, tx, "c2", credJSON2)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID2 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID2, uid, connID, "Beta", v2)

	defInst, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || defInst == nil {
		t.Fatalf("default instance: %v", defInst)
	}
	instances, err := db.ListAgentConnectorInstances(ctx, tx, agentID, uid, connID)
	if err != nil || len(instances) != 2 {
		t.Fatalf("instances: %v len=%d", err, len(instances))
	}
	var secondID string
	for _, inst := range instances {
		if inst.ConnectorInstanceID != defInst.ConnectorInstanceID {
			secondID = inst.ConnectorInstanceID
			break
		}
	}
	if secondID == "" {
		t.Fatal("expected two distinct instances")
	}

	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: defInst.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID1,
	}); err != nil {
		t.Fatalf("bind default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: secondID, ApproverID: uid, CredentialID: &credID2,
	}); err != nil {
		t.Fatalf("bind second: %v", err)
	}

	// Clear default flags — simulates legacy or inconsistent data where no instance is marked default.
	testhelper.MustExec(t, tx, `UPDATE agent_connectors SET is_default = false WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3`,
		agentID, uid, connID)
	if def2, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID); err != nil || def2 != nil {
		t.Fatalf("expected no default after update, got %v err=%v", def2, err)
	}

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, connID+".ping"))
	deps := &Deps{DB: tx, Vault: v, SupabaseJWTSecret: testJWTSecret, Connectors: reg}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_mi_need_inst_nd","action":{"type":"` + connID + `.ping","parameters":{}},"context":{"description":"x"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrConnectorInstanceRequired {
		t.Errorf("expected %q, got %q", ErrConnectorInstanceRequired, errResp.Error.Code)
	}
	rawList, _ := errResp.Error.Details["available_instances"].([]any)
	if len(rawList) != 2 {
		t.Fatalf("expected 2 available_instances, got %#v", errResp.Error.Details)
	}
	gotIDs := make(map[string]struct{})
	for _, x := range rawList {
		m, ok := x.(map[string]any)
		if !ok {
			t.Fatalf("expected object entries in available_instances, got %#v", x)
		}
		id, _ := m["id"].(string)
		gotIDs[id] = struct{}{}
	}
	if _, ok := gotIDs[defInst.ConnectorInstanceID]; !ok {
		t.Errorf("missing default id %q in %#v", defInst.ConnectorInstanceID, rawList)
	}
	if _, ok := gotIDs[inst2.ConnectorInstanceID]; !ok {
		t.Errorf("missing second id %q in %#v", inst2.ConnectorInstanceID, rawList)
	}
}

func TestAgentRequestApproval_MultiInstance_ExplicitSelector_OverridesDefault(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	connID := "mi_test_conn_ov"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".ping", "Ping")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	ctx := context.Background()
	inst2, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}

	v := vault.NewMockVaultStore()
	credJSON, _ := json.Marshal(map[string]string{"api_key": "k1"})
	v1, err := v.CreateSecret(ctx, tx, "c1", credJSON)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID1 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID1, uid, connID, "Alpha", v1)

	credJSON2, _ := json.Marshal(map[string]string{"api_key": "k2"})
	v2, err := v.CreateSecret(ctx, tx, "c2", credJSON2)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID2 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID2, uid, connID, "Beta", v2)

	defInst, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || defInst == nil {
		t.Fatalf("default instance: %v", defInst)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: defInst.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID1,
	}); err != nil {
		t.Fatalf("bind default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: inst2.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID2,
	}); err != nil {
		t.Fatalf("bind second: %v", err)
	}

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, connID+".ping"))
	deps := &Deps{DB: tx, Vault: v, SupabaseJWTSecret: testJWTSecret, Connectors: reg}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_mi_override","action":{"type":"` + connID + `.ping","parameters":{"connector_instance":"` + inst2.ConnectorInstanceID + `"}},"context":{"description":"x"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var createResp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	appr, err := db.GetApprovalByIDAndAgent(t.Context(), tx, createResp.ApprovalID, agentID)
	if err != nil || appr == nil {
		t.Fatalf("get approval: %v", err)
	}
	var actionObj map[string]json.RawMessage
	if err := json.Unmarshal(appr.Action, &actionObj); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}
	var gotID string
	_ = json.Unmarshal(actionObj["_connector_instance_id"], &gotID)
	if gotID != inst2.ConnectorInstanceID {
		t.Errorf("expected explicit instance %q, got %q", inst2.ConnectorInstanceID, gotID)
	}
	if gotID == defInst.ConnectorInstanceID {
		t.Error("explicit selector should not route to default")
	}
}

func TestAgentRequestApproval_MultiInstance_WithLabel_FreezesInstanceOnAction(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	connID := "mi_test_conn2"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".ping", "Ping")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	ctx := context.Background()
	inst2, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}

	v := vault.NewMockVaultStore()
	credJSON1, _ := json.Marshal(map[string]string{"api_key": "k1"})
	v1, err := v.CreateSecret(ctx, tx, "c1", credJSON1)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID1 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID1, uid, connID, "Alpha", v1)

	credJSON2, _ := json.Marshal(map[string]string{"api_key": "k2"})
	v2, err := v.CreateSecret(ctx, tx, "c2", credJSON2)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID2 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID2, uid, connID, "Sales", v2)

	defInst, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || defInst == nil {
		t.Fatalf("default instance: %v", defInst)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: defInst.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID1,
	}); err != nil {
		t.Fatalf("bind default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: inst2.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID2,
	}); err != nil {
		t.Fatalf("bind sales: %v", err)
	}

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, connID+".ping"))
	deps := &Deps{DB: tx, Vault: v, SupabaseJWTSecret: testJWTSecret, Connectors: reg}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_mi_label","action":{"type":"` + connID + `.ping","parameters":{"connector_instance":"Sales"}},"context":{"description":"x"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var createResp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	appr, err := db.GetApprovalByIDAndAgent(t.Context(), tx, createResp.ApprovalID, agentID)
	if err != nil || appr == nil {
		t.Fatalf("get approval: %v", err)
	}
	var actionObj map[string]json.RawMessage
	if err := json.Unmarshal(appr.Action, &actionObj); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}
	var gotID, gotDisplay string
	_ = json.Unmarshal(actionObj["_connector_instance_id"], &gotID)
	_ = json.Unmarshal(actionObj["_connector_instance_display"], &gotDisplay)
	if gotID != inst2.ConnectorInstanceID {
		t.Errorf("connector_instance_id: want %q got %q", inst2.ConnectorInstanceID, gotID)
	}
	if gotDisplay != "Sales" {
		t.Errorf("connector_instance_display: want Sales got %q", gotDisplay)
	}
	if raw, ok := actionObj["parameters"]; ok {
		var params map[string]json.RawMessage
		if err := json.Unmarshal(raw, &params); err != nil {
			t.Fatalf("parameters: %v", err)
		}
		if _, ok := params["connector_instance"]; ok {
			t.Error("connector_instance should be stripped from parameters")
		}
	}
}

// Regression: when connector_instance is the only parameter, the routing code
// must leave parameters as "{}" instead of deleting it. Otherwise downstream
// action parsers that json.Unmarshal(req.Parameters, &p) fail with
// "unexpected end of JSON input" and zero-param actions (list_channels,
// list_users, etc.) become unusable on any non-default instance.
func TestAgentRequestApproval_MultiInstance_OnlyConnectorInstance_PreservesEmptyParameters(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	connID := "mi_only_inst_conn"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".ping", "Ping")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	ctx := context.Background()
	inst2, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("second instance: %v", err)
	}

	v := vault.NewMockVaultStore()
	credJSON1, _ := json.Marshal(map[string]string{"api_key": "k1"})
	v1, err := v.CreateSecret(ctx, tx, "c1", credJSON1)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID1 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID1, uid, connID, "Alpha", v1)

	credJSON2, _ := json.Marshal(map[string]string{"api_key": "k2"})
	v2, err := v.CreateSecret(ctx, tx, "c2", credJSON2)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID2 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID2, uid, connID, "Beta", v2)

	defInst, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || defInst == nil {
		t.Fatalf("default instance: %v", defInst)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: defInst.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID1,
	}); err != nil {
		t.Fatalf("bind default: %v", err)
	}
	if _, err := db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: inst2.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID2,
	}); err != nil {
		t.Fatalf("bind second: %v", err)
	}

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, connID+".ping"))
	deps := &Deps{DB: tx, Vault: v, SupabaseJWTSecret: testJWTSecret, Connectors: reg}
	router := NewRouter(deps)

	// Body sends connector_instance as the ONLY parameter.
	reqBody := `{"request_id":"req_mi_only_inst","action":{"type":"` + connID + `.ping","parameters":{"connector_instance":"` + inst2.ConnectorInstanceID + `"}},"context":{"description":"x"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var createResp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	appr, err := db.GetApprovalByIDAndAgent(t.Context(), tx, createResp.ApprovalID, agentID)
	if err != nil || appr == nil {
		t.Fatalf("get approval: %v", err)
	}
	var actionObj map[string]json.RawMessage
	if err := json.Unmarshal(appr.Action, &actionObj); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}

	raw, ok := actionObj["parameters"]
	if !ok {
		t.Fatal("parameters key missing — downstream action parsers will fail with 'unexpected end of JSON input'")
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatalf("parameters must be valid JSON object, got %q: %v", string(raw), err)
	}
	if len(params) != 0 {
		t.Errorf("parameters should be empty after stripping connector_instance, got %v", params)
	}
}
