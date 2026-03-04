package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
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
