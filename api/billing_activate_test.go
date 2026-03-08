package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
	pstripe "github.com/supersuit-tech/permission-slip-web/stripe"
)

// ── POST /billing/activate ──────────────────────────────────────────────────

func TestActivate_MissingSessionID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: pstripe.New(pstripe.Config{})}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/billing/activate", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error.Code != ErrInvalidRequest {
		t.Errorf("expected error code %s, got %s", ErrInvalidRequest, errResp.Error.Code)
	}
}

func TestActivate_AlreadyOnPaidPlan(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: pstripe.New(pstripe.Config{})}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/billing/activate", uid, `{"session_id":"cs_test_123"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp activateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Status != "already_active" {
		t.Errorf("expected status=already_active, got %s", resp.Status)
	}
	if resp.PlanID != db.PlanPayAsYouGo {
		t.Errorf("expected plan_id=%s, got %s", db.PlanPayAsYouGo, resp.PlanID)
	}
}

func TestActivate_NoStripeClient(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// No Stripe client → 503.
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: nil}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/billing/activate", uid, `{"session_id":"cs_test_123"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestActivate_NoSubscription(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	// No subscription at all — should hit the Stripe call which will fail,
	// but the nil subscription check for customer mismatch should catch it.

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: pstripe.New(pstripe.Config{})}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/billing/activate", uid, `{"session_id":"cs_test_123"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// Will fail at Stripe retrieval (no real API key) — returns 502.
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestActivate_CustomerMismatch(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

	// User has a different Stripe customer ID than what the session returns.
	customerID := "cus_user_123"
	if _, err := db.UpdateSubscriptionStripe(ctx, tx, uid, &customerID, nil); err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}

	// This test can't fully exercise the mismatch path because we'd need a
	// real Stripe API call to return a session with a different customer.
	// The test for NoStripeClient above validates the error path instead.
	// A full integration test would use Stripe's test mode.

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BillingEnabled: true, Stripe: pstripe.New(pstripe.Config{})}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/billing/activate", uid, `{"session_id":"cs_test_fake"}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// Will fail at Stripe retrieval (no real API key) — returns 502.
	// In production with a real session, this would check customer mismatch.
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}
