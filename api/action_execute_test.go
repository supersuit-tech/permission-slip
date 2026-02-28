package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// mintTestActionToken creates a signed action token for testing. It builds
// claims from the provided parameters and signs with the given key.
func mintTestActionToken(t *testing.T, key *ecdsa.PrivateKey, keyID string, agentID int64, approvalID, scope, scopeVersion, paramsHashVal, jti string, expiresAt time.Time) string {
	t.Helper()
	now := time.Now().UTC()
	claims := ActionTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(agentID, 10),
			Audience:  jwt.ClaimStrings{actionTokenAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
		Approver:     "testuser",
		ApprovalID:   approvalID,
		Scope:        scope,
		ScopeVersion: scopeVersion,
		ParamsHash:   paramsHashVal,
	}
	token, err := MintActionToken(key, keyID, claims)
	if err != nil {
		t.Fatalf("mint test action token: %v", err)
	}
	return token
}

// setupExecuteTest creates the full test fixture: user, agent, approval with
// JTI, and returns the deps, router, agent ID, private key, and JTI.
func setupExecuteTest(t *testing.T) (tx db.DBTX, deps *Deps, router http.Handler, agentID int64, privKey ed25519.PrivateKey, apprID, jti string) {
	t.Helper()
	txVal := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, txVal, uid, "u_"+uid[:8])

	pubKeySSH, pk, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	aid := testhelper.InsertAgentWithPublicKey(t, txVal, uid, "registered", pubKeySSH)

	apprIDVal := testhelper.GenerateID(t, "appr_")
	jtiVal := testhelper.GenerateID(t, "tok_")
	testhelper.InsertApprovalWithJTI(t, txVal, apprIDVal, aid, uid, jtiVal)

	d := testDepsWithSigningKey(t, txVal)
	r := NewRouter(d)

	return txVal, d, r, aid, pk, apprIDVal, jtiVal
}

// ── POST /actions/execute (token-based) ────────────────────────────────────

func TestExecuteActionToken_Success(t *testing.T) {
	t.Parallel()
	_, deps, router, agentID, privKey, apprID, jti := setupExecuteTest(t)

	// Compute the params hash for the parameters we'll send.
	params := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, err := HashParameters(params)
	if err != nil {
		t.Fatalf("HashParameters: %v", err)
	}

	token := mintTestActionToken(t, deps.ActionTokenSigningKey, deps.ActionTokenKeyID,
		agentID, apprID, "email.send", "1", hash, jti, time.Now().Add(5*time.Minute))

	reqBody := fmt.Sprintf(`{"token":%q,"action_id":"email.send","parameters":{"to":"alice@example.com"}}`, token)
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp executeActionTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %q", resp.Status)
	}
	if resp.ActionID != "email.send" {
		t.Errorf("expected action_id 'email.send', got %q", resp.ActionID)
	}
	if resp.ExecutedAt.IsZero() {
		t.Error("expected executed_at to be set")
	}
}

func TestExecuteActionToken_ReplayReturns403(t *testing.T) {
	t.Parallel()
	_, deps, router, agentID, privKey, apprID, jti := setupExecuteTest(t)

	params := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, _ := HashParameters(params)

	token := mintTestActionToken(t, deps.ActionTokenSigningKey, deps.ActionTokenKeyID,
		agentID, apprID, "email.send", "1", hash, jti, time.Now().Add(5*time.Minute))

	reqBody := fmt.Sprintf(`{"token":%q,"action_id":"email.send","parameters":{"to":"alice@example.com"}}`, token)

	// First request should succeed.
	r1 := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}

	// Second request (replay) should return 403 with token_already_used.
	r2 := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusForbidden {
		t.Fatalf("replay: expected 403, got %d: %s", w2.Code, w2.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if errResp.Error.Code != ErrTokenAlreadyUsed {
		t.Errorf("expected error code %q, got %q", ErrTokenAlreadyUsed, errResp.Error.Code)
	}
	if errResp.Error.Details["jti"] != jti {
		t.Errorf("expected jti %q in details, got %v", jti, errResp.Error.Details["jti"])
	}
}

func TestExecuteActionToken_ConcurrentRace(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupPool(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, pool, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, pool, uid, "registered", pubKeySSH)
	apprID := testhelper.GenerateID(t, "appr_")
	jti := testhelper.GenerateID(t, "tok_")
	testhelper.InsertApprovalWithJTI(t, pool, apprID, agentID, uid, jti)

	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM consumed_tokens WHERE jti = $1`, jti)
		pool.Exec(ctx, `DELETE FROM approvals WHERE approval_id = $1`, apprID)
		pool.Exec(ctx, `DELETE FROM agents WHERE agent_id = $1`, agentID)
		pool.Exec(ctx, `DELETE FROM profiles WHERE id = $1`, uid)
		pool.Exec(ctx, `DELETE FROM auth.users WHERE id = $1`, uid)
	})

	signingKey := testActionSigningKey(t)
	deps := &Deps{
		DB:                    pool,
		SupabaseJWTSecret:     testJWTSecret,
		ActionTokenSigningKey: signingKey,
		ActionTokenKeyID:      "test-key-1",
	}
	router := NewRouter(deps)

	params := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, _ := HashParameters(params)

	token := mintTestActionToken(t, signingKey, "test-key-1",
		agentID, apprID, "email.send", "1", hash, jti, time.Now().Add(5*time.Minute))

	reqBody := fmt.Sprintf(`{"token":%q,"action_id":"email.send","parameters":{"to":"alice@example.com"}}`, token)

	const goroutines = 5
	results := make([]int, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			results[idx] = w.Code
		}(i)
	}

	wg.Wait()

	// Exactly one should succeed (200), the rest should get 403.
	requireExactlyOneSuccess(t, results, http.StatusForbidden)
}

func TestExecuteActionToken_InvalidSignature(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, apprID, jti := setupExecuteTest(t)

	// Mint token with a different key.
	wrongKey, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
	if err != nil {
		t.Fatalf("generate wrong key: %v", err)
	}

	params := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, _ := HashParameters(params)

	token := mintTestActionToken(t, wrongKey, "wrong-key",
		agentID, apprID, "email.send", "1", hash, jti, time.Now().Add(5*time.Minute))

	reqBody := fmt.Sprintf(`{"token":%q,"action_id":"email.send","parameters":{"to":"alice@example.com"}}`, token)
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != ErrInvalidToken {
		t.Errorf("expected error code %q, got %q", ErrInvalidToken, errResp.Error.Code)
	}
}

func TestExecuteActionToken_ExpiredToken(t *testing.T) {
	t.Parallel()
	_, deps, router, agentID, privKey, apprID, jti := setupExecuteTest(t)

	params := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, _ := HashParameters(params)

	// Mint an already-expired token.
	token := mintTestActionToken(t, deps.ActionTokenSigningKey, deps.ActionTokenKeyID,
		agentID, apprID, "email.send", "1", hash, jti, time.Now().Add(-1*time.Minute))

	reqBody := fmt.Sprintf(`{"token":%q,"action_id":"email.send","parameters":{"to":"alice@example.com"}}`, token)
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteActionToken_ScopeMismatch(t *testing.T) {
	t.Parallel()
	_, deps, router, agentID, privKey, apprID, jti := setupExecuteTest(t)

	params := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, _ := HashParameters(params)

	// Token has scope "email.send" but request asks for "payment.charge".
	token := mintTestActionToken(t, deps.ActionTokenSigningKey, deps.ActionTokenKeyID,
		agentID, apprID, "email.send", "1", hash, jti, time.Now().Add(5*time.Minute))

	reqBody := fmt.Sprintf(`{"token":%q,"action_id":"payment.charge","parameters":{"to":"alice@example.com"}}`, token)
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != ErrInsufficientScope {
		t.Errorf("expected error code %q, got %q", ErrInsufficientScope, errResp.Error.Code)
	}
	if errResp.Error.Details["token_scope"] != "email.send" {
		t.Errorf("expected token_scope 'email.send', got %v", errResp.Error.Details["token_scope"])
	}
	if errResp.Error.Details["requested_action"] != "payment.charge" {
		t.Errorf("expected requested_action 'payment.charge', got %v", errResp.Error.Details["requested_action"])
	}
}

func TestExecuteActionToken_ParameterMismatch(t *testing.T) {
	t.Parallel()
	_, deps, router, agentID, privKey, apprID, jti := setupExecuteTest(t)

	// Compute hash for original parameters.
	originalParams := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, _ := HashParameters(originalParams)

	token := mintTestActionToken(t, deps.ActionTokenSigningKey, deps.ActionTokenKeyID,
		agentID, apprID, "email.send", "1", hash, jti, time.Now().Add(5*time.Minute))

	// Send with different parameters.
	reqBody := fmt.Sprintf(`{"token":%q,"action_id":"email.send","parameters":{"to":"evil@example.com"}}`, token)
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != ErrInvalidParameters {
		t.Errorf("expected error code %q, got %q", ErrInvalidParameters, errResp.Error.Code)
	}
}

func TestExecuteActionToken_WrongAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Create two agents.
	pubKeySSH1, privKey1, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key1: %v", err)
	}
	agentID1 := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH1)

	apprID := testhelper.GenerateID(t, "appr_")
	jti := testhelper.GenerateID(t, "tok_")
	testhelper.InsertApprovalWithJTI(t, tx, apprID, agentID1, uid, jti)

	deps := testDepsWithSigningKey(t, tx)
	router := NewRouter(deps)

	params := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, _ := HashParameters(params)

	// Mint token with subject agentID1+999, but sign the HTTP request as agentID1,
	// to verify that a subject/authenticated-agent mismatch is rejected.
	token := mintTestActionToken(t, deps.ActionTokenSigningKey, deps.ActionTokenKeyID,
		agentID1+999, apprID, "email.send", "1", hash, jti, time.Now().Add(5*time.Minute))

	reqBody := fmt.Sprintf(`{"token":%q,"action_id":"email.send","parameters":{"to":"alice@example.com"}}`, token)
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey1, agentID1)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteActionToken_MissingToken(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, _, _ := setupExecuteTest(t)

	reqBody := `{"action_id":"email.send","parameters":{"to":"alice@example.com"}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteActionToken_MissingActionID(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, _, _ := setupExecuteTest(t)

	reqBody := `{"token":"sometoken","parameters":{"to":"alice@example.com"}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteActionToken_JTIMismatchWithApproval(t *testing.T) {
	t.Parallel()
	_, deps, router, agentID, privKey, apprID, _ := setupExecuteTest(t)

	// Use a different JTI than what's stored in the approval.
	wrongJTI := testhelper.GenerateID(t, "tok_")
	params := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, _ := HashParameters(params)

	token := mintTestActionToken(t, deps.ActionTokenSigningKey, deps.ActionTokenKeyID,
		agentID, apprID, "email.send", "1", hash, wrongJTI, time.Now().Add(5*time.Minute))

	reqBody := fmt.Sprintf(`{"token":%q,"action_id":"email.send","parameters":{"to":"alice@example.com"}}`, token)
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != ErrInvalidToken {
		t.Errorf("expected error code %q, got %q", ErrInvalidToken, errResp.Error.Code)
	}
}

func TestExecuteActionToken_EmitsAuditEvent(t *testing.T) {
	t.Parallel()
	tx, deps, router, agentID, privKey, apprID, jti := setupExecuteTest(t)

	params := json.RawMessage(`{"to":"alice@example.com"}`)
	hash, _ := HashParameters(params)

	token := mintTestActionToken(t, deps.ActionTokenSigningKey, deps.ActionTokenKeyID,
		agentID, apprID, "email.send", "1", hash, jti, time.Now().Add(5*time.Minute))

	reqBody := fmt.Sprintf(`{"token":%q,"action_id":"email.send","parameters":{"to":"alice@example.com"}}`, token)
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Look up the approver ID to verify the audit event.
	var approverID string
	err := tx.QueryRow(context.Background(), `SELECT approver_id FROM approvals WHERE approval_id = $1`, apprID).Scan(&approverID)
	if err != nil {
		t.Fatalf("lookup approver_id: %v", err)
	}

	testhelper.RequireAuditEventCount(t, tx, approverID, "action.executed", 1)
}

// ── POST /actions/execute (standing approval path) ─────────────────────────

// setupStandingExecuteTest creates the test fixture for standing approval
// execution: user, agent, and active standing approval. If opts is non-nil,
// InsertStandingApprovalFull is used for full control; otherwise a simple
// active standing approval with the given action type is created.
func setupStandingExecuteTest(t *testing.T, actionType string, opts ...testhelper.StandingApprovalOpts) (tx db.DBTX, deps *Deps, router http.Handler, agentID int64, privKey ed25519.PrivateKey, saID, uid string) {
	t.Helper()
	txVal := testhelper.SetupTestDB(t)
	uidVal := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, txVal, uidVal, "u_"+uidVal[:8])

	pubKeySSH, pk, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	aid := testhelper.InsertAgentWithPublicKey(t, txVal, uidVal, "registered", pubKeySSH)

	saIDVal := testhelper.GenerateID(t, "sa_")
	if len(opts) > 0 {
		o := opts[0]
		if o.ActionType == "" {
			o.ActionType = actionType
		}
		testhelper.InsertStandingApprovalFull(t, txVal, saIDVal, aid, uidVal, o)
	} else {
		testhelper.InsertStandingApprovalWithActionType(t, txVal, saIDVal, aid, uidVal, actionType)
	}

	d := testDepsWithSigningKey(t, txVal)
	r := NewRouter(d)

	return txVal, d, r, aid, pk, saIDVal, uidVal
}

func TestExecuteActionStanding_Success(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, privKey, saID, _ := setupStandingExecuteTest(t, "email.read")

	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{"sender":"*@github.com"}}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp executeActionStandingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.StandingApprovalID != saID {
		t.Errorf("expected standing_approval_id %q, got %q", saID, resp.StandingApprovalID)
	}
	// No max_executions set, so executions_remaining should be nil (unlimited).
	if resp.ExecutionsRemaining != nil {
		t.Errorf("expected nil executions_remaining, got %v", *resp.ExecutionsRemaining)
	}

	// Verify execution_count was incremented.
	testhelper.RequireRowValue(t, tx, "standing_approvals", "standing_approval_id", saID, "execution_count", "1")
}

func TestExecuteActionStanding_NoMatchReturns404WithHint(t *testing.T) {
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

	// No standing approval exists for this agent/action type.
	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"payment.charge","version":"1","parameters":{"amount":100}}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != ErrNoMatchingStanding {
		t.Errorf("expected error code %q, got %q", ErrNoMatchingStanding, errResp.Error.Code)
	}
	hint, _ := errResp.Error.Details["hint"].(string)
	if hint == "" {
		t.Error("expected hint in error details")
	}
}

func TestExecuteActionStanding_MissingAction(t *testing.T) {
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

	// No token, no action field → standing approval path → 400 missing action.
	reqBody := `{"request_id":"abc123"}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteActionStanding_MissingRequestID(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, _, _ := setupStandingExecuteTest(t, "email.read")

	reqBody := `{"action":{"type":"email.read","version":"1","parameters":{}}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteActionStanding_EmitsAuditEvent(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, privKey, _, uid := setupStandingExecuteTest(t, "email.read")

	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{}}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify a standing_approval.executed audit event was emitted.
	testhelper.RequireAuditEventCount(t, tx, uid, "standing_approval.executed", 1)
}

func TestExecuteActionStanding_ConstraintViolation(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, _, _ := setupStandingExecuteTest(t, "email.read", testhelper.StandingApprovalOpts{
		Constraints: []byte(`{"sender":{"$pattern":"*@github.com"}}`),
	})

	// Parameters violate the constraint: sender is not @github.com.
	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{"sender":"evil@competitor.com"}}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != ErrConstraintViolation {
		t.Errorf("expected error code %q, got %q", ErrConstraintViolation, errResp.Error.Code)
	}
}

func TestExecuteActionStanding_ConstraintSatisfied(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, privKey, saID, _ := setupStandingExecuteTest(t, "email.read", testhelper.StandingApprovalOpts{
		Constraints: []byte(`{"sender":{"$pattern":"*@github.com"}}`),
	})

	// Parameters satisfy the constraint.
	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{"sender":"noreply@github.com"}}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify execution was recorded.
	testhelper.RequireRowValue(t, tx, "standing_approvals", "standing_approval_id", saID, "execution_count", "1")
}

func TestExecuteActionStanding_ExecutionsRemaining(t *testing.T) {
	t.Parallel()
	maxExec := 3
	_, _, router, agentID, privKey, _, _ := setupStandingExecuteTest(t, "email.read", testhelper.StandingApprovalOpts{
		MaxExecutions: &maxExec,
	})

	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{}}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp executeActionStandingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ExecutionsRemaining == nil {
		t.Fatal("expected executions_remaining to be non-nil")
	}
	if *resp.ExecutionsRemaining != 2 {
		t.Errorf("expected executions_remaining 2, got %d", *resp.ExecutionsRemaining)
	}
}

func TestExecuteActionStanding_ExpiredApproval(t *testing.T) {
	t.Parallel()
	// Create an expired standing approval (active status but time window has passed).
	_, _, router, agentID, privKey, _, _ := setupStandingExecuteTest(t, "email.read", testhelper.StandingApprovalOpts{
		StartsAt:  time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{}}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	// Should return 404 (no matching standing approval, since it's expired).
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != ErrNoMatchingStanding {
		t.Errorf("expected error code %q, got %q", ErrNoMatchingStanding, errResp.Error.Code)
	}
}

func TestExecuteActionStanding_DuplicateRequestID(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, _, _ := setupStandingExecuteTest(t, "email.read")

	reqBody := `{"request_id":"idempotent-req-001","action":{"type":"email.read","version":"1","parameters":{}}}`

	// First request should succeed.
	r1 := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}

	// Second request with same request_id should return 409 Conflict.
	r2 := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if errResp.Error.Code != ErrDuplicateRequestID {
		t.Errorf("expected error code %q, got %q", ErrDuplicateRequestID, errResp.Error.Code)
	}
}

func TestExecuteActionStanding_RevokedApproval(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, _, _ := setupStandingExecuteTest(t, "test.action", testhelper.StandingApprovalOpts{
		Status: "revoked",
	})

	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"test.action","version":"1","parameters":{}}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	// Revoked standing approval should not match.
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
