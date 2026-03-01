package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// ── Approval request metering ───────────────────────────────────────────────

func TestMetering_ApprovalRequestIncrementsUsage(t *testing.T) {
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

	// No usage before any requests.
	testhelper.RequireUsageCount(t, tx, uid, 0)

	// Submit first approval request.
	reqBody := `{"request_id":"meter_req_001","action":{"type":"email.send","parameters":{"to":"alice@example.com"}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Usage should now be 1 with correct breakdown.
	usage := testhelper.RequireUsageCount(t, tx, uid, 1)
	agentKey := strconv.FormatInt(agentID, 10)
	testhelper.RequireUsageBreakdown(t, usage,
		map[string]int{agentKey: 1},     // by_agent
		nil,                             // by_connector (not checked)
		map[string]int{"email.send": 1}, // by_action_type
	)

	// Submit second approval request with different action type.
	reqBody2 := `{"request_id":"meter_req_002","action":{"type":"slack.send_message","parameters":{"channel":"#general"}},"context":{"description":"test2"}}`
	r2 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody2, privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("second request: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Usage should now be 2.
	testhelper.RequireUsageCount(t, tx, uid, 2)
}

// ── Standing approval execution metering (dashboard) ────────────────────────

func TestMetering_StandingApprovalExecutionIncrementsUsage(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "email.send")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// No usage before execution.
	testhelper.RequireUsageCount(t, tx, uid, 0)

	// Execute standing approval.
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{"parameters":{"sender":"*@github.com"}}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Usage should be 1 with action type tracked.
	usage := testhelper.RequireUsageCount(t, tx, uid, 1)
	testhelper.RequireUsageBreakdown(t, usage,
		nil,
		nil,
		map[string]int{"email.send": 1},
	)
}

// ── Standing approval execution metering (agent path) ───────────────────────

func TestMetering_AgentStandingExecutionIncrementsUsage(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, privKey, _, uid := setupStandingExecuteTest(t, "email.read")

	// No usage before execution.
	testhelper.RequireUsageCount(t, tx, uid, 0)

	reqBody := `{"request_id":"meter_sa_001","action":{"type":"email.read","version":"1","parameters":{}}}`
	r := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Usage should be 1 with agent and action type tracked.
	usage := testhelper.RequireUsageCount(t, tx, uid, 1)
	agentKey := strconv.FormatInt(agentID, 10)
	testhelper.RequireUsageBreakdown(t, usage,
		map[string]int{agentKey: 1},
		nil,
		map[string]int{"email.read": 1},
	)
}

// ── Token-based execution is NOT billable ───────────────────────────────────

func TestMetering_TokenExecutionDoesNotIncrementUsage(t *testing.T) {
	t.Parallel()
	tx, deps, router, agentID, privKey, apprID, jti := setupExecuteTest(t)

	// The approval request already happened in setupExecuteTest (via direct DB
	// insert), so no usage_period row should exist yet.
	userID := userIDFromApproval(t, tx, apprID)
	testhelper.RequireUsageCount(t, tx, userID, 0)

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

	// Token-based execution is NOT billable — usage should remain 0.
	testhelper.RequireUsageCount(t, tx, userID, 0)
}

// userIDFromApproval looks up the approver_id for a given approval.
func userIDFromApproval(t *testing.T, d db.DBTX, approvalID string) string {
	t.Helper()
	var uid string
	err := d.QueryRow(context.Background(),
		`SELECT approver_id FROM approvals WHERE approval_id = $1`, approvalID).Scan(&uid)
	if err != nil {
		t.Fatalf("lookup approver_id for approval %s: %v", approvalID, err)
	}
	return uid
}

// ── Dedup: duplicate request_id does NOT double-count ───────────────────────

func TestMetering_DuplicateApprovalRequestDoesNotDoubleCount(t *testing.T) {
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

	reqBody := `{"request_id":"dedup_001","action":{"type":"email.send","parameters":{"to":"alice@example.com"}},"context":{"description":"test"}}`

	// First request succeeds.
	r1 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	testhelper.RequireUsageCount(t, tx, uid, 1)

	// Second request with same request_id returns 409 Conflict.
	r2 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate request: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}

	// Usage should still be 1 — the duplicate was rejected before metering.
	testhelper.RequireUsageCount(t, tx, uid, 1)
}

func TestMetering_DuplicateStandingExecutionDoesNotDoubleCount(t *testing.T) {
	t.Parallel()
	// Use a pool (not a test transaction) because the standing approval
	// execution dedup uses a CTE — a unique violation inside a CTE aborts
	// the enclosing PostgreSQL transaction, making subsequent queries fail.
	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pu := testhelper.SetupPoolUser(t, "dedup", pubKeySSH)

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalWithActionType(t, pu.Pool, saID, pu.AgentID, pu.UserID, "email.read")

	deps := testDepsWithSigningKey(t, pu.Pool)
	router := NewRouter(deps)

	reqBody := `{"request_id":"dedup_sa_001","action":{"type":"email.read","version":"1","parameters":{}}}`

	// First execution succeeds.
	r1 := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, pu.AgentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first execution: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	testhelper.RequireUsageCount(t, pu.Pool, pu.UserID, 1)

	// Second execution with same request_id returns 409 Conflict.
	r2 := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody, privKey, pu.AgentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate execution: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}

	// Usage should still be 1 — the duplicate was rejected before metering.
	testhelper.RequireUsageCount(t, pu.Pool, pu.UserID, 1)
}

// ── Non-billable events should NOT meter ────────────────────────────────────

func TestMetering_ApproveDoesNotIncrementUsage(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, InviteHMACKey: "test-hmac-key"}
	router := NewRouter(deps)

	// Submit approval request (this is billable → count = 1).
	reqBody := `{"request_id":"approve_meter_001","action":{"type":"email.send","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("request: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	testhelper.RequireUsageCount(t, tx, uid, 1)

	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Approve the request (this is NOT billable).
	approveReq := authenticatedJSONRequest(t, http.MethodPost, "/approvals/"+resp.ApprovalID+"/approve", uid, `{}`)
	approveW := httptest.NewRecorder()
	router.ServeHTTP(approveW, approveReq)

	if approveW.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d: %s", approveW.Code, approveW.Body.String())
	}

	// Usage should still be 1 — approval resolution is not billable.
	testhelper.RequireUsageCount(t, tx, uid, 1)
}

// ── Multiple billable events accumulate correctly ───────────────────────────

func TestMetering_MultipleEventsAccumulate(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Also create a standing approval for the agent.
	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "email.read")

	deps := testDepsWithSigningKey(t, tx)
	router := NewRouter(deps)

	// 1. Submit approval request (billable).
	reqBody1 := `{"request_id":"multi_001","action":{"type":"email.send","parameters":{}},"context":{"description":"test"}}`
	r1 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody1, privKey, agentID)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("approval request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}

	// 2. Execute standing approval via agent path (billable).
	reqBody2 := `{"request_id":"multi_002","action":{"type":"email.read","version":"1","parameters":{}}}`
	r2 := signedJSONRequest(t, http.MethodPost, "/actions/execute", reqBody2, privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("standing execution: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// 3. Submit another approval request (billable).
	reqBody3 := `{"request_id":"multi_003","action":{"type":"slack.post","parameters":{}},"context":{"description":"test"}}`
	r3 := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody3, privKey, agentID)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, r3)
	if w3.Code != http.StatusOK {
		t.Fatalf("second approval request: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}

	// Total usage should be 3 with correct breakdown.
	usage := testhelper.RequireUsageCount(t, tx, uid, 3)
	testhelper.RequireUsageBreakdown(t, usage, nil, nil, map[string]int{
		"email.send": 1,
		"email.read": 1,
		"slack.post": 1,
	})
}

// ── Load test: metering correctness under volume ────────────────────────────

// TestMetering_HighVolumeAccuracy runs 50 approval requests serially and
// verifies all are metered correctly. Timing is logged but not asserted on
// (CI environments have unpredictable latency). Not run in parallel to avoid
// contention affecting timing measurements.
func TestMetering_HighVolumeAccuracy(t *testing.T) {
	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pu := testhelper.SetupPoolUser(t, "load", pubKeySSH)

	deps := &Deps{DB: pu.Pool, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	const iterations = 50
	start := time.Now()
	for i := 0; i < iterations; i++ {
		reqID := fmt.Sprintf("load_%d", i)
		reqBody := fmt.Sprintf(`{"request_id":%q,"action":{"type":"email.send","parameters":{}},"context":{"description":"load"}}`, reqID)
		r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, pu.AgentID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("iteration %d: expected 200, got %d: %s", i, w.Code, w.Body.String())
		}
	}
	elapsed := time.Since(start)

	t.Logf("metering latency: %d requests in %v (avg %v/req)", iterations, elapsed, elapsed/iterations)

	// Verify all requests were metered — this is the real assertion.
	testhelper.RequireUsageCount(t, pu.Pool, pu.UserID, iterations)
}

// ── Concurrent metering ─────────────────────────────────────────────────────

func TestMetering_ConcurrentApprovalsAccurateCount(t *testing.T) {
	t.Parallel()
	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pu := testhelper.SetupPoolUser(t, "conc", pubKeySSH)

	deps := &Deps{DB: pu.Pool, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	const goroutines = 10
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	// Build requests on the main goroutine to avoid calling testing.T
	// methods (via signedJSONRequest → t.Helper) from concurrent goroutines.
	reqs := make([]*http.Request, goroutines)
	for i := 0; i < goroutines; i++ {
		reqID := fmt.Sprintf("conc_%d", i)
		reqBody := fmt.Sprintf(`{"request_id":%q,"action":{"type":"email.send","parameters":{}},"context":{"description":"concurrent"}}`, reqID)
		reqs[i] = signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, pu.AgentID)
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, reqs[idx])
			if w.Code != http.StatusOK {
				errs <- fmt.Errorf("goroutine %d: expected 200, got %d: %s", idx, w.Code, w.Body.String())
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatal(err)
	}

	// All 10 concurrent approval requests should result in exactly 10 metered events.
	usage, err := db.GetCurrentPeriodUsage(context.Background(), pu.Pool, pu.UserID)
	if err != nil {
		t.Fatalf("GetCurrentPeriodUsage: %v", err)
	}
	if usage == nil {
		t.Fatal("expected usage row, got nil")
	}
	if usage.RequestCount != goroutines {
		t.Errorf("expected request_count=%d after concurrent requests, got %d", goroutines, usage.RequestCount)
	}
}
