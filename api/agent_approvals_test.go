package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// testActionSigningKey generates a fresh ECDSA P-256 key pair for testing.
func testActionSigningKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA test key: %v", err)
	}
	return key
}

// testDepsWithSigningKey creates a Deps with a fresh action token signing key.
func testDepsWithSigningKey(t *testing.T, tx db.DBTX) *Deps {
	t.Helper()
	return &Deps{
		DB:                    tx,
		SupabaseJWTSecret:     testJWTSecret,
		ActionTokenSigningKey: testActionSigningKey(t),
		ActionTokenKeyID:      "test-key-1",
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
	if !resp.VerificationRequired {
		t.Error("expected verification_required to be true")
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

// ── POST /approvals/{approval_id}/verify ───────────────────────────────────

func TestAgentVerifyApproval_Success(t *testing.T) {
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

	deps := testDepsWithSigningKey(t, tx)
	router := NewRouter(deps)

	// First, approve the approval via the dashboard.
	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/approve", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var approveResp approveResponse
	json.Unmarshal(w.Body.Bytes(), &approveResp)

	// Now verify the approval as the agent.
	verifyBody := `{"confirmation_code":"` + approveResp.ConfirmationCode + `"}`
	r2 := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/verify", verifyBody, privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var verifyResp agentVerifyApprovalResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if verifyResp.Status != "approved" {
		t.Errorf("expected status 'approved', got %q", verifyResp.Status)
	}
	if verifyResp.ApprovedAt.IsZero() {
		t.Error("expected approved_at to be set")
	}
	if verifyResp.Token == nil {
		t.Fatal("expected token to be present")
	}
	if verifyResp.Token.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if verifyResp.Token.ExpiresAt.IsZero() {
		t.Error("expected token expires_at to be set")
	}
	if verifyResp.Token.ExpiresAt.Before(time.Now()) {
		t.Error("expected token expires_at to be in the future")
	}

	// Second verify with same code should fail — code is invalidated on success.
	r3 := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/verify", verifyBody, privKey, agentID)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, r3)

	if w3.Code != http.StatusUnprocessableEntity {
		t.Fatalf("re-verify: expected 422, got %d: %s", w3.Code, w3.Body.String())
	}
}

func TestAgentVerifyApproval_WrongCode(t *testing.T) {
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

	// Approve the approval.
	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/approve", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d", w.Code)
	}

	// Try to verify with wrong code.
	verifyBody := `{"confirmation_code":"AAA-BBB"}`
	r2 := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/verify", verifyBody, privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w2.Code, w2.Body.String())
	}

	var errResp ErrorResponse
	json.Unmarshal(w2.Body.Bytes(), &errResp)
	if errResp.Error.Code != ErrInvalidCode {
		t.Errorf("expected error code %q, got %q", ErrInvalidCode, errResp.Error.Code)
	}
}

func TestAgentVerifyApproval_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	verifyBody := `{"confirmation_code":"AAA-BBB"}`
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)
	r := signedJSONRequest(t, http.MethodPost, "/approvals/appr_nonexistent/verify", verifyBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentVerifyApproval_StillPending(t *testing.T) {
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

	// Try to verify without approving first — should get conflict.
	verifyBody := `{"confirmation_code":"AAA-BBB"}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/verify", verifyBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentVerifyApproval_Expired(t *testing.T) {
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

	// Insert an approved approval that is already expired.
	testhelper.MustExec(t, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, confirmation_code_hash, expires_at, approved_at)
		 VALUES ($1, $2, $3, '{"type":"test"}', '{"description":"test"}', 'approved', 'somehash', $4, now())`,
		apprID, agentID, uid, time.Now().Add(-1*time.Hour))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	verifyBody := `{"confirmation_code":"AAA-BBB"}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/verify", verifyBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentVerifyApproval_LockoutAfterMaxAttempts(t *testing.T) {
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

	// Insert an approved approval with attempts at the limit.
	testhelper.MustExec(t, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, confirmation_code_hash, verification_attempts, expires_at, approved_at)
		 VALUES ($1, $2, $3, '{"type":"test"}', '{"description":"test"}', 'approved', 'somehash', 5, now() + interval '1 hour', now())`,
		apprID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	verifyBody := `{"confirmation_code":"AAA-BBB"}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/verify", verifyBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != ErrVerificationLocked {
		t.Errorf("expected error code %q, got %q", ErrVerificationLocked, errResp.Error.Code)
	}
}

func TestAgentVerifyApproval_OtherAgentsApproval(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Create two agents.
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

	// Create approval for agent 1.
	apprID := testhelper.GenerateID(t, "appr_")
	testhelper.InsertApprovalWithStatus(t, tx, apprID, agent1, uid, "approved")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Agent 2 tries to verify agent 1's approval — should get 404.
	verifyBody := `{"confirmation_code":"AAA-BBB"}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/verify", verifyBody, privKey2, agent2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentVerifyApproval_Unauthenticated(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/approvals/appr_xyz/verify", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
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

// ── Full end-to-end flow: request → approve → verify ───────────────────────

func TestAgentApproval_FullFlow(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	username := "u_" + uid[:8]
	testhelper.InsertUser(t, tx, uid, username)

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := testDepsWithSigningKey(t, tx)
	router := NewRouter(deps)

	// 1. Agent requests approval.
	reqBody := `{"request_id":"req_full_flow","action":{"type":"github.create_issue","version":"1","parameters":{"repo":"test/repo"}},"context":{"description":"Create GitHub issue"}}`
	r1 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)

	if w1.Code != http.StatusOK {
		t.Fatalf("request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}

	var createResp agentRequestApprovalResponse
	json.Unmarshal(w1.Body.Bytes(), &createResp)

	// 2. User approves via dashboard.
	r2 := authenticatedRequest(t, http.MethodPost, "/approvals/"+createResp.ApprovalID+"/approve", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var approveResp approveResponse
	json.Unmarshal(w2.Body.Bytes(), &approveResp)

	// 3. Agent verifies with confirmation code.
	verifyBody := `{"confirmation_code":"` + approveResp.ConfirmationCode + `"}`
	r3 := signedJSONRequest(t, http.MethodPost, "/approvals/"+createResp.ApprovalID+"/verify", verifyBody, privKey, agentID)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, r3)

	if w3.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}

	var verifyResp agentVerifyApprovalResponse
	json.Unmarshal(w3.Body.Bytes(), &verifyResp)
	if verifyResp.Status != "approved" {
		t.Errorf("expected status 'approved', got %q", verifyResp.Status)
	}
	if verifyResp.Token == nil {
		t.Fatal("expected token in verify response")
	}
	if verifyResp.Token.Scope != "github.create_issue" {
		t.Errorf("expected scope 'github.create_issue', got %q", verifyResp.Token.Scope)
	}
	if verifyResp.Token.ScopeVersion != "1" {
		t.Errorf("expected scope_version '1', got %q", verifyResp.Token.ScopeVersion)
	}

	// 4. Verify JWT claims by parsing the token.
	token, err := jwt.ParseWithClaims(verifyResp.Token.AccessToken, &ActionTokenClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return &deps.ActionTokenSigningKey.PublicKey, nil
		})
	if err != nil {
		t.Fatalf("parse JWT: %v", err)
	}
	claims := token.Claims.(*ActionTokenClaims)
	if claims.Subject != strconv.FormatInt(agentID, 10) {
		t.Errorf("expected sub %q, got %q", strconv.FormatInt(agentID, 10), claims.Subject)
	}
	if claims.Approver != username {
		t.Errorf("expected approver %q, got %q", username, claims.Approver)
	}
	if claims.ApprovalID != createResp.ApprovalID {
		t.Errorf("expected approval_id %q, got %q", createResp.ApprovalID, claims.ApprovalID)
	}
	if claims.Scope != "github.create_issue" {
		t.Errorf("expected scope 'github.create_issue', got %q", claims.Scope)
	}
	if claims.ParamsHash == "" {
		t.Error("expected non-empty params_hash")
	}
	if claims.ID == "" {
		t.Error("expected non-empty jti")
	}
	if !strings.HasPrefix(claims.ID, "tok_") {
		t.Errorf("expected jti to start with 'tok_', got %q", claims.ID)
	}

	// 5. Verify JTI was persisted to the approval row.
	var tokenJTI *string
	err = tx.QueryRow(context.Background(),
		`SELECT token_jti FROM approvals WHERE approval_id = $1`, createResp.ApprovalID,
	).Scan(&tokenJTI)
	if err != nil {
		t.Fatalf("query token_jti: %v", err)
	}
	if tokenJTI == nil {
		t.Fatal("expected token_jti to be set in DB")
	}
	if *tokenJTI != claims.ID {
		t.Errorf("expected DB token_jti %q to match JWT jti %q", *tokenJTI, claims.ID)
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

// ── Token structure and claims tests ────────────────────────────────────────

func TestAgentVerifyApproval_TokenClaimsStructure(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	username := "u_" + uid[:8]
	testhelper.InsertUser(t, tx, uid, username)

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := testDepsWithSigningKey(t, tx)
	router := NewRouter(deps)

	// Create, approve, and verify an approval.
	reqBody := `{"request_id":"req_token_claims","action":{"type":"email.send","version":"2","parameters":{"to":"alice@example.com","subject":"hi"}},"context":{"description":"test"}}`
	r1 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	var createResp agentRequestApprovalResponse
	json.Unmarshal(w1.Body.Bytes(), &createResp)

	r2 := authenticatedRequest(t, http.MethodPost, "/approvals/"+createResp.ApprovalID+"/approve", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var approveResp approveResponse
	json.Unmarshal(w2.Body.Bytes(), &approveResp)

	verifyBody := `{"confirmation_code":"` + approveResp.ConfirmationCode + `"}`
	r3 := signedJSONRequest(t, http.MethodPost, "/approvals/"+createResp.ApprovalID+"/verify", verifyBody, privKey, agentID)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, r3)
	if w3.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}

	var verifyResp agentVerifyApprovalResponse
	json.Unmarshal(w3.Body.Bytes(), &verifyResp)

	// Parse and validate all JWT claims.
	token, err := jwt.ParseWithClaims(verifyResp.Token.AccessToken, &ActionTokenClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return &deps.ActionTokenSigningKey.PublicKey, nil
		})
	if err != nil {
		t.Fatalf("parse JWT: %v", err)
	}

	// Verify header.
	if token.Header["alg"] != "ES256" {
		t.Errorf("expected alg ES256, got %v", token.Header["alg"])
	}
	if token.Header["kid"] != "test-key-1" {
		t.Errorf("expected kid 'test-key-1', got %v", token.Header["kid"])
	}

	claims := token.Claims.(*ActionTokenClaims)

	// sub = agent ID.
	if claims.Subject != strconv.FormatInt(agentID, 10) {
		t.Errorf("expected sub %q, got %q", strconv.FormatInt(agentID, 10), claims.Subject)
	}

	// aud = permissionslip.dev.
	aud, _ := claims.GetAudience()
	if len(aud) != 1 || aud[0] != "permissionslip.dev" {
		t.Errorf("expected aud [permissionslip.dev], got %v", aud)
	}

	// approver = username.
	if claims.Approver != username {
		t.Errorf("expected approver %q, got %q", username, claims.Approver)
	}

	// approval_id.
	if claims.ApprovalID != createResp.ApprovalID {
		t.Errorf("expected approval_id %q, got %q", createResp.ApprovalID, claims.ApprovalID)
	}

	// scope = action type.
	if claims.Scope != "email.send" {
		t.Errorf("expected scope 'email.send', got %q", claims.Scope)
	}

	// scope_version from action.
	if claims.ScopeVersion != "2" {
		t.Errorf("expected scope_version '2', got %q", claims.ScopeVersion)
	}

	// params_hash is SHA-256 of JCS-canonicalized parameters.
	if claims.ParamsHash == "" {
		t.Error("expected non-empty params_hash")
	}
	// Verify hash matches by computing it ourselves.
	expectedHash, err := HashParameters(json.RawMessage(`{"to":"alice@example.com","subject":"hi"}`))
	if err != nil {
		t.Fatalf("compute expected hash: %v", err)
	}
	if claims.ParamsHash != expectedHash {
		t.Errorf("expected params_hash %q, got %q", expectedHash, claims.ParamsHash)
	}

	// iat and exp.
	if claims.IssuedAt == nil || claims.IssuedAt.Time.IsZero() {
		t.Error("expected iat to be set")
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.IsZero() {
		t.Error("expected exp to be set")
	}
	ttl := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if ttl > 5*time.Minute+time.Second {
		t.Errorf("expected TTL ≤5 min, got %v", ttl)
	}

	// jti starts with "tok_".
	if !strings.HasPrefix(claims.ID, "tok_") {
		t.Errorf("expected jti prefix 'tok_', got %q", claims.ID)
	}

	// Response scope fields match JWT claims.
	if verifyResp.Token.Scope != claims.Scope {
		t.Errorf("response scope %q doesn't match JWT scope %q", verifyResp.Token.Scope, claims.Scope)
	}
	if verifyResp.Token.ScopeVersion != claims.ScopeVersion {
		t.Errorf("response scope_version %q doesn't match JWT scope_version %q", verifyResp.Token.ScopeVersion, claims.ScopeVersion)
	}
}

func TestAgentVerifyApproval_TokenDefaultScopeVersion(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := testDepsWithSigningKey(t, tx)
	router := NewRouter(deps)

	// Action without explicit version — should default to "1".
	reqBody := `{"request_id":"req_default_ver","action":{"type":"email.send","parameters":{}},"context":{"description":"test"}}`
	r1 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	var createResp agentRequestApprovalResponse
	json.Unmarshal(w1.Body.Bytes(), &createResp)

	r2 := authenticatedRequest(t, http.MethodPost, "/approvals/"+createResp.ApprovalID+"/approve", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d", w2.Code)
	}
	var approveResp approveResponse
	json.Unmarshal(w2.Body.Bytes(), &approveResp)

	verifyBody := `{"confirmation_code":"` + approveResp.ConfirmationCode + `"}`
	r3 := signedJSONRequest(t, http.MethodPost, "/approvals/"+createResp.ApprovalID+"/verify", verifyBody, privKey, agentID)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, r3)
	if w3.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}

	var verifyResp agentVerifyApprovalResponse
	json.Unmarshal(w3.Body.Bytes(), &verifyResp)
	if verifyResp.Token.ScopeVersion != "1" {
		t.Errorf("expected default scope_version '1', got %q", verifyResp.Token.ScopeVersion)
	}
}

func TestAgentVerifyApproval_JTIPersisted(t *testing.T) {
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

	deps := testDepsWithSigningKey(t, tx)
	router := NewRouter(deps)

	// Approve.
	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/approve", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d", w.Code)
	}
	var approveResp approveResponse
	json.Unmarshal(w.Body.Bytes(), &approveResp)

	// Verify.
	verifyBody := `{"confirmation_code":"` + approveResp.ConfirmationCode + `"}`
	r2 := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/verify", verifyBody, privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Check that token_jti was written to the approval row.
	var tokenJTI *string
	err = tx.QueryRow(context.Background(),
		`SELECT token_jti FROM approvals WHERE approval_id = $1`, apprID,
	).Scan(&tokenJTI)
	if err != nil {
		t.Fatalf("query token_jti: %v", err)
	}
	if tokenJTI == nil {
		t.Fatal("expected token_jti to be set")
	}
	if !strings.HasPrefix(*tokenJTI, "tok_") {
		t.Errorf("expected token_jti to start with 'tok_', got %q", *tokenJTI)
	}
}

func TestAgentVerifyApproval_TokenExpiryNotBeyondApproval(t *testing.T) {
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

	// Create an approved approval that expires in 2 minutes (less than 5-min token TTL).
	approvalExpiry := time.Now().Add(2 * time.Minute)
	testhelper.MustExec(t, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, confirmation_code_hash, expires_at, approved_at)
		 VALUES ($1, $2, $3, '{"type":"test"}', '{"description":"test"}', 'approved', $4, $5, now())`,
		apprID, agentID, uid, hashCodeHex("AAAAAA", ""), approvalExpiry)

	deps := testDepsWithSigningKey(t, tx)
	router := NewRouter(deps)

	verifyBody := `{"confirmation_code":"AAAAAA"}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/"+apprID+"/verify", verifyBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var verifyResp agentVerifyApprovalResponse
	json.Unmarshal(w.Body.Bytes(), &verifyResp)

	// Token should not expire after the approval does.
	if verifyResp.Token.ExpiresAt.After(approvalExpiry.Add(time.Second)) {
		t.Errorf("token expires_at %v should not exceed approval expires_at %v",
			verifyResp.Token.ExpiresAt, approvalExpiry)
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
