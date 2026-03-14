package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
	pstripe "github.com/supersuit-tech/permission-slip-web/stripe"
)

// ── GET /billing/plan ──────────────────────────────────────────────────────

func TestGetBillingPlan_ReturnsPlanSubscriptionUsage(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Create agents, standing approvals, credentials, and requests.
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	testhelper.InsertCredential(t, tx, testhelper.GenerateID(t, "cred_"), uid, "github")

	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	for i := 0; i < 3; i++ {
		if _, err := db.IncrementRequestCount(ctx, tx, uid, periodStart, periodEnd); err != nil {
			t.Fatalf("IncrementRequestCount: %v", err)
		}
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/plan", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp billingPlanResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Plan fields.
	if resp.Plan.ID != db.PlanFree {
		t.Errorf("expected plan.id=%s, got %s", db.PlanFree, resp.Plan.ID)
	}
	if resp.Plan.Name == "" {
		t.Error("expected plan.name to be non-empty")
	}
	if resp.Plan.AuditRetentionDays == 0 {
		t.Error("expected plan.audit_retention_days > 0")
	}

	// Subscription fields.
	if resp.Subscription.Status != "active" {
		t.Errorf("expected subscription.status=active, got %s", resp.Subscription.Status)
	}
	if !resp.Subscription.CanUpgrade {
		t.Error("expected subscription.can_upgrade=true for free plan")
	}
	if resp.Subscription.CanDowngrade {
		t.Error("expected subscription.can_downgrade=false for free plan")
	}

	// Usage fields.
	if resp.Usage.Requests != 3 {
		t.Errorf("expected usage.requests=3, got %d", resp.Usage.Requests)
	}
	if resp.Usage.Agents != 1 {
		t.Errorf("expected usage.agents=1, got %d", resp.Usage.Agents)
	}
	if resp.Usage.StandingApprovals != 1 {
		t.Errorf("expected usage.standing_approvals=1, got %d", resp.Usage.StandingApprovals)
	}
	if resp.Usage.Credentials != 1 {
		t.Errorf("expected usage.credentials=1, got %d", resp.Usage.Credentials)
	}
}

func TestGetBillingPlan_NoSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/plan", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetBillingPlan_ZeroUsage(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/plan", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp billingPlanResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Usage.Requests != 0 {
		t.Errorf("expected usage.requests=0, got %d", resp.Usage.Requests)
	}
	if resp.Usage.Agents != 0 {
		t.Errorf("expected usage.agents=0, got %d", resp.Usage.Agents)
	}
	if resp.Usage.StandingApprovals != 0 {
		t.Errorf("expected usage.standing_approvals=0, got %d", resp.Usage.StandingApprovals)
	}
	if resp.Usage.Credentials != 0 {
		t.Errorf("expected usage.credentials=0, got %d", resp.Usage.Credentials)
	}
}

func TestGetBillingPlan_PaidPlan(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Set Stripe customer ID to verify has_payment_method.
	customerID := "cus_test_plan"
	if _, err := db.UpdateSubscriptionStripe(ctx, tx, uid, &customerID, nil); err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/plan", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp billingPlanResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Plan.ID != db.PlanPayAsYouGo {
		t.Errorf("expected plan.id=%s, got %s", db.PlanPayAsYouGo, resp.Plan.ID)
	}
	if resp.Subscription.CanUpgrade {
		t.Error("expected can_upgrade=false for paid plan")
	}
	if !resp.Subscription.CanDowngrade {
		t.Error("expected can_downgrade=true for paid plan")
	}
	if !resp.Subscription.HasPaymentMethod {
		t.Error("expected has_payment_method=true when stripe_customer_id is set")
	}
}

// ── GET /billing/subscription ─────────────────────────────────────────────

func TestGetSubscription_ReturnsSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	// Create profile and subscription (simulating onboarding).
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/subscription", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp subscriptionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.PlanID != db.PlanFree {
		t.Errorf("expected plan_id=%s, got %s", db.PlanFree, resp.PlanID)
	}
	if resp.Status != "active" {
		t.Errorf("expected status=active, got %s", resp.Status)
	}
	if resp.HasPaymentMethod {
		t.Error("expected has_payment_method=false for free plan")
	}
	if !resp.CanUpgrade {
		t.Error("expected can_upgrade=true for free plan")
	}
	if resp.CanDowngrade {
		t.Error("expected can_downgrade=false for free plan")
	}
	if resp.PlanLimits.AuditRetentionDays == 0 {
		t.Error("expected plan_limits.audit_retention_days > 0")
	}
}

func TestGetSubscription_NoSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	// Create profile but no subscription.
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/subscription", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetSubscription_WithUsage(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Record some usage for the current period.
	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	for i := 0; i < 5; i++ {
		if _, err := db.IncrementRequestCount(ctx, tx, uid, periodStart, periodEnd); err != nil {
			t.Fatalf("IncrementRequestCount: %v", err)
		}
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/subscription", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp subscriptionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Usage == nil {
		t.Fatal("expected usage to be present")
	}
	if resp.Usage.RequestCount != 5 {
		t.Errorf("expected request_count=5, got %d", resp.Usage.RequestCount)
	}
	if resp.Usage.OverLimit {
		t.Error("expected over_limit=false for 5 requests on free plan (250 limit)")
	}
}

func TestCreateCheckout_RequiresStripeClient(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// No Stripe client configured → should return 503.
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: nil}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/billing/checkout", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCheckout_AlreadyPaid(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Business logic (already-paid) is checked before Stripe dependency,
	// so even with nil Stripe the user gets the correct 409 error.
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: nil}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/billing/checkout", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error.Code != ErrAlreadySubscribed {
		t.Errorf("expected error code %s, got %s", ErrAlreadySubscribed, errResp.Error.Code)
	}
}

func TestGetSubscription_HasPaymentMethodWhenStripeCustomerSet(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Set Stripe customer ID.
	customerID := "cus_test_payment"
	if _, err := db.UpdateSubscriptionStripe(ctx, tx, uid, &customerID, nil); err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/subscription", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp subscriptionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.HasPaymentMethod {
		t.Error("expected has_payment_method=true when stripe_customer_id is set")
	}
	if resp.CanUpgrade {
		t.Error("expected can_upgrade=false for pay_as_you_go plan")
	}
	if !resp.CanDowngrade {
		t.Error("expected can_downgrade=true for pay_as_you_go plan")
	}
}

// ── GET /billing/usage ────────────────────────────────────────────────────

func TestGetUsage_ReturnsUsage(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Record some usage.
	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	for i := 0; i < 5; i++ {
		if _, err := db.IncrementRequestCount(ctx, tx, uid, periodStart, periodEnd); err != nil {
			t.Fatalf("IncrementRequestCount: %v", err)
		}
	}
	for i := 0; i < 3; i++ {
		if _, err := db.IncrementSMSCount(ctx, tx, uid, periodStart, periodEnd); err != nil {
			t.Fatalf("IncrementSMSCount: %v", err)
		}
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/usage", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp usageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Requests.Total != 5 {
		t.Errorf("expected requests.total=5, got %d", resp.Requests.Total)
	}
	if resp.Requests.Included != 250 {
		t.Errorf("expected requests.included=250, got %d", resp.Requests.Included)
	}
	if resp.Requests.Overage != 0 {
		t.Errorf("expected requests.overage=0, got %d", resp.Requests.Overage)
	}
	if resp.Requests.CostCents != 0 {
		t.Errorf("expected requests.overage_cost_cents=0, got %d", resp.Requests.CostCents)
	}
	if resp.SMS.Total != 3 {
		t.Errorf("expected sms.total=3, got %d", resp.SMS.Total)
	}
	if resp.SMS.CostCents != 3 {
		t.Errorf("expected sms.cost_cents=3, got %d", resp.SMS.CostCents)
	}
}

func TestGetUsage_NoSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/usage", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetUsage_ZeroUsage(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/usage", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp usageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Requests.Total != 0 {
		t.Errorf("expected requests.total=0, got %d", resp.Requests.Total)
	}
	if resp.Requests.Included != 250 {
		t.Errorf("expected requests.included=250 (free plan), got %d", resp.Requests.Included)
	}
}

func TestGetUsage_WithOverage(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Insert usage that exceeds the free limit (250) directly.
	periodStart, periodEnd := db.BillingPeriodBounds(time.Now())
	testhelper.MustExec(t, tx,
		`INSERT INTO usage_periods (user_id, period_start, period_end, request_count)
		 VALUES ($1, $2, $3, 300)`,
		uid, periodStart, periodEnd)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/usage", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp usageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Requests.Total != 300 {
		t.Errorf("expected requests.total=300, got %d", resp.Requests.Total)
	}
	if resp.Requests.Overage != 50 {
		t.Errorf("expected requests.overage=50, got %d", resp.Requests.Overage)
	}
	// 50 requests at $0.005 = 25 cents. Formula: ceil(50 * 0.5) = 25
	if resp.Requests.CostCents != 25 {
		t.Errorf("expected requests.overage_cost_cents=25, got %d", resp.Requests.CostCents)
	}
}

func TestGetUsage_HistoricalPeriod(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Insert usage for a past period (February 2026).
	testhelper.MustExec(t, tx,
		`INSERT INTO usage_periods (user_id, period_start, period_end, request_count, sms_count)
		 VALUES ($1, '2026-02-01T00:00:00Z', '2026-03-01T00:00:00Z', 800, 2)`,
		uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/usage?period_start=2026-02-01T00:00:00Z", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp usageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Requests.Total != 800 {
		t.Errorf("expected requests.total=800, got %d", resp.Requests.Total)
	}
	if resp.SMS.Total != 2 {
		t.Errorf("expected sms.total=2, got %d", resp.SMS.Total)
	}
	// Period bounds should reflect the requested period, not current.
	if resp.PeriodStart.Month() != 2 {
		t.Errorf("expected period_start month=2, got %d", resp.PeriodStart.Month())
	}
}

func TestGetUsage_InvalidPeriodStart(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/usage?period_start=not-a-date", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ── POST /billing/upgrade ─────────────────────────────────────────────────

func TestUpgrade_RequiresStripeClient(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// No Stripe client → 503.
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: nil}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/billing/upgrade", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// ── POST /billing/downgrade ───────────────────────────────────────────────

func TestDowngrade_AlreadyFree(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/billing/downgrade", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error.Code != ErrAlreadyDowngraded {
		t.Errorf("expected error code %s, got %s", ErrAlreadyDowngraded, errResp.Error.Code)
	}
}

func TestDowngrade_NoSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/billing/downgrade", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDowngrade_Success_NoStripe(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// No Stripe client — downgrade should still succeed locally.
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: nil}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/billing/downgrade", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp downgradeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.PlanID != db.PlanFree {
		t.Errorf("expected plan_id=%s, got %s", db.PlanFree, resp.PlanID)
	}

	// Verify subscription was actually downgraded in DB.
	sub, err := db.GetSubscriptionByUserID(context.Background(), tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub.PlanID != db.PlanFree {
		t.Errorf("DB subscription plan_id: expected %s, got %s", db.PlanFree, sub.PlanID)
	}
	if sub.DowngradedAt == nil {
		t.Error("expected downgraded_at to be set after downgrade")
	}
}

func TestDowngrade_TooManyAgents(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Create more registered agents than the free plan allows.
	// Free plan limit is 5 agents (from seed data).
	for i := 0; i < 6; i++ {
		testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/billing/downgrade", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error.Code != ErrDowngradeLimitExceeded {
		t.Errorf("expected error code %s, got %s", ErrDowngradeLimitExceeded, errResp.Error.Code)
	}
	if errResp.Error.Details["resource"] != "agents" {
		t.Errorf("expected resource=agents, got %v", errResp.Error.Details["resource"])
	}
}

func TestDowngrade_TooManyStandingApprovals(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	// Free plan limit is 10 active standing approvals (from seed data).
	for i := 0; i < 11; i++ {
		saID := testhelper.GenerateID(t, "sa_")
		testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/billing/downgrade", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error.Code != ErrDowngradeLimitExceeded {
		t.Errorf("expected error code %s, got %s", ErrDowngradeLimitExceeded, errResp.Error.Code)
	}
	if errResp.Error.Details["resource"] != "standing_approvals" {
		t.Errorf("expected resource=standing_approvals, got %v", errResp.Error.Details["resource"])
	}
}

func TestDowngrade_TooManyCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Free plan limit is 5 credentials (from seed data).
	for i := 0; i < 6; i++ {
		credID := testhelper.GenerateID(t, "cred_")
		service := testhelper.GenerateID(t, "svc_")
		testhelper.InsertCredential(t, tx, credID, uid, service)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/billing/downgrade", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error.Code != ErrDowngradeLimitExceeded {
		t.Errorf("expected error code %s, got %s", ErrDowngradeLimitExceeded, errResp.Error.Code)
	}
	if errResp.Error.Details["resource"] != "credentials" {
		t.Errorf("expected resource=credentials, got %v", errResp.Error.Details["resource"])
	}
}

// ── GET /billing/invoices ─────────────────────────────────────────────────

func TestListInvoices_RequiresStripeClient(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: nil}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/invoices", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListInvoices_NoStripeCustomer(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// Stripe is configured but user has no Stripe customer ID → empty list.
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: pstripe.New(pstripe.Config{})}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/invoices", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp invoiceListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Invoices) != 0 {
		t.Errorf("expected empty invoices list, got %d", len(resp.Invoices))
	}
}

// ── toAPIInvoice mapping ────────────────────────────────────────────────────

func TestToAPIInvoice_FullFields(t *testing.T) {
	t.Parallel()
	hostedURL := "https://invoice.stripe.com/i/acct_123/inv_456"
	s := pstripe.InvoiceSummary{
		ID:          "inv_456",
		Status:      "paid",
		AmountPaid:  1099,
		Created:     1740787200, // 2025-03-01T00:00:00Z
		PeriodStart: 1738368000, // 2025-02-01T00:00:00Z
		PeriodEnd:   1740787200, // 2025-03-01T00:00:00Z
		HostedURL:   hostedURL,
	}

	inv := toAPIInvoice(s)

	if inv.ID != "inv_456" {
		t.Errorf("ID: got %q, want %q", inv.ID, "inv_456")
	}
	if inv.AmountCents != 1099 {
		t.Errorf("AmountCents: got %d, want 1099", inv.AmountCents)
	}
	if inv.Status != "paid" {
		t.Errorf("Status: got %q, want %q", inv.Status, "paid")
	}
	wantDate := time.Unix(1740787200, 0).UTC().Format(time.RFC3339)
	if inv.Date != wantDate {
		t.Errorf("Date: got %q, want %q", inv.Date, wantDate)
	}
	if inv.PeriodStart == nil {
		t.Fatal("PeriodStart: expected non-nil")
	}
	wantPS := time.Unix(1738368000, 0).UTC().Format(time.RFC3339)
	if *inv.PeriodStart != wantPS {
		t.Errorf("PeriodStart: got %q, want %q", *inv.PeriodStart, wantPS)
	}
	if inv.PeriodEnd == nil {
		t.Fatal("PeriodEnd: expected non-nil")
	}
	wantPE := time.Unix(1740787200, 0).UTC().Format(time.RFC3339)
	if *inv.PeriodEnd != wantPE {
		t.Errorf("PeriodEnd: got %q, want %q", *inv.PeriodEnd, wantPE)
	}
	if inv.StripeInvoiceURL == nil {
		t.Fatal("StripeInvoiceURL: expected non-nil")
	}
	if *inv.StripeInvoiceURL != hostedURL {
		t.Errorf("StripeInvoiceURL: got %q, want %q", *inv.StripeInvoiceURL, hostedURL)
	}
	// Ensure pointer independence — mutating the original doesn't affect the copy.
	s.HostedURL = "https://other.example.com"
	if *inv.StripeInvoiceURL != hostedURL {
		t.Error("StripeInvoiceURL pointer aliases original struct field")
	}
}

func TestToAPIInvoice_OptionalFieldsAbsent(t *testing.T) {
	t.Parallel()
	s := pstripe.InvoiceSummary{
		ID:      "inv_789",
		Status:  "paid",
		Created: 1740787200,
		// PeriodStart, PeriodEnd, and HostedURL intentionally zero/empty
	}

	inv := toAPIInvoice(s)

	if inv.PeriodStart != nil {
		t.Errorf("PeriodStart: expected nil when zero, got %q", *inv.PeriodStart)
	}
	if inv.PeriodEnd != nil {
		t.Errorf("PeriodEnd: expected nil when zero, got %q", *inv.PeriodEnd)
	}
	if inv.StripeInvoiceURL != nil {
		t.Errorf("StripeInvoiceURL: expected nil when empty, got %q", *inv.StripeInvoiceURL)
	}
}

func TestListInvoices_NoSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	// No subscription → empty list (not an error).

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: pstripe.New(pstripe.Config{})}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/billing/invoices", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp invoiceListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Invoices) != 0 {
		t.Errorf("expected empty invoices list, got %d", len(resp.Invoices))
	}
}

