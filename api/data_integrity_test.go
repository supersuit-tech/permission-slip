package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// ── Phase 4: Data Integrity & Contracts ─────────────────────────────────────

// auditEventRow holds the columns we need to validate in audit event tests.
type auditEventRow struct {
	EventType  string
	Outcome    string
	SourceID   string
	SourceType string
	AgentMeta  []byte
	AgentID    int64
}

// queryAuditEvents returns all audit events matching the given user and event type,
// ordered by creation time (oldest first).
func queryAuditEvents(t *testing.T, d db.DBTX, userID, eventType string) []auditEventRow {
	t.Helper()
	rows, err := d.Query(context.Background(),
		`SELECT event_type, outcome, source_id, source_type, agent_meta, agent_id
		 FROM audit_events
		 WHERE user_id = $1 AND event_type = $2
		 ORDER BY created_at ASC, id ASC`,
		userID, eventType,
	)
	if err != nil {
		t.Fatalf("queryAuditEvents: %v", err)
	}
	defer rows.Close()

	var events []auditEventRow
	for rows.Next() {
		var e auditEventRow
		if err := rows.Scan(&e.EventType, &e.Outcome, &e.SourceID, &e.SourceType, &e.AgentMeta, &e.AgentID); err != nil {
			t.Fatalf("scan audit event: %v", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	return events
}

// setupPoolUser creates a user directly in the pool and registers t.Cleanup
// to delete all child rows in FK-safe order (leaf tables first). This prevents
// the silent-failure bug where deleting agents before audit_events violates
// ON DELETE RESTRICT and leaves orphaned rows.
func setupPoolUser(t *testing.T, d db.DBTX) string {
	t.Helper()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, d, uid, "u_"+uid[:8])
	t.Cleanup(func() {
		ctx := context.Background()
		// FK-safe order: leaf tables first, root tables last.
		d.Exec(ctx, `DELETE FROM audit_events WHERE user_id = $1`, uid)
		d.Exec(ctx, `DELETE FROM agents WHERE approver_id = $1`, uid)
		d.Exec(ctx, `DELETE FROM registration_invites WHERE user_id = $1`, uid)
		d.Exec(ctx, `DELETE FROM profiles WHERE id = $1`, uid)
		d.Exec(ctx, `DELETE FROM auth.users WHERE id = $1`, uid)
	})
	return uid
}

// requireAuditEvent asserts that an audit event row matches the expected fields.
func requireAuditEvent(t *testing.T, got auditEventRow, wantOutcome, wantSourceID, wantSourceType string, wantAgentID int64) {
	t.Helper()
	if got.EventType != "agent.registered" {
		t.Errorf("expected event_type 'agent.registered', got %q", got.EventType)
	}
	if got.Outcome != wantOutcome {
		t.Errorf("expected outcome %q, got %q", wantOutcome, got.Outcome)
	}
	if got.SourceID != wantSourceID {
		t.Errorf("expected source_id %q, got %q", wantSourceID, got.SourceID)
	}
	if got.SourceType != wantSourceType {
		t.Errorf("expected source_type %q, got %q", wantSourceType, got.SourceType)
	}
	if got.AgentID != wantAgentID {
		t.Errorf("expected agent_id %d, got %d", wantAgentID, got.AgentID)
	}
}

// ── Test 1: Audit event structure validation ─────────────────────────────────

func TestAuditEventStructure_InviteRegistration(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Register via invite to produce a "pending" audit event.
	reg := registerViaInvite(t, tx, uid)

	events := queryAuditEvents(t, tx, uid, "agent.registered")
	if len(events) != 1 {
		t.Fatalf("expected 1 audit event after invite registration, got %d", len(events))
	}

	pending := events[0]
	if pending.EventType != "agent.registered" {
		t.Errorf("expected event_type 'agent.registered', got %q", pending.EventType)
	}
	if pending.Outcome != "pending" {
		t.Errorf("expected outcome 'pending', got %q", pending.Outcome)
	}
	if !strings.HasPrefix(pending.SourceID, "ri:") {
		t.Errorf("expected source_id to start with 'ri:', got %q", pending.SourceID)
	}
	if pending.SourceType != "registration_invite" {
		t.Errorf("expected source_type 'registration_invite', got %q", pending.SourceType)
	}
	if pending.AgentID != reg.AgentID {
		t.Errorf("expected agent_id %d, got %d", reg.AgentID, pending.AgentID)
	}

	// Now verify the agent to produce a "registered" audit event.
	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	body := fmt.Sprintf(`{"request_id":"verify-audit-struct","confirmation_code":%q}`, reg.ConfirmCode)
	bodyBytes := []byte(body)
	path := fmt.Sprintf("/agents/%d/verify", reg.AgentID)
	r := httptest.NewRequest(http.MethodPost, path, io.NopCloser(strings.NewReader(body)))
	r.Header.Set("Content-Type", "application/json")
	SignRequest(reg.PrivKey, reg.AgentID, r, bodyBytes)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	events = queryAuditEvents(t, tx, uid, "agent.registered")
	if len(events) != 2 {
		t.Fatalf("expected 2 audit events after verification, got %d", len(events))
	}

	requireAuditEvent(t, events[1], "registered", fmt.Sprintf("ar:%d", reg.AgentID), "agent", reg.AgentID)
}

func TestAuditEventStructure_DashboardRegistration(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/register", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	events := queryAuditEvents(t, tx, uid, "agent.registered")
	if len(events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(events))
	}

	requireAuditEvent(t, events[0], "registered", fmt.Sprintf("ar:%d", agentID), "agent", agentID)
}

func TestAuditEventStructure_InviteWithMetadata(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Create invite + register with metadata.
	inviteCode := testhelper.GenerateID(t, "PS-")
	codeHash := hashCodeHex(inviteCode, "")
	riID := testhelper.GenerateID(t, "ri_")
	if _, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, codeHash, 900); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	handler := InviteHandler(&Deps{DB: tx})

	metadata := `{"name":"test-agent","env":"staging"}`
	body := fmt.Sprintf(`{"request_id":"req-meta","public_key":%q,"metadata":%s}`, pubKeySSH, metadata)
	bodyBytes := []byte(body)
	r := httptest.NewRequest(http.MethodPost, "/invite/"+inviteCode, io.NopCloser(strings.NewReader(body)))
	r.Header.Set("Content-Type", "application/json")
	SignRequest(privKey, 0, r, bodyBytes)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	events := queryAuditEvents(t, tx, uid, "agent.registered")
	if len(events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(events))
	}

	ev := events[0]
	if ev.AgentMeta == nil {
		t.Fatal("expected non-nil agent_meta")
	}

	var meta map[string]any
	if err := json.Unmarshal(ev.AgentMeta, &meta); err != nil {
		t.Fatalf("unmarshal agent_meta: %v", err)
	}
	if meta["name"] != "test-agent" {
		t.Errorf("expected agent_meta.name='test-agent', got %v", meta["name"])
	}
	if meta["env"] != "staging" {
		t.Errorf("expected agent_meta.env='staging', got %v", meta["env"])
	}
}

// ── Test 2: Response field validation by agent status ────────────────────────

func TestResponseFields_PendingAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Set a confirmation code on the pending agent for the test.
	testhelper.MustExec(t, tx,
		`UPDATE agents SET confirmation_code = 'AB3-CD4' WHERE agent_id = $1`, agentID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp["status"] != "pending" {
		t.Errorf("expected status 'pending', got %v", resp["status"])
	}
	if resp["confirmation_code"] == nil {
		t.Error("expected confirmation_code to be present for pending agent")
	}
	if resp["confirmation_code"] != "AB3-CD4" {
		t.Errorf("expected confirmation_code 'AB3-CD4', got %v", resp["confirmation_code"])
	}
	if resp["registered_at"] != nil {
		t.Error("expected registered_at to be null for pending agent")
	}
	if resp["deactivated_at"] != nil {
		t.Error("expected deactivated_at to be null for pending agent")
	}
}

func TestResponseFields_RegisteredAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	// Set registered_at for the registered agent.
	testhelper.MustExec(t, tx,
		`UPDATE agents SET registered_at = now() WHERE agent_id = $1`, agentID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp["status"] != "registered" {
		t.Errorf("expected status 'registered', got %v", resp["status"])
	}
	// confirmation_code should NOT be present for registered agents.
	if resp["confirmation_code"] != nil {
		t.Errorf("expected confirmation_code to be absent for registered agent, got %v", resp["confirmation_code"])
	}
	if resp["registered_at"] == nil {
		t.Error("expected registered_at to be set for registered agent")
	}
	if resp["deactivated_at"] != nil {
		t.Error("expected deactivated_at to be null for active registered agent")
	}
}

func TestResponseFields_DeactivatedAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "deactivated")

	// Set deactivated_at for the deactivated agent.
	testhelper.MustExec(t, tx,
		`UPDATE agents SET deactivated_at = now() WHERE agent_id = $1`, agentID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp["status"] != "deactivated" {
		t.Errorf("expected status 'deactivated', got %v", resp["status"])
	}
	// confirmation_code should NOT be present for deactivated agents.
	if resp["confirmation_code"] != nil {
		t.Errorf("expected confirmation_code to be absent for deactivated agent, got %v", resp["confirmation_code"])
	}
	if resp["deactivated_at"] == nil {
		t.Error("expected deactivated_at to be set for deactivated agent")
	}
}

// ── Test 3: Confirmation code uniqueness ─────────────────────────────────────

func TestConfirmationCodeUniqueness_ConcurrentRegistrations(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupPool(t)
	uid := setupPoolUser(t, pool)

	const concurrency = 20

	// Create invites for each concurrent registration.
	inviteCodes := make([]string, concurrency)
	for i := 0; i < concurrency; i++ {
		inviteCode := testhelper.GenerateID(t, "PS-")
		codeHash := hashCodeHex(inviteCode, "")
		riID := testhelper.GenerateID(t, "ri_")
		if _, err := db.CreateRegistrationInvite(context.Background(), pool, riID, uid, codeHash, 900); err != nil {
			t.Fatalf("create invite %d: %v", i, err)
		}
		inviteCodes[i] = inviteCode
	}

	// Register all agents concurrently, collecting their confirmation codes.
	codes := make([]string, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("generate key %d: %w", idx, err))
				mu.Unlock()
				return
			}

			handler := InviteHandler(&Deps{DB: pool})

			body := fmt.Sprintf(`{"request_id":"req-uniq-%d","public_key":%q}`, idx, pubKeySSH)
			bodyBytes := []byte(body)
			r := httptest.NewRequest(http.MethodPost, "/invite/"+inviteCodes[idx], io.NopCloser(strings.NewReader(body)))
			r.Header.Set("Content-Type", "application/json")
			SignRequest(privKey, 0, r, bodyBytes)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			if w.Code != http.StatusOK {
				mu.Lock()
				errs = append(errs, fmt.Errorf("registration %d: expected 200, got %d: %s", idx, w.Code, w.Body.String()))
				mu.Unlock()
				return
			}

			var resp registerAgentResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("unmarshal %d: %w", idx, err))
				mu.Unlock()
				return
			}

			// Fetch the confirmation code from the DB.
			agent, err := db.GetAgentByIDUnscoped(context.Background(), pool, resp.AgentID)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("get agent %d: %w", idx, err))
				mu.Unlock()
				return
			}
			if agent == nil || agent.ConfirmationCode == nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("agent %d: nil confirmation code", idx))
				mu.Unlock()
				return
			}

			mu.Lock()
			codes[idx] = *agent.ConfirmationCode
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	for _, err := range errs {
		t.Error(err)
	}

	// Verify all codes are unique.
	seen := make(map[string]int)
	var skipped int
	for i, code := range codes {
		if code == "" {
			skipped++
			continue // skip failures
		}
		if prev, exists := seen[code]; exists {
			t.Errorf("duplicate confirmation code %q found at indices %d and %d", code, prev, i)
		}
		seen[code] = i
	}

	// Fail hard if any registrations were skipped — a partial success makes
	// the uniqueness assertion meaningless.
	if skipped > 0 {
		t.Fatalf("expected all %d registrations to succeed, but %d failed (only %d codes to compare)",
			concurrency, skipped, len(seen))
	}
}

// ── Test 4: Concurrent rate limit enforcement ────────────────────────────────

func TestConcurrentRateLimitEnforcement(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupPool(t)
	uid := setupPoolUser(t, pool)

	deps := &Deps{DB: pool, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Pre-fill the full quota sequentially.
	for i := 0; i < inviteRateLimit; i++ {
		r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("pre-fill invite %d: expected 201, got %d: %s", i, w.Code, w.Body.String())
		}
	}

	// Now send concurrent requests — they should all be rate limited since
	// the quota is already exhausted. All concurrent readers will see
	// count >= inviteRateLimit and reject.
	const concurrentRequests = 5

	// Create requests on the main goroutine (authenticatedJSONRequest uses t.Helper()).
	requests := make([]*http.Request, concurrentRequests)
	for i := 0; i < concurrentRequests; i++ {
		requests[i] = authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
	}

	var wg sync.WaitGroup
	statusCodes := make([]int, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, requests[idx])
			statusCodes[idx] = w.Code
		}(i)
	}

	wg.Wait()

	var created, rateLimited, other int
	for _, code := range statusCodes {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusTooManyRequests:
			rateLimited++
		default:
			other++
		}
	}

	// All concurrent requests should be rate limited since the quota was
	// fully exhausted before the concurrent batch.
	if rateLimited != concurrentRequests {
		t.Errorf("expected all %d concurrent requests to be rate-limited, got %d (created=%d, other=%d)",
			concurrentRequests, rateLimited, created, other)
	}

	// No unexpected status codes.
	if other > 0 {
		t.Errorf("got %d unexpected status codes (not 201 or 429)", other)
	}

	// Verify the total invites in the DB match exactly the pre-filled amount.
	var totalInvites int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM registration_invites WHERE user_id = $1`, uid,
	).Scan(&totalInvites)
	if err != nil {
		t.Fatalf("count invites: %v", err)
	}

	if totalInvites != inviteRateLimit {
		t.Errorf("expected exactly %d invites in DB, got %d", inviteRateLimit, totalInvites)
	}

	t.Logf("concurrent rate limit results: %d created, %d rate-limited, %d other (total in DB: %d)",
		created, rateLimited, other, totalInvites)
}

// ── Test 5: Last-slot contention (atomic rate limiter) ────────────────────────

// TestConcurrentRateLimitLastSlotContention races concurrent requests for the
// last available rate-limit slot. With the atomic check-and-insert rate limiter,
// exactly one goroutine should succeed and the rest should be rate-limited.
func TestConcurrentRateLimitLastSlotContention(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupPool(t)
	uid := setupPoolUser(t, pool)

	// Pre-fill to one below the limit so there is exactly one slot remaining.
	deps := &Deps{DB: pool, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	for i := 0; i < inviteRateLimit-1; i++ {
		r := authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("pre-fill invite %d: expected 201, got %d: %s", i, w.Code, w.Body.String())
		}
	}

	// Race 5 goroutines for the single remaining slot.
	// With the atomic check-and-insert, at most 1 should succeed.
	const concurrentRequests = 5

	// Create requests on the main goroutine (authenticatedJSONRequest uses t.Helper()).
	requests := make([]*http.Request, concurrentRequests)
	for i := 0; i < concurrentRequests; i++ {
		requests[i] = authenticatedJSONRequest(t, http.MethodPost, "/registration-invites", uid, `{}`)
	}

	var wg sync.WaitGroup
	statusCodes := make([]int, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, requests[idx])
			statusCodes[idx] = w.Code
		}(i)
	}

	wg.Wait()

	var created, rateLimited, other int
	for _, code := range statusCodes {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusTooManyRequests:
			rateLimited++
		default:
			other++
		}
	}

	// Exactly 1 should win the slot; the rest must be rate-limited.
	if created != 1 {
		t.Errorf("expected exactly 1 created invite, got %d (5 concurrent requests racing for the last slot)", created)
	}
	if other > 0 {
		t.Errorf("got %d unexpected status codes (not 201 or 429)", other)
	}

	// Verify the total invites in the DB do not exceed the rate limit.
	var totalInvites int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM registration_invites WHERE user_id = $1`, uid,
	).Scan(&totalInvites)
	if err != nil {
		t.Fatalf("count invites: %v", err)
	}

	if totalInvites > inviteRateLimit {
		t.Errorf("total invites (%d) exceeds rate limit (%d)", totalInvites, inviteRateLimit)
	}

	t.Logf("last-slot contention: %d created, %d rate-limited, %d other (total in DB: %d, limit: %d)",
		created, rateLimited, other, totalInvites, inviteRateLimit)
}

// ── Test 6: Concurrent verification attempt counting ─────────────────────────

func TestConcurrentVerificationAttemptCounting(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupPool(t)
	uid := setupPoolUser(t, pool)

	// Create a pending agent.
	agent, err := db.InsertPendingAgent(context.Background(), pool,
		uid, "ssh-ed25519 AAAAtest_concurrent_verify", "XY3-ZW4", 900, nil)
	if err != nil {
		t.Fatalf("InsertPendingAgent: %v", err)
	}

	// Send 8 concurrent wrong-code verification attempts.
	// The lockout threshold is 5, so exactly 5 should increment attempts,
	// and the remaining should see the lockout.
	const totalAttempts = 8

	var wg sync.WaitGroup
	results := make([]error, totalAttempts)

	for i := 0; i < totalAttempts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := db.VerifyAgentConfirmationCode(
				context.Background(), pool, agent.AgentID, "ZZZZZZ")
			results[idx] = err
		}(i)
	}

	wg.Wait()

	var invalidCount, lockedCount, otherCount int
	for _, err := range results {
		switch {
		case err == db.ErrInvalidConfirmation:
			invalidCount++
		case err == db.ErrVerificationLocked:
			lockedCount++
		default:
			otherCount++
			if err != nil {
				t.Logf("unexpected error: %v", err)
			}
		}
	}

	// Exactly 5 attempts should have been counted (returning ErrInvalidConfirmation).
	if invalidCount != 5 {
		t.Errorf("expected exactly 5 ErrInvalidConfirmation, got %d", invalidCount)
	}

	// The remaining should have been locked out.
	if lockedCount != totalAttempts-5 {
		t.Errorf("expected %d ErrVerificationLocked, got %d", totalAttempts-5, lockedCount)
	}

	if otherCount > 0 {
		t.Errorf("got %d unexpected results", otherCount)
	}

	// Verify the final state: exactly 5 attempts recorded.
	finalAgent, err := db.GetAgentByIDUnscoped(context.Background(), pool, agent.AgentID)
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if finalAgent.VerificationAttempts != 5 {
		t.Errorf("expected exactly 5 verification_attempts, got %d", finalAgent.VerificationAttempts)
	}

	// The agent should still be pending (no correct code was submitted).
	if finalAgent.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", finalAgent.Status)
	}

	// Now verify that the correct code is rejected (locked out).
	_, err = db.VerifyAgentConfirmationCode(context.Background(), pool, agent.AgentID, "XY3ZW4")
	if err != db.ErrVerificationLocked {
		t.Errorf("expected ErrVerificationLocked after lockout, got %v", err)
	}

	t.Logf("concurrent verification results: %d invalid, %d locked, %d other", invalidCount, lockedCount, otherCount)
}
