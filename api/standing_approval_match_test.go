package api

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// setupStandingApprovalTest creates the test fixture for standing approval
// auto-approval: user, agent, and active standing approval. Returns everything
// needed to submit a POST /approvals/request that should auto-approve.
func setupStandingApprovalTest(t *testing.T, actionType string, opts ...testhelper.StandingApprovalOpts) (tx db.DBTX, deps *Deps, router http.Handler, agentID int64, privKey ed25519.PrivateKey, saID, uid string) {
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

	d := testDepsForDB(t, txVal)
	r := NewRouter(d)

	return txVal, d, r, aid, pk, saIDVal, uidVal
}

// Alias for backward compat — used by metering_test.go and quota_test.go.
var setupStandingExecuteTest = setupStandingApprovalTest

// ── POST /approvals/request with standing approval auto-approve ─────────────

func TestRequestApproval_AutoApprove_Success(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, privKey, saID, _ := setupStandingApprovalTest(t, "email.read")

	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{"sender":"*@github.com"}},"context":{"description":"test"}}`
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
	if resp.Status != "approved" {
		t.Errorf("expected status \"approved\", got %q", resp.Status)
	}
	if resp.StandingApprovalID != saID {
		t.Errorf("expected standing_approval_id %q, got %q", saID, resp.StandingApprovalID)
	}
	// No approval_id should be set for auto-approved requests.
	if resp.ApprovalID != "" {
		t.Errorf("expected empty approval_id for auto-approval, got %q", resp.ApprovalID)
	}
	// No max_executions set, so executions_remaining should be nil (unlimited).
	if resp.ExecutionsRemaining != nil {
		t.Errorf("expected nil executions_remaining, got %v", *resp.ExecutionsRemaining)
	}

	// Verify execution_count was incremented.
	testhelper.RequireRowValue(t, tx, "standing_approvals", "standing_approval_id", saID, "execution_count", "1")
}

func TestRequestApproval_NoStandingApproval_CreatesPending(t *testing.T) {
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

	// No standing approval exists for this agent/action type.
	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"payment.charge","parameters":{"amount":100}},"context":{"description":"test"}}`
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
	if resp.Status != "pending" {
		t.Errorf("expected status \"pending\", got %q", resp.Status)
	}
	if resp.ApprovalID == "" {
		t.Error("expected non-empty approval_id for pending request")
	}
}

func TestRequestApproval_AutoApprove_ConstraintViolation_FallsThroughToPending(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, _, _ := setupStandingApprovalTest(t, "email.read", testhelper.StandingApprovalOpts{
		Constraints: []byte(`{"sender":{"$pattern":"*@github.com"}}`),
	})

	// Parameters violate the constraint: sender is not @github.com.
	// Should fall through to creating a pending approval.
	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{"sender":"evil@competitor.com"}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (pending), got %d: %s", w.Code, w.Body.String())
	}

	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "pending" {
		t.Errorf("expected status \"pending\" (fallthrough), got %q", resp.Status)
	}
}

func TestRequestApproval_AutoApprove_ConstraintSatisfied(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, privKey, saID, _ := setupStandingApprovalTest(t, "email.read", testhelper.StandingApprovalOpts{
		Constraints: []byte(`{"sender":{"$pattern":"*@github.com"}}`),
	})

	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{"sender":"noreply@github.com"}},"context":{"description":"test"}}`
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
	if resp.Status != "approved" {
		t.Errorf("expected status \"approved\", got %q", resp.Status)
	}

	// Verify execution was recorded.
	testhelper.RequireRowValue(t, tx, "standing_approvals", "standing_approval_id", saID, "execution_count", "1")
}

func TestRequestApproval_AutoApprove_ExecutionsRemaining(t *testing.T) {
	t.Parallel()
	maxExec := 3
	_, _, router, agentID, privKey, _, _ := setupStandingApprovalTest(t, "email.read", testhelper.StandingApprovalOpts{
		MaxExecutions: &maxExec,
	})

	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{}},"context":{"description":"test"}}`
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
	if resp.ExecutionsRemaining == nil {
		t.Fatal("expected executions_remaining to be non-nil")
	}
	if *resp.ExecutionsRemaining != 2 {
		t.Errorf("expected executions_remaining 2, got %d", *resp.ExecutionsRemaining)
	}
}

func TestRequestApproval_AutoApprove_ExpiredApproval_FallsThroughToPending(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, _, _ := setupStandingApprovalTest(t, "email.read", testhelper.StandingApprovalOpts{
		StartsAt:  time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	// Should fall through to pending (expired standing approval doesn't match).
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (pending), got %d: %s", w.Code, w.Body.String())
	}

	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "pending" {
		t.Errorf("expected status \"pending\" (expired SA), got %q", resp.Status)
	}
}

func TestRequestApproval_AutoApprove_DuplicateRequestID(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, _, _ := setupStandingApprovalTest(t, "email.read")

	reqBody := `{"request_id":"idempotent-req-001","action":{"type":"email.read","version":"1","parameters":{}},"context":{"description":"test"}}`

	// First request should succeed (auto-approve).
	r1 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}

	// Second request with same request_id should return 409 Conflict.
	r2 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestRequestApproval_AutoApprove_EmitsAuditEvent(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, privKey, _, uid := setupStandingApprovalTest(t, "email.read")

	reqBody := `{"request_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","action":{"type":"email.read","version":"1","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify a standing_approval.executed audit event was emitted.
	testhelper.RequireAuditEventCount(t, tx, uid, "standing_approval.executed", 1)
}

func TestRequestApproval_AutoApprove_WildcardActionTypeMatches(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, saID, _ := setupStandingApprovalTest(t, "any.action", testhelper.StandingApprovalOpts{
		ActionType: "*",
	})

	reqBody := `{"request_id":"wildcard-action-test-001","action":{"type":"any.action","version":"1","parameters":{}},"context":{"description":"test"}}`
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
	if resp.StandingApprovalID != saID {
		t.Errorf("expected standing_approval_id %q, got %q", saID, resp.StandingApprovalID)
	}
}

func TestRequestApproval_AutoApprove_SecondApprovalMatchesWhenFirstDoesNot(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Older approval: matches sender=*@github.com
	sa1ID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, sa1ID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "email.read",
		Constraints: []byte(`{"sender":{"$pattern":"*@github.com"}}`),
		StartsAt:    time.Now().Add(-2 * time.Hour),
	})

	// Newer approval: requires sender=*@competitor.com (won't match our request)
	sa2ID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, sa2ID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "email.read",
		Constraints: []byte(`{"sender":{"$pattern":"*@competitor.com"}}`),
		StartsAt:    time.Now().Add(-1 * time.Hour),
	})

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	reqBody := `{"request_id":"multi-sa-test-001","action":{"type":"email.read","version":"1","parameters":{"sender":"alice@github.com"}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (second approval should match), got %d: %s", w.Code, w.Body.String())
	}

	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "approved" {
		t.Errorf("expected status \"approved\", got %q", resp.Status)
	}

	// Assert based on execution counts rather than standing_approval_id, since
	// the query tiebreaker (standing_approval_id DESC) over random IDs makes
	// the iteration order non-deterministic. What matters: the github.com
	// constraint matched (execution_count=1) and competitor.com did not (0).
	testhelper.RequireRowValue(t, tx, "standing_approvals", "standing_approval_id", sa1ID, "execution_count", "1")
	testhelper.RequireRowValue(t, tx, "standing_approvals", "standing_approval_id", sa2ID, "execution_count", "0")
}

func TestRequestApproval_AutoApprove_ExhaustedBetweenFindAndRecord_FallsThroughToPending(t *testing.T) {
	t.Parallel()
	maxExec := 1
	tx, _, router, agentID, privKey, saID, _ := setupStandingApprovalTest(t, "email.read", testhelper.StandingApprovalOpts{
		MaxExecutions: &maxExec,
	})

	// Simulate race: the standing approval was found by FindActiveStandingApprovalsForAgent
	// (status=active, execution_count < max_executions), but between that query and
	// RecordStandingApprovalExecutionByAgent, another request exhausted the quota.
	// Set execution_count = max_executions to trigger StandingApprovalErrNotActive.
	testhelper.InsertStandingApprovalExecution(t, tx, saID)

	reqBody := `{"request_id":"exhausted-race-test-001","action":{"type":"email.read","version":"1","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// Should fall through to pending (exhausted SA triggers StandingApprovalErrNotActive).
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (pending), got %d: %s", w.Code, w.Body.String())
	}

	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "pending" {
		t.Errorf("expected status \"pending\" (exhausted SA race), got %q", resp.Status)
	}
	if resp.ApprovalID == "" {
		t.Error("expected non-empty approval_id for pending request")
	}
}

func TestRequestApproval_AutoApprove_RevokedApproval_FallsThroughToPending(t *testing.T) {
	t.Parallel()
	_, _, router, agentID, privKey, _, _ := setupStandingApprovalTest(t, "test.action", testhelper.StandingApprovalOpts{
		Status: "revoked",
	})

	reqBody := `{"request_id":"revoked-sa-test-001","action":{"type":"test.action","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// Revoked standing approval should not auto-approve — must fall through to pending.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (pending), got %d: %s", w.Code, w.Body.String())
	}

	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "pending" {
		t.Errorf("expected status \"pending\" (revoked SA), got %q", resp.Status)
	}
}
