package api

// quota_test.go — tests for monthly request quota enforcement.
//
// These tests verify that free tier users are blocked with 429 when their
// monthly request quota is exhausted, and that paid tier users bypass the limit.
// The quota is checked before processing approval requests and standing
// approval executions (both agent and dashboard paths).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// ── checkRequestQuota unit tests ────────────────────────────────────────────

func TestCheckRequestQuota_FreePlan_UnderLimit_Allows(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// 249 requests used (under limit of 250).
	testhelper.SetUsageCount(t, tx, uid, 249)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	r, blocked := checkRequestQuota(r.Context(), w, r, tx, uid)

	if blocked {
		t.Fatalf("expected under-limit request to be allowed, got blocked: %s", w.Body.String())
	}

	// Context should be marked as quota-reserved after successful check.
	if !IsQuotaReserved(r.Context()) {
		t.Error("expected context to be marked as quota-reserved after successful check")
	}

	// Verify informational quota headers are set on allowed requests.
	if w.Header().Get("X-Quota-Limit") != "250" {
		t.Errorf("expected X-Quota-Limit=250, got %q", w.Header().Get("X-Quota-Limit"))
	}
	if w.Header().Get("X-Quota-Remaining") != "0" {
		t.Errorf("expected X-Quota-Remaining=0 (249 used, +1 for this request), got %q", w.Header().Get("X-Quota-Remaining"))
	}
	if w.Header().Get("X-Quota-Reset") == "" {
		t.Error("expected X-Quota-Reset header to be set")
	}
}

func TestCheckRequestQuota_FreePlan_AtLimit_Returns429(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Exactly at the limit (250).
	testhelper.SetUsageCount(t, tx, uid, 250)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	_, blocked := checkRequestQuota(r.Context(), w, r, tx, uid)

	if !blocked {
		t.Fatal("expected at-limit request to be blocked")
	}
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error.Code != ErrMonthlyQuotaExceeded {
		t.Errorf("expected error code %q, got %q", ErrMonthlyQuotaExceeded, resp.Error.Code)
	}
	if !resp.Error.Retryable {
		t.Error("expected retryable=true")
	}
	if resp.Error.RetryAfter <= 0 {
		t.Errorf("expected positive retry_after, got %d", resp.Error.RetryAfter)
	}
	if resp.Error.Details == nil {
		t.Fatal("expected details in error response")
	}
	if usage, ok := resp.Error.Details["current_usage"].(float64); !ok || int(usage) != 250 {
		t.Errorf("expected current_usage=250, got %v", resp.Error.Details["current_usage"])
	}
	if limit, ok := resp.Error.Details["limit"].(float64); !ok || int(limit) != 250 {
		t.Errorf("expected limit=250, got %v", resp.Error.Details["limit"])
	}
	if planID, ok := resp.Error.Details["plan_id"].(string); !ok || planID != db.PlanFree {
		t.Errorf("expected plan_id=%q, got %v", db.PlanFree, resp.Error.Details["plan_id"])
	}
	if resetAt, ok := resp.Error.Details["reset_at"].(string); !ok || resetAt == "" {
		t.Errorf("expected non-empty reset_at timestamp, got %v", resp.Error.Details["reset_at"])
	} else {
		if _, err := time.Parse(time.RFC3339, resetAt); err != nil {
			t.Errorf("reset_at is not valid RFC3339: %q", resetAt)
		}
	}
}

func TestCheckRequestQuota_FreePlan_OverLimit_Returns429(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Over the limit.
	testhelper.SetUsageCount(t, tx, uid, 1500)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	_, blocked := checkRequestQuota(r.Context(), w, r, tx, uid)

	if !blocked {
		t.Fatal("expected over-limit request to be blocked")
	}
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestCheckRequestQuota_FreePlan_ZeroUsage_Allows(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)
	// No usage row at all — should be allowed.

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	_, blocked := checkRequestQuota(r.Context(), w, r, tx, uid)

	if blocked {
		t.Fatalf("expected zero-usage request to be allowed, got blocked: %s", w.Body.String())
	}
}

func TestCheckRequestQuota_PaidPlan_Bypass(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Even with usage far exceeding the free limit, paid plan is unlimited.
	testhelper.SetUsageCount(t, tx, uid, 50000)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	_, blocked := checkRequestQuota(r.Context(), w, r, tx, uid)

	if blocked {
		t.Fatalf("expected paid plan to bypass quota, but got blocked: %s", w.Body.String())
	}
}

func TestCheckRequestQuota_NoSubscription_Allows(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	// No subscription — should not block.

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	_, blocked := checkRequestQuota(r.Context(), w, r, tx, uid)

	if blocked {
		t.Fatalf("expected no subscription to bypass quota, but got blocked: %s", w.Body.String())
	}
}

func TestCheckRequestQuota_RetryAfterHeader(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)
	testhelper.SetUsageCount(t, tx, uid, 250)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	_, _ = checkRequestQuota(r.Context(), w, r, tx, uid)

	retryAfter := w.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("expected Retry-After header to be set")
	}

	// The Retry-After value should be a positive integer representing seconds
	// until the billing period ends (first of next month).
	_, periodEnd := db.BillingPeriodBounds(time.Now())
	expectedMax := int(time.Until(periodEnd).Seconds()) + 2 // +2 for clock skew
	var retryAfterInt int
	if _, err := fmt.Sscanf(retryAfter, "%d", &retryAfterInt); err != nil {
		t.Fatalf("Retry-After not a valid integer: %q", retryAfter)
	}
	if retryAfterInt < 1 {
		t.Errorf("Retry-After should be >= 1, got %d", retryAfterInt)
	}
	if retryAfterInt > expectedMax {
		t.Errorf("Retry-After %d exceeds expected max ~%d", retryAfterInt, expectedMax)
	}

	// Verify the JSON body also has retry_after matching the header.
	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.RetryAfter != retryAfterInt {
		t.Errorf("JSON retry_after=%d does not match header Retry-After=%d",
			resp.Error.RetryAfter, retryAfterInt)
	}
}

// ── Integration: approval request blocked by quota ──────────────────────────

func TestQuota_ApprovalRequest_FreePlan_AtLimit_Returns429(t *testing.T) {
	t.Parallel()
	m := setupMeteringTest(t)
	testhelper.InsertSubscription(t, m.DB, m.UserID, db.PlanFree)
	testhelper.SetUsageCount(t, m.DB, m.UserID, 250)

	reqBody := `{"request_id":"quota_blocked_001","action":{"type":"email.send","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, m.PrivKey, m.AgentID)
	w := httptest.NewRecorder()
	m.Router.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Code != ErrMonthlyQuotaExceeded {
		t.Errorf("expected error code %q, got %q", ErrMonthlyQuotaExceeded, resp.Error.Code)
	}

	// Usage count should not have increased (request was rejected).
	testhelper.RequireUsageCount(t, m.DB, m.UserID, 250)
}

func TestQuota_ApprovalRequest_FreePlan_UnderLimit_Succeeds(t *testing.T) {
	t.Parallel()
	m := setupMeteringTest(t)
	testhelper.InsertSubscription(t, m.DB, m.UserID, db.PlanFree)
	testhelper.SetUsageCount(t, m.DB, m.UserID, 249)

	reqBody := `{"request_id":"quota_ok_001","action":{"type":"email.send","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, m.PrivKey, m.AgentID)
	w := httptest.NewRecorder()
	m.Router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Usage should have incremented to 250.
	testhelper.RequireUsageCount(t, m.DB, m.UserID, 250)
}

func TestQuota_ApprovalRequest_PaidPlan_NoLimit(t *testing.T) {
	t.Parallel()
	m := setupMeteringTest(t)
	testhelper.InsertSubscription(t, m.DB, m.UserID, db.PlanPayAsYouGo)
	testhelper.SetUsageCount(t, m.DB, m.UserID, 50000)

	reqBody := `{"request_id":"quota_paid_001","action":{"type":"email.send","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, m.PrivKey, m.AgentID)
	w := httptest.NewRecorder()
	m.Router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (paid plan, unlimited), got %d: %s", w.Code, w.Body.String())
	}
}

// ── Integration: standing approval execution (agent path) blocked by quota ──

func TestQuota_AgentStandingExecution_FreePlan_AtLimit_Returns429(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, privKey, _, uid := setupStandingExecuteTest(t, "email.read")
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)
	testhelper.SetUsageCount(t, tx, uid, 250)

	reqBody := `{"request_id":"quota_sa_001","action":{"type":"email.read","version":"1","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", w.Code, w.Body.String())
	}

	// Usage should not have increased.
	testhelper.RequireUsageCount(t, tx, uid, 250)
}

func TestQuota_AgentStandingExecution_FreePlan_UnderLimit_Succeeds(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, privKey, _, uid := setupStandingExecuteTest(t, "email.read")
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)
	testhelper.SetUsageCount(t, tx, uid, 249)

	reqBody := `{"request_id":"quota_sa_ok_001","action":{"type":"email.read","version":"1","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Usage should have incremented.
	testhelper.RequireUsageCount(t, tx, uid, 250)
}

// ── Integration: standing approval execution (dashboard path) blocked by quota ──

func TestQuota_DashboardStandingExecution_FreePlan_AtLimit_Returns429(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "email.send")
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)
	testhelper.SetUsageCount(t, tx, uid, 250)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{"parameters":{}}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", w.Code, w.Body.String())
	}

	// Usage should not have increased.
	testhelper.RequireUsageCount(t, tx, uid, 250)
}

func TestQuota_DashboardStandingExecution_FreePlan_UnderLimit_Succeeds(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithActionType(t, tx, saID, agentID, uid, "email.send")
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)
	testhelper.SetUsageCount(t, tx, uid, 249)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{"parameters":{}}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Usage should have incremented.
	testhelper.RequireUsageCount(t, tx, uid, 250)
}

