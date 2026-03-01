package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
	"github.com/supersuit-tech/permission-slip-web/vault"
)

// ── Standing Approval Limit Tests ───────────────────────────────────────────

func TestCreateStandingApproval_FreePlan_AtLimit_Returns403(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Insert standing approvals up to the limit (5 for free tier).
	for i := 0; i < 5; i++ {
		testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{
		"agent_id": %d,
		"action_type": "test.action",
		"expires_at": "%s"
	}`, agentID, time.Now().Add(24*time.Hour).Format(time.RFC3339))

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error.Code != ErrStandingApprovalLimitReached {
		t.Errorf("expected error code %q, got %q", ErrStandingApprovalLimitReached, resp.Error.Code)
	}
	if resp.Error.Details == nil {
		t.Fatal("expected details in error response")
	}
	if count, ok := resp.Error.Details["current_count"].(float64); !ok || int(count) != 5 {
		t.Errorf("expected current_count=5, got %v", resp.Error.Details["current_count"])
	}
	if limit, ok := resp.Error.Details["limit"].(float64); !ok || int(limit) != 5 {
		t.Errorf("expected limit=5, got %v", resp.Error.Details["limit"])
	}
}

func TestCreateStandingApproval_FreePlan_UnderLimit_Succeeds(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	// Register the agent so standing approvals can reference it.
	testhelper.MustExec(t, tx, `UPDATE agents SET status = 'registered', registered_at = now() WHERE agent_id = $1`, agentID)
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Insert 4 standing approvals (under the limit of 5).
	for i := 0; i < 4; i++ {
		testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{
		"agent_id": %d,
		"action_type": "test.action",
		"expires_at": "%s"
	}`, agentID, time.Now().Add(24*time.Hour).Format(time.RFC3339))

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_PaidPlan_NoLimit(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.MustExec(t, tx, `UPDATE agents SET status = 'registered', registered_at = now() WHERE agent_id = $1`, agentID)
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Insert 10 standing approvals (would exceed free limit of 5).
	for i := 0; i < 10; i++ {
		testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{
		"agent_id": %d,
		"action_type": "test.action",
		"expires_at": "%s"
	}`, agentID, time.Now().Add(24*time.Hour).Format(time.RFC3339))

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (paid plan, unlimited), got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_NoSubscription_NoLimit(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.MustExec(t, tx, `UPDATE agents SET status = 'registered', registered_at = now() WHERE agent_id = $1`, agentID)
	// Intentionally no subscription — should bypass limits.

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{
		"agent_id": %d,
		"action_type": "test.action",
		"expires_at": "%s"
	}`, agentID, time.Now().Add(24*time.Hour).Format(time.RFC3339))

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (no subscription, no enforcement), got %d: %s", w.Code, w.Body.String())
	}
}

// ── Credential Limit Tests ──────────────────────────────────────────────────

func TestStoreCredential_FreePlan_AtLimit_Returns403(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Insert 5 credentials (limit for free tier).
	for i := 0; i < 5; i++ {
		testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid, fmt.Sprintf("service%d", i))
	}

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "newservice", "credentials": {"api_key": "test123"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error.Code != ErrCredentialLimitReached {
		t.Errorf("expected error code %q, got %q", ErrCredentialLimitReached, resp.Error.Code)
	}
	if resp.Error.Details == nil {
		t.Fatal("expected details in error response")
	}
	if count, ok := resp.Error.Details["current_count"].(float64); !ok || int(count) != 5 {
		t.Errorf("expected current_count=5, got %v", resp.Error.Details["current_count"])
	}
}

func TestStoreCredential_FreePlan_UnderLimit_Succeeds(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Insert 4 credentials (under limit of 5).
	for i := 0; i < 4; i++ {
		testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid, fmt.Sprintf("service%d", i))
	}

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "newservice", "credentials": {"api_key": "test123"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStoreCredential_PaidPlan_NoLimit(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Insert 10 credentials (would exceed free limit).
	for i := 0; i < 10; i++ {
		testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid, fmt.Sprintf("service%d", i))
	}

	mockVault := vault.NewMockVaultStore()
	deps := &Deps{DB: tx, Vault: mockVault, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"service": "newservice", "credentials": {"api_key": "test123"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/credentials", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (paid plan, unlimited), got %d: %s", w.Code, w.Body.String())
	}
}

// ── Agent Limit Tests (via registration) ────────────────────────────────────
// Agent registration goes through POST /invite/{invite_code} which requires
// invite code crypto. We test the limit check helper directly instead.

func TestCheckAgentLimit_FreePlan_AtLimit_ReturnsForbidden(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Insert 3 registered agents (limit for free tier).
	for i := 0; i < 3; i++ {
		testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	blocked := checkAgentLimit(r.Context(), w, r, tx, uid)

	if !blocked {
		t.Fatal("expected checkAgentLimit to block (return true)")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error.Code != ErrAgentLimitReached {
		t.Errorf("expected error code %q, got %q", ErrAgentLimitReached, resp.Error.Code)
	}
	if resp.Error.Details == nil {
		t.Fatal("expected details in error response")
	}
	if count, ok := resp.Error.Details["current_count"].(float64); !ok || int(count) != 3 {
		t.Errorf("expected current_count=3, got %v", resp.Error.Details["current_count"])
	}
	if limit, ok := resp.Error.Details["limit"].(float64); !ok || int(limit) != 3 {
		t.Errorf("expected limit=3, got %v", resp.Error.Details["limit"])
	}
	if planID, ok := resp.Error.Details["plan_id"].(string); !ok || planID != db.PlanFree {
		t.Errorf("expected plan_id=%q, got %v", db.PlanFree, resp.Error.Details["plan_id"])
	}
}

func TestCheckAgentLimit_FreePlan_UnderLimit_Allows(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Insert 2 registered agents (under limit of 3).
	for i := 0; i < 2; i++ {
		testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	blocked := checkAgentLimit(r.Context(), w, r, tx, uid)

	if blocked {
		t.Fatalf("expected checkAgentLimit to allow (return false), but it blocked: %s", w.Body.String())
	}
}

func TestCheckAgentLimit_PaidPlan_Unlimited(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Insert 10 registered agents (would exceed free limit of 3).
	for i := 0; i < 10; i++ {
		testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	blocked := checkAgentLimit(r.Context(), w, r, tx, uid)

	if blocked {
		t.Fatalf("expected paid plan to bypass agent limit, but it blocked: %s", w.Body.String())
	}
}

func TestCheckAgentLimit_NoSubscription_Allows(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	// No subscription — should not block.

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	blocked := checkAgentLimit(r.Context(), w, r, tx, uid)

	if blocked {
		t.Fatalf("expected no subscription to bypass limits, but it blocked: %s", w.Body.String())
	}
}

func TestCheckAgentLimit_PendingAgentsCountTowardLimit(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// 2 registered + 1 pending (not expired) = 3, at the limit.
	testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	testhelper.InsertAgent(t, tx, uid)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	blocked := checkAgentLimit(r.Context(), w, r, tx, uid)

	if !blocked {
		t.Fatal("expected pending agents to count toward limit, but was allowed")
	}
}

// ── Invite Creation Limit Tests ──────────────────────────────────────────────
// Invite creation (POST /registration-invites) proactively checks the agent
// limit so users learn early, before sharing an invite that would fail.

func TestCreateInvite_FreePlan_AtAgentLimit_Returns403(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Insert 3 registered agents (limit for free tier).
	for i := 0; i < 3; i++ {
		testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error.Code != ErrAgentLimitReached {
		t.Errorf("expected error code %q, got %q", ErrAgentLimitReached, resp.Error.Code)
	}
}

func TestCreateInvite_PaidPlan_Succeeds(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// 10 registered agents — paid plan has no limit.
	for i := 0; i < 10; i++ {
		testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, InviteHMACKey: "testkey"}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (paid plan), got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_RevokedDoNotCountTowardLimit(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.MustExec(t, tx, `UPDATE agents SET status = 'registered', registered_at = now() WHERE agent_id = $1`, agentID)
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// 4 active + 5 revoked = only 4 count toward limit (under 5).
	// If revoked counted, total would be 9 and creation would be blocked.
	for i := 0; i < 4; i++ {
		testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	}
	for i := 0; i < 5; i++ {
		testhelper.InsertStandingApprovalWithStatus(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, "revoked")
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{
		"agent_id": %d,
		"action_type": "test.action",
		"expires_at": "%s"
	}`, agentID, time.Now().Add(24*time.Hour).Format(time.RFC3339))

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// 4 active (under limit of 5) — should succeed because revoked don't count.
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (revoked don't count, 4 active under limit of 5), got %d: %s", w.Code, w.Body.String())
	}
}
