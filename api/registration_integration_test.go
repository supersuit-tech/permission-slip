package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// ── Phase 1: End-to-End Integration Tests ───────────────────────────────────

// TestIntegration_HappyPath_InviteToVerify chains the full registration flow:
// create invite → agent registers via POST /invite/{code} → user sees confirmation
// code on dashboard → agent verifies via POST /agents/{id}/verify → agent is registered
// and can make authenticated (signed) requests.
func TestIntegration_HappyPath_InviteToVerify(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Step 1: Create an invite.
	inviteCode := testhelper.GenerateID(t, "PS-")
	codeHash := hashCodeHex(inviteCode, "")
	riID := testhelper.GenerateID(t, "ri_")
	if _, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, codeHash, 900); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	// Step 2: Agent registers via POST /invite/{code}.
	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	handler := InviteHandler(&Deps{DB: tx})
	body := fmt.Sprintf(`{"request_id":"req-happy-1","public_key":%q,"metadata":{"name":"test-agent"}}`, pubKeySSH)
	r := signedJSONRequest(t, http.MethodPost, "/invite/"+inviteCode, body, privKey, 0)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("invite register: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var regResp registerAgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &regResp); err != nil {
		t.Fatalf("unmarshal register response: %v", err)
	}
	if regResp.AgentID == 0 {
		t.Fatal("expected non-zero agent_id")
	}
	if !regResp.VerificationRequired {
		t.Error("expected verification_required=true")
	}
	if regResp.ExpiresAt == nil {
		t.Error("expected expires_at to be set")
	}

	// Step 3: User sees the confirmation code on the dashboard (GET /agents/{id}).
	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	getReq := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d", regResp.AgentID), uid)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get agent: expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var agentResp agentResponse
	if err := json.Unmarshal(getW.Body.Bytes(), &agentResp); err != nil {
		t.Fatalf("unmarshal agent response: %v", err)
	}
	if agentResp.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", agentResp.Status)
	}
	if agentResp.ConfirmationCode == nil {
		t.Fatal("expected confirmation_code to be set for pending agent")
	}
	confirmCode := *agentResp.ConfirmationCode

	// Step 4: Agent verifies via POST /agents/{id}/verify with the confirmation code.
	verifyPath := fmt.Sprintf("/agents/%d/verify", regResp.AgentID)
	verifyReq := signedJSONRequest(t, http.MethodPost, verifyPath,
		verifyRequestBody("verify-happy-1", confirmCode), privKey, regResp.AgentID)

	verifyW := httptest.NewRecorder()
	router.ServeHTTP(verifyW, verifyReq)

	if verifyW.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", verifyW.Code, verifyW.Body.String())
	}

	var verifyResp verifyRegistrationResponse
	if err := json.Unmarshal(verifyW.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("unmarshal verify response: %v", err)
	}
	if verifyResp.Status != "registered" {
		t.Errorf("expected status 'registered', got %q", verifyResp.Status)
	}
	if verifyResp.RegisteredAt == nil {
		t.Error("expected registered_at to be set")
	}

	// Step 5: Verify the agent is now registered in the database with correct state.
	agent, err := db.GetAgentByIDUnscoped(context.Background(), tx, regResp.AgentID)
	if err != nil {
		t.Fatalf("get agent from db: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent to exist in db")
	}
	if agent.Status != "registered" {
		t.Errorf("expected db status 'registered', got %q", agent.Status)
	}
	if agent.RegisteredAt == nil {
		t.Error("expected db registered_at to be set")
	}
	if agent.ConfirmationCode != nil {
		t.Error("expected confirmation_code to be cleared after registration")
	}
	if agent.ApproverID != uid {
		t.Errorf("expected approver_id %q, got %q", uid, agent.ApproverID)
	}

	// Step 6: Verify the registered agent can make an authenticated (signed) request.
	// Use the verify endpoint again — it should return 409 (already registered),
	// proving the signature was accepted and the agent was found.
	authReq := signedJSONRequest(t, http.MethodPost, verifyPath,
		verifyRequestBody("auth-check", "AAA-BBB"), privKey, regResp.AgentID)

	authW := httptest.NewRecorder()
	router.ServeHTTP(authW, authReq)

	// 409 Conflict proves: signature verified (not 401), agent found (not 404),
	// and already registered (not 200 or other). This confirms end-to-end auth.
	if authW.Code != http.StatusConflict {
		t.Fatalf("authenticated request: expected 409, got %d: %s", authW.Code, authW.Body.String())
	}
	var authErr ErrorResponse
	if err := json.Unmarshal(authW.Body.Bytes(), &authErr); err != nil {
		t.Fatalf("unmarshal auth error: %v", err)
	}
	if authErr.Error.Code != ErrAgentAlreadyRegistered {
		t.Errorf("expected error code %q, got %q", ErrAgentAlreadyRegistered, authErr.Error.Code)
	}
}

// TestIntegration_DashboardRegistration tests the full lifecycle with dashboard
// registration: create invite → agent registers (pending) → user registers
// agent via dashboard POST /agents/{id}/register (bypassing verification code)
// → agent is registered.
func TestIntegration_DashboardRegistration(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Step 1: Create an invite and register an agent (creates pending agent).
	reg := registerViaInvite(t, tx, uid)

	// Step 2: Verify the agent is pending with a confirmation code.
	agent, err := db.GetAgentByIDUnscoped(context.Background(), tx, reg.AgentID)
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if agent.Status != "pending" {
		t.Fatalf("expected status 'pending', got %q", agent.Status)
	}
	if agent.ConfirmationCode == nil {
		t.Fatal("expected confirmation code to be set for pending agent")
	}

	// Step 3: Dashboard user registers the agent (bypasses code verification).
	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})
	r := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/register", reg.AgentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("dashboard register: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != "registered" {
		t.Errorf("expected status 'registered', got %q", resp.Status)
	}
	if resp.RegisteredAt == nil {
		t.Error("expected registered_at to be set")
	}
	if resp.AgentID != reg.AgentID {
		t.Errorf("expected agent_id %d, got %d", reg.AgentID, resp.AgentID)
	}

	// Step 4: Verify the agent is fully registered in the database.
	dbAgent, err := db.GetAgentByIDUnscoped(context.Background(), tx, reg.AgentID)
	if err != nil {
		t.Fatalf("get agent from db: %v", err)
	}
	if dbAgent.Status != "registered" {
		t.Errorf("expected db status 'registered', got %q", dbAgent.Status)
	}
	if dbAgent.RegisteredAt == nil {
		t.Error("expected db registered_at to be set")
	}
}

// TestIntegration_BothPathsSameOutcome verifies that agents registered via
// /agents/{id}/verify and via dashboard /agents/{id}/register end up in an
// identical state: same status, registered_at set, confirmation_code cleared.
func TestIntegration_BothPathsSameOutcome(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	// Path A: Register via /verify (code verification).
	regA := registerViaInvite(t, tx, uid)
	verifyReq := signedJSONRequest(t, http.MethodPost,
		fmt.Sprintf("/agents/%d/verify", regA.AgentID),
		verifyRequestBody("verify-cmp", regA.ConfirmCode),
		regA.PrivKey, regA.AgentID)

	verifyW := httptest.NewRecorder()
	router.ServeHTTP(verifyW, verifyReq)
	if verifyW.Code != http.StatusOK {
		t.Fatalf("verify path: expected 200, got %d: %s", verifyW.Code, verifyW.Body.String())
	}

	// Path B: Register via dashboard /register.
	regB := registerViaInvite(t, tx, uid)
	dashReq := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/register", regB.AgentID), uid)
	dashW := httptest.NewRecorder()
	router.ServeHTTP(dashW, dashReq)
	if dashW.Code != http.StatusOK {
		t.Fatalf("dashboard path: expected 200, got %d: %s", dashW.Code, dashW.Body.String())
	}

	// Fetch both agents from the database and compare state.
	agentA, err := db.GetAgentByIDUnscoped(context.Background(), tx, regA.AgentID)
	if err != nil {
		t.Fatalf("get agent A: %v", err)
	}
	agentB, err := db.GetAgentByIDUnscoped(context.Background(), tx, regB.AgentID)
	if err != nil {
		t.Fatalf("get agent B: %v", err)
	}

	// Both should be registered.
	if agentA.Status != "registered" {
		t.Errorf("agent A: expected status 'registered', got %q", agentA.Status)
	}
	if agentB.Status != "registered" {
		t.Errorf("agent B: expected status 'registered', got %q", agentB.Status)
	}

	// Both should have registered_at set.
	if agentA.RegisteredAt == nil {
		t.Error("agent A: expected registered_at to be set")
	}
	if agentB.RegisteredAt == nil {
		t.Error("agent B: expected registered_at to be set")
	}

	// Both should have confirmation_code cleared.
	if agentA.ConfirmationCode != nil {
		t.Errorf("agent A: expected confirmation_code to be nil, got %q", *agentA.ConfirmationCode)
	}
	if agentB.ConfirmationCode != nil {
		t.Errorf("agent B: expected confirmation_code to be nil, got %q", *agentB.ConfirmationCode)
	}

	// Both should have the same approver.
	if agentA.ApproverID != agentB.ApproverID {
		t.Errorf("approver_id mismatch: A=%q, B=%q", agentA.ApproverID, agentB.ApproverID)
	}

	// Neither should be deactivated.
	if agentA.DeactivatedAt != nil {
		t.Error("agent A: expected deactivated_at to be nil")
	}
	if agentB.DeactivatedAt != nil {
		t.Error("agent B: expected deactivated_at to be nil")
	}

	// Both registration paths should treat expires_at consistently after
	// registration. This test enforces that they match (both nil or both non-nil).
	if (agentA.ExpiresAt == nil) != (agentB.ExpiresAt == nil) {
		t.Errorf("expires_at mismatch: A=%v, B=%v", agentA.ExpiresAt, agentB.ExpiresAt)
	}

	// Both should have public keys set (Ed25519 keys generated during registration).
	if agentA.PublicKey == "" {
		t.Error("agent A: expected public_key to be set")
	}
	if agentB.PublicKey == "" {
		t.Error("agent B: expected public_key to be set")
	}

	// Dashboard path should have 0 verification_attempts (skips code verification).
	// Verify path may have >0 (the successful verify counts as an attempt).
	if agentB.VerificationAttempts != 0 {
		t.Errorf("agent B (dashboard): expected 0 verification_attempts, got %d", agentB.VerificationAttempts)
	}
}

// ── Phase 1: Concurrency Tests ──────────────────────────────────────────────

// TestIntegration_ConcurrentInviteConsumption races two goroutines to consume
// the same invite code simultaneously. Exactly one should succeed (200), the
// other should get a conflict or not-found response.
func TestIntegration_ConcurrentInviteConsumption(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupPool(t)

	// Create test data using the pool directly.
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, pool, uid, "u_"+uid[:8])

	inviteCode := testhelper.GenerateID(t, "PS-")
	codeHash := hashCodeHex(inviteCode, "")
	riID := testhelper.GenerateID(t, "ri_")
	if _, err := db.CreateRegistrationInvite(context.Background(), pool, riID, uid, codeHash, 900); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	handler := InviteHandler(&Deps{DB: pool})
	const goroutines = 5

	// Build requests on the main goroutine to avoid calling t.Helper() from
	// concurrent goroutines (which causes a data race on testing.T internals).
	reqs := make([]*http.Request, goroutines)
	for i := range goroutines {
		pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
		if err != nil {
			t.Fatalf("generate key %d: %v", i, err)
		}
		body := inviteRequestBody(t, fmt.Sprintf("race-%d", i), pubKeySSH)
		reqs[i] = signedJSONRequest(t, http.MethodPost, "/invite/"+inviteCode, body, privKey, 0)
	}

	results := make([]int, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, reqs[idx])
			results[idx] = w.Code
		}(i)
	}

	wg.Wait()
	requireExactlyOneSuccess(t, results, http.StatusConflict)
}

// TestIntegration_ConcurrentVerification races multiple goroutines to verify
// the same pending agent with the correct confirmation code. Exactly one should
// succeed (200), the others should get a conflict (409).
func TestIntegration_ConcurrentVerification(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupPool(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, pool, uid, "u_"+uid[:8])

	// Register an agent via invite to get a pending agent with a confirmation code.
	reg := registerViaInvite(t, pool, uid)

	router := NewRouter(&Deps{DB: pool, SupabaseJWTSecret: testJWTSecret})
	const goroutines = 5

	// Build requests on the main goroutine to avoid calling t.Helper() from
	// concurrent goroutines (which causes a data race on testing.T internals).
	reqs := make([]*http.Request, goroutines)
	for i := range goroutines {
		path := fmt.Sprintf("/agents/%d/verify", reg.AgentID)
		body := verifyRequestBody(fmt.Sprintf("race-verify-%d", i), reg.ConfirmCode)
		reqs[i] = signedJSONRequest(t, http.MethodPost, path, body, reg.PrivKey, reg.AgentID)
	}

	results := make([]int, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, reqs[idx])
			results[idx] = w.Code
		}(i)
	}

	wg.Wait()
	requireExactlyOneSuccess(t, results, http.StatusConflict)

	// Verify the agent is registered in the database.
	agent, err := db.GetAgentByIDUnscoped(context.Background(), pool, reg.AgentID)
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if agent.Status != "registered" {
		t.Errorf("expected status 'registered', got %q", agent.Status)
	}
}

// TestIntegration_ConcurrentDashboardRegistration races two simultaneous
// POST /agents/{id}/register requests for the same pending agent. Exactly one
// should succeed (200), the other should get a conflict (409).
func TestIntegration_ConcurrentDashboardRegistration(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupPool(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, pool, uid, "u_"+uid[:8])

	// Create a pending agent via invite registration.
	reg := registerViaInvite(t, pool, uid)

	router := NewRouter(&Deps{DB: pool, SupabaseJWTSecret: testJWTSecret})
	const goroutines = 2

	// Build requests on the main goroutine to avoid calling t.Helper() from
	// concurrent goroutines (which causes a data race on testing.T internals).
	reqs := make([]*http.Request, goroutines)
	for i := range goroutines {
		reqs[i] = authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/register", reg.AgentID), uid)
	}

	results := make([]int, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, reqs[idx])
			results[idx] = w.Code
		}(i)
	}

	wg.Wait()
	requireExactlyOneSuccess(t, results, http.StatusConflict)

	// Verify the agent is registered in the database.
	dashAgent, err := db.GetAgentByIDUnscoped(context.Background(), pool, reg.AgentID)
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if dashAgent.Status != "registered" {
		t.Errorf("expected status 'registered', got %q", dashAgent.Status)
	}
}

// TestIntegration_DeactivationLifecycle tests the full lifecycle: create via
// invite → verify → deactivate → verify terminal state → confirm
// re-registration is rejected.
func TestIntegration_DeactivationLifecycle(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	// Step 1: Register an agent via invite and verify it.
	reg := registerViaInvite(t, tx, uid)
	verifyReq := signedJSONRequest(t, http.MethodPost,
		fmt.Sprintf("/agents/%d/verify", reg.AgentID),
		verifyRequestBody("deact-verify", reg.ConfirmCode),
		reg.PrivKey, reg.AgentID)

	verifyW := httptest.NewRecorder()
	router.ServeHTTP(verifyW, verifyReq)
	if verifyW.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", verifyW.Code, verifyW.Body.String())
	}

	// Step 2: Deactivate the agent.
	deactReq := authenticatedRequest(t, http.MethodPost,
		fmt.Sprintf("/agents/%d/deactivate", reg.AgentID), uid)
	deactW := httptest.NewRecorder()
	router.ServeHTTP(deactW, deactReq)

	if deactW.Code != http.StatusOK {
		t.Fatalf("deactivate: expected 200, got %d: %s", deactW.Code, deactW.Body.String())
	}

	var deactResp agentResponse
	if err := json.Unmarshal(deactW.Body.Bytes(), &deactResp); err != nil {
		t.Fatalf("unmarshal deactivate response: %v", err)
	}
	if deactResp.Status != "deactivated" {
		t.Errorf("expected status 'deactivated', got %q", deactResp.Status)
	}
	if deactResp.DeactivatedAt == nil {
		t.Error("expected deactivated_at to be set")
	}

	// Step 3: Verify terminal state in the database.
	dbAgent, err := db.GetAgentByIDUnscoped(context.Background(), tx, reg.AgentID)
	if err != nil {
		t.Fatalf("get agent from db: %v", err)
	}
	if dbAgent.Status != "deactivated" {
		t.Errorf("expected db status 'deactivated', got %q", dbAgent.Status)
	}
	if dbAgent.DeactivatedAt == nil {
		t.Error("expected db deactivated_at to be set")
	}

	// Step 4: Attempt re-registration via dashboard — should be rejected.
	reRegReq := authenticatedRequest(t, http.MethodPost,
		fmt.Sprintf("/agents/%d/register", reg.AgentID), uid)
	reRegW := httptest.NewRecorder()
	router.ServeHTTP(reRegW, reRegReq)

	if reRegW.Code != http.StatusConflict {
		t.Fatalf("re-register deactivated agent: expected 409, got %d: %s",
			reRegW.Code, reRegW.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(reRegW.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal re-register error: %v", err)
	}
	if errResp.Error.Code != ErrAgentAlreadyRegistered {
		t.Errorf("expected error code %q, got %q", ErrAgentAlreadyRegistered, errResp.Error.Code)
	}

	// Step 5: Deactivating again should return 404 (already deactivated).
	deact2Req := authenticatedRequest(t, http.MethodPost,
		fmt.Sprintf("/agents/%d/deactivate", reg.AgentID), uid)
	deact2W := httptest.NewRecorder()
	router.ServeHTTP(deact2W, deact2Req)

	if deact2W.Code != http.StatusNotFound {
		t.Fatalf("double deactivate: expected 404, got %d: %s",
			deact2W.Code, deact2W.Body.String())
	}
}
