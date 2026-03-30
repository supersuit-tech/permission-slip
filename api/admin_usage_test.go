package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// ── GET /admin/usage ────────────────────────────────────────────────────────

func TestAdminGetUsage_ReturnsAuthenticatedUserData(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	for i := 0; i < 5; i++ {
		if _, err := db.IncrementRequestCount(ctx, tx, uid, periodStart, periodEnd); err != nil {
			t.Fatalf("IncrementRequestCount: %v", err)
		}
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/admin/usage", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp adminUsageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.UserID != uid {
		t.Errorf("expected user_id=%s, got %s", uid, resp.UserID)
	}
	if resp.Requests != 5 {
		t.Errorf("expected request_count=5, got %d", resp.Requests)
	}
}

func TestAdminGetUsage_IgnoresUserIDParam(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	// Create two users, only the authenticated one should have data returned.
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u1_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:8])

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	for i := 0; i < 10; i++ {
		if _, err := db.IncrementRequestCount(ctx, tx, uid2, periodStart, periodEnd); err != nil {
			t.Fatalf("IncrementRequestCount: %v", err)
		}
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Try to view uid2's data while authenticated as uid1 — should return uid1's data (0), not uid2's.
	r := authenticatedRequest(t, http.MethodGet, "/admin/usage?user_id="+uid2, uid1)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp adminUsageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	// Must return the authenticated user's ID and data, NOT the queried user_id.
	if resp.UserID != uid1 {
		t.Errorf("IDOR: expected user_id=%s (authenticated user), got %s", uid1, resp.UserID)
	}
	if resp.Requests != 0 {
		t.Errorf("IDOR: expected request_count=0 for uid1, got %d (may be returning uid2's data)", resp.Requests)
	}
}

func TestAdminGetUsage_WithBreakdown(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	if _, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid, periodStart, periodEnd, db.UsageBreakdownKeys{
		AgentID: 1, ConnectorID: "gmail", ActionType: "email.send",
	}); err != nil {
		t.Fatalf("IncrementRequestCountWithBreakdown: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/admin/usage", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp adminUsageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Breakdown == nil {
		t.Fatal("expected breakdown to be non-nil")
	}
	if resp.Breakdown.ByAgent["1"] != 1 {
		t.Errorf("expected by_agent[1]=1, got %d", resp.Breakdown.ByAgent["1"])
	}
	if resp.Breakdown.ByConnector["gmail"] != 1 {
		t.Errorf("expected by_connector[gmail]=1, got %d", resp.Breakdown.ByConnector["gmail"])
	}
}

func TestAdminGetUsage_ZeroUsage(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/admin/usage", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp adminUsageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Requests != 0 {
		t.Errorf("expected request_count=0, got %d", resp.Requests)
	}
	if resp.Breakdown != nil {
		t.Error("expected breakdown to be nil for zero usage")
	}
}

func TestAdminGetUsage_InvalidPeriod(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/admin/usage?period=not-a-date", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminGetUsage_FlexiblePeriodFormats(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	formats := []string{
		"2026-03",
		"2026-03-01",
		"2026-03-01T00:00:00Z",
	}

	for _, f := range formats {
		r := authenticatedRequest(t, http.MethodGet, "/admin/usage?period="+f, uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("period=%s: expected 200, got %d: %s", f, w.Code, w.Body.String())
		}
	}
}

// ── GET /admin/usage/by-connector ──────────────────────────────────────────

func TestAdminUsageByConnector_ScopedToAuthenticatedUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	for i := 0; i < 4; i++ {
		if _, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid, periodStart, periodEnd, db.UsageBreakdownKeys{
			AgentID: 1, ConnectorID: "gmail", ActionType: "email.send",
		}); err != nil {
			t.Fatalf("increment gmail: %v", err)
		}
	}
	if _, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid, periodStart, periodEnd, db.UsageBreakdownKeys{
		AgentID: 1, ConnectorID: "stripe", ActionType: "payment.create",
	}); err != nil {
		t.Fatalf("increment stripe: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/admin/usage/by-connector", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp connectorUsageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.UserID != uid {
		t.Errorf("expected user_id=%s, got %s", uid, resp.UserID)
	}
	if len(resp.Connectors) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(resp.Connectors))
	}
	if resp.Connectors[0].ConnectorID != "gmail" {
		t.Errorf("expected first connector=gmail, got %s", resp.Connectors[0].ConnectorID)
	}
	if resp.Connectors[0].RequestCount != 4 {
		t.Errorf("expected gmail count=4, got %d", resp.Connectors[0].RequestCount)
	}
}

func TestAdminUsageByConnector_DoesNotLeakCrossUserData(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u1_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:8])

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	// uid2 has 10 requests via slack — uid1 should NOT see this.
	for i := 0; i < 10; i++ {
		if _, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid2, periodStart, periodEnd, db.UsageBreakdownKeys{
			AgentID: 1, ConnectorID: "slack", ActionType: "message.send",
		}); err != nil {
			t.Fatalf("increment uid2: %v", err)
		}
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// uid1 should see 0 connectors (no usage of their own).
	r := authenticatedRequest(t, http.MethodGet, "/admin/usage/by-connector", uid1)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp connectorUsageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Connectors) != 0 {
		t.Errorf("cross-tenant leak: expected 0 connectors for uid1, got %d", len(resp.Connectors))
	}
}

// ── GET /admin/usage/by-agent ──────────────────────────────────────────────

func TestAdminUsageByAgent_ScopedToAuthenticatedUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	for i := 0; i < 3; i++ {
		if _, err := db.IncrementRequestCountWithBreakdown(ctx, tx, uid, periodStart, periodEnd, db.UsageBreakdownKeys{
			AgentID: 42, ConnectorID: "gmail", ActionType: "email.send",
		}); err != nil {
			t.Fatalf("increment: %v", err)
		}
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/admin/usage/by-agent", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentUsageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.UserID != uid {
		t.Errorf("expected user_id=%s, got %s", uid, resp.UserID)
	}
	if len(resp.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.Agents))
	}
	if resp.Agents[0].AgentID != "42" {
		t.Errorf("expected agent_id=42, got %s", resp.Agents[0].AgentID)
	}
	if resp.Agents[0].RequestCount != 3 {
		t.Errorf("expected request_count=3, got %d", resp.Agents[0].RequestCount)
	}
}

func TestAdminUsageByAgent_EmptyResult(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/admin/usage/by-agent", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentUsageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Agents == nil {
		t.Error("expected agents to be an empty array, not null")
	}
}

// ── Auth ────────────────────────────────────────────────────────────────────

func TestAdminEndpoints_RequireAuth(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	endpoints := []string{
		"/admin/usage",
		"/admin/usage/by-connector",
		"/admin/usage/by-agent",
	}

	for _, ep := range endpoints {
		r := httptest.NewRequest(http.MethodGet, ep, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: expected 401, got %d", ep, w.Code)
		}
	}
}
