package api

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

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

	// No action field → 400 missing action.
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
