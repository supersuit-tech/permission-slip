package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
	pstripe "github.com/supersuit-tech/permission-slip-web/stripe"
)

const testWebhookSecret = "whsec_test_secret_for_webhook_testing"

// signedWebhookRequest creates a POST request to the webhook endpoint with a
// valid Stripe signature. The payload is signed with testWebhookSecret so
// VerifyWebhook succeeds during tests.
func signedWebhookRequest(t *testing.T, payload []byte) *http.Request {
	t.Helper()
	now := time.Now()
	sig := webhook.ComputeSignature(now, payload, testWebhookSecret)
	sigHeader := fmt.Sprintf("t=%d,v1=%s", now.Unix(), hex.EncodeToString(sig))

	r := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", strings.NewReader(string(payload)))
	r.Header.Set("Stripe-Signature", sigHeader)
	r.Header.Set("Content-Type", "application/json")
	return r
}

// stripeEventPayload builds a minimal Stripe event JSON payload.
func stripeEventPayload(t *testing.T, eventID, eventType string, data interface{}) []byte {
	t.Helper()
	dataJSON, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal event data: %v", err)
	}
	payload := fmt.Sprintf(`{
		"id": %q,
		"type": %q,
		"data": {"object": %s},
		"livemode": false,
		"api_version": %q
	}`, eventID, eventType, string(dataJSON), gostripe.APIVersion)
	return []byte(payload)
}

// setupWebhookTest creates a user with a paid subscription and Stripe IDs,
// and returns the test deps and mux ready for webhook testing.
func setupWebhookTest(t *testing.T) (db.DBTX, *Deps, *http.ServeMux, string) {
	t.Helper()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

	// Set Stripe IDs.
	customerID := "cus_" + uid[:8]
	subscriptionID := "sub_" + uid[:8]
	if _, err := db.UpdateSubscriptionStripe(ctx, tx, uid, &customerID, &subscriptionID); err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}

	stripeClient := pstripe.New(pstripe.Config{
		WebhookSecret: testWebhookSecret,
	})

	deps := &Deps{
		DB:     tx,
		Stripe: stripeClient,
	}

	mux := http.NewServeMux()
	RegisterBillingWebhookRoutes(mux, deps)

	return tx, deps, mux, uid
}

// ── checkout.session.completed ──────────────────────────────────────────────

func TestWebhook_CheckoutSessionCompleted(t *testing.T) {
	t.Parallel()
	tx, _, mux, _ := setupWebhookTest(t)
	ctx := context.Background()

	// Create a new user on the free plan with a Stripe customer ID.
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertSubscription(t, tx, uid, db.PlanFree)
	customerID := "cus_checkout_" + uid[:8]
	if _, err := db.UpdateSubscriptionStripe(ctx, tx, uid, &customerID, nil); err != nil {
		t.Fatalf("UpdateSubscriptionStripe: %v", err)
	}

	payload := stripeEventPayload(t, "evt_checkout_1", "checkout.session.completed", map[string]interface{}{
		"id":           "cs_test_123",
		"customer":     map[string]interface{}{"id": customerID},
		"subscription": map[string]interface{}{"id": "sub_new_123"},
	})

	r := signedWebhookRequest(t, payload)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify subscription was upgraded.
	sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub.PlanID != db.PlanPayAsYouGo {
		t.Errorf("expected plan=%s, got %s", db.PlanPayAsYouGo, sub.PlanID)
	}
	if sub.StripeSubscriptionID == nil || *sub.StripeSubscriptionID != "sub_new_123" {
		t.Errorf("expected stripe_subscription_id=sub_new_123, got %v", sub.StripeSubscriptionID)
	}
}

// ── invoice.paid ────────────────────────────────────────────────────────────

func TestWebhook_InvoicePaid_SetsActiveAndSyncsPeriod(t *testing.T) {
	t.Parallel()
	tx, _, mux, uid := setupWebhookTest(t)
	ctx := context.Background()

	// Set subscription to past_due (simulating a previously failed payment).
	if _, err := db.UpdateSubscriptionStatus(ctx, tx, uid, db.SubscriptionStatusPastDue); err != nil {
		t.Fatalf("UpdateSubscriptionStatus: %v", err)
	}

	subID := "sub_" + uid[:8]
	periodStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC).Unix()
	periodEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).Unix()

	payload := stripeEventPayload(t, "evt_inv_paid_1", "invoice.paid", map[string]interface{}{
		"id":           "in_test_paid",
		"period_start": periodStart,
		"period_end":   periodEnd,
		"parent": map[string]interface{}{
			"subscription_details": map[string]interface{}{
				"subscription": map[string]interface{}{"id": subID},
			},
		},
	})

	r := signedWebhookRequest(t, payload)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify subscription status is now active.
	sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub.Status != db.SubscriptionStatusActive {
		t.Errorf("expected status=active, got %s", sub.Status)
	}

	// Verify period dates were synced.
	expectedStart := time.Unix(periodStart, 0)
	expectedEnd := time.Unix(periodEnd, 0)
	if !sub.CurrentPeriodStart.Equal(expectedStart) {
		t.Errorf("expected period_start=%v, got %v", expectedStart, sub.CurrentPeriodStart)
	}
	if !sub.CurrentPeriodEnd.Equal(expectedEnd) {
		t.Errorf("expected period_end=%v, got %v", expectedEnd, sub.CurrentPeriodEnd)
	}
}

func TestWebhook_InvoicePaid_NoParentSubscription_Skips(t *testing.T) {
	t.Parallel()
	_, _, mux, _ := setupWebhookTest(t)

	// Invoice without parent subscription (e.g. one-off charge).
	payload := stripeEventPayload(t, "evt_inv_paid_noparent", "invoice.paid", map[string]interface{}{
		"id": "in_test_nop",
	})

	r := signedWebhookRequest(t, payload)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	// Should still return 200 (not an error — just nothing to do).
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ── invoice.payment_failed ──────────────────────────────────────────────────

func TestWebhook_InvoicePaymentFailed_SetsPastDue(t *testing.T) {
	t.Parallel()
	tx, _, mux, uid := setupWebhookTest(t)
	ctx := context.Background()

	subID := "sub_" + uid[:8]
	payload := stripeEventPayload(t, "evt_inv_failed_1", "invoice.payment_failed", map[string]interface{}{
		"id": "in_test_failed",
		"parent": map[string]interface{}{
			"subscription_details": map[string]interface{}{
				"subscription": map[string]interface{}{"id": subID},
			},
		},
	})

	r := signedWebhookRequest(t, payload)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify subscription is now past_due.
	sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub.Status != db.SubscriptionStatusPastDue {
		t.Errorf("expected status=past_due, got %s", sub.Status)
	}
}

func TestWebhook_InvoicePaymentFailed_UnknownSubscription_Returns200(t *testing.T) {
	t.Parallel()
	_, _, mux, _ := setupWebhookTest(t)

	payload := stripeEventPayload(t, "evt_inv_failed_unknown", "invoice.payment_failed", map[string]interface{}{
		"id": "in_test_unknown",
		"parent": map[string]interface{}{
			"subscription_details": map[string]interface{}{
				"subscription": map[string]interface{}{"id": "sub_nonexistent"},
			},
		},
	})

	r := signedWebhookRequest(t, payload)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	// Unknown subscription is a no-op, not an error.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ── customer.subscription.updated ───────────────────────────────────────────

func TestWebhook_SubscriptionUpdated_SyncsStatus(t *testing.T) {
	t.Parallel()
	tx, _, mux, uid := setupWebhookTest(t)
	ctx := context.Background()

	subID := "sub_" + uid[:8]

	tests := []struct {
		name           string
		stripeStatus   string
		expectedStatus db.SubscriptionStatus
	}{
		{"active", "active", db.SubscriptionStatusActive},
		{"trialing", "trialing", db.SubscriptionStatusActive},
		{"past_due", "past_due", db.SubscriptionStatusPastDue},
		{"unpaid", "unpaid", db.SubscriptionStatusPastDue},
		{"canceled", "canceled", db.SubscriptionStatusCancelled},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventID := fmt.Sprintf("evt_sub_updated_%d", i)
			payload := stripeEventPayload(t, eventID, "customer.subscription.updated", map[string]interface{}{
				"id":     subID,
				"status": tt.stripeStatus,
			})

			r := signedWebhookRequest(t, payload)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}

			sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
			if err != nil {
				t.Fatalf("GetSubscriptionByUserID: %v", err)
			}
			if sub.Status != tt.expectedStatus {
				t.Errorf("expected status=%s, got %s", tt.expectedStatus, sub.Status)
			}
		})
	}
}

// ── customer.subscription.deleted ───────────────────────────────────────────

func TestWebhook_SubscriptionDeleted_DowngradesToFree(t *testing.T) {
	t.Parallel()
	tx, _, mux, uid := setupWebhookTest(t)
	ctx := context.Background()

	subID := "sub_" + uid[:8]
	payload := stripeEventPayload(t, "evt_sub_deleted_1", "customer.subscription.deleted", map[string]interface{}{
		"id":     subID,
		"status": "canceled",
	})

	r := signedWebhookRequest(t, payload)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub.PlanID != db.PlanFree {
		t.Errorf("expected plan=%s, got %s", db.PlanFree, sub.PlanID)
	}
	if sub.Status != db.SubscriptionStatusCancelled {
		t.Errorf("expected status=cancelled, got %s", sub.Status)
	}
}

// ── Idempotency ─────────────────────────────────────────────────────────────

func TestWebhook_Idempotent_SkipsDuplicateEvents(t *testing.T) {
	t.Parallel()
	tx, _, mux, uid := setupWebhookTest(t)
	ctx := context.Background()

	subID := "sub_" + uid[:8]
	eventID := "evt_idempotent_1"

	payload := stripeEventPayload(t, eventID, "customer.subscription.deleted", map[string]interface{}{
		"id":     subID,
		"status": "canceled",
	})

	// First request: should process.
	r := signedWebhookRequest(t, payload)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w.Code)
	}

	// Verify event was recorded.
	already, err := db.IsStripeEventProcessed(ctx, tx, eventID)
	if err != nil {
		t.Fatalf("IsStripeEventProcessed: %v", err)
	}
	if !already {
		t.Error("expected event to be recorded after first processing")
	}

	// Reset the subscription to detect if it gets re-processed.
	if _, err := db.UpdateSubscriptionPlan(ctx, tx, uid, db.PlanPayAsYouGo); err != nil {
		t.Fatalf("UpdateSubscriptionPlan: %v", err)
	}
	if _, err := db.UpdateSubscriptionStatus(ctx, tx, uid, db.SubscriptionStatusActive); err != nil {
		t.Fatalf("UpdateSubscriptionStatus: %v", err)
	}

	// Second request with same event ID: should skip.
	r = signedWebhookRequest(t, payload)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("second request: expected 200, got %d", w.Code)
	}

	// Subscription should still be pay_as_you_go (not downgraded again).
	sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
	if err != nil {
		t.Fatalf("GetSubscriptionByUserID: %v", err)
	}
	if sub.PlanID != db.PlanPayAsYouGo {
		t.Errorf("expected plan=%s after idempotent skip, got %s", db.PlanPayAsYouGo, sub.PlanID)
	}
}

// ── Signature verification ──────────────────────────────────────────────────

func TestWebhook_InvalidSignature_Returns400(t *testing.T) {
	t.Parallel()
	_, _, mux, _ := setupWebhookTest(t)

	payload := []byte(`{"id": "evt_bad", "type": "invoice.paid"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", strings.NewReader(string(payload)))
	r.Header.Set("Stripe-Signature", "t=12345,v1=invalidsignature")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWebhook_MissingSignature_Returns400(t *testing.T) {
	t.Parallel()
	_, _, mux, _ := setupWebhookTest(t)

	payload := []byte(`{"id": "evt_nosig", "type": "invoice.paid"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", strings.NewReader(string(payload)))
	// No Stripe-Signature header.

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── No Stripe client ────────────────────────────────────────────────────────

func TestWebhook_NoStripeClient_Returns503(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, Stripe: nil}
	mux := http.NewServeMux()
	RegisterBillingWebhookRoutes(mux, deps)

	r := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// ── Unhandled event type ────────────────────────────────────────────────────

func TestWebhook_UnhandledEventType_Returns200(t *testing.T) {
	t.Parallel()
	_, _, mux, _ := setupWebhookTest(t)

	payload := stripeEventPayload(t, "evt_unknown_type", "some.unknown.event", map[string]interface{}{
		"id": "obj_123",
	})

	r := signedWebhookRequest(t, payload)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	// Unhandled events should return 200 (acknowledge receipt, don't retry).
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ── Payment failure flow (past_due → invoice.paid → active) ────────────────

func TestWebhook_PaymentFailureRecoveryFlow(t *testing.T) {
	t.Parallel()
	tx, _, mux, uid := setupWebhookTest(t)
	ctx := context.Background()

	subID := "sub_" + uid[:8]

	// Step 1: invoice.payment_failed → past_due.
	failPayload := stripeEventPayload(t, "evt_flow_fail", "invoice.payment_failed", map[string]interface{}{
		"id": "in_flow_fail",
		"parent": map[string]interface{}{
			"subscription_details": map[string]interface{}{
				"subscription": map[string]interface{}{"id": subID},
			},
		},
	})
	r := signedWebhookRequest(t, failPayload)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("payment_failed: expected 200, got %d", w.Code)
	}

	sub, _ := db.GetSubscriptionByUserID(ctx, tx, uid)
	if sub.Status != db.SubscriptionStatusPastDue {
		t.Fatalf("after payment_failed: expected past_due, got %s", sub.Status)
	}

	// Step 2: invoice.paid (Stripe retry succeeded) → active.
	paidPayload := stripeEventPayload(t, "evt_flow_paid", "invoice.paid", map[string]interface{}{
		"id":           "in_flow_paid",
		"period_start": time.Now().Unix(),
		"period_end":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		"parent": map[string]interface{}{
			"subscription_details": map[string]interface{}{
				"subscription": map[string]interface{}{"id": subID},
			},
		},
	})
	r = signedWebhookRequest(t, paidPayload)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("invoice.paid: expected 200, got %d", w.Code)
	}

	sub, _ = db.GetSubscriptionByUserID(ctx, tx, uid)
	if sub.Status != db.SubscriptionStatusActive {
		t.Errorf("after invoice.paid: expected active, got %s", sub.Status)
	}
	// Plan should still be pay_as_you_go (not downgraded).
	if sub.PlanID != db.PlanPayAsYouGo {
		t.Errorf("after invoice.paid: expected plan=%s, got %s", db.PlanPayAsYouGo, sub.PlanID)
	}
}

// ── Full cancellation flow ──────────────────────────────────────────────────

func TestWebhook_FullCancellationFlow(t *testing.T) {
	t.Parallel()
	tx, _, mux, uid := setupWebhookTest(t)
	ctx := context.Background()

	subID := "sub_" + uid[:8]

	// Step 1: invoice.payment_failed → past_due.
	failPayload := stripeEventPayload(t, "evt_cancel_fail", "invoice.payment_failed", map[string]interface{}{
		"id": "in_cancel_fail",
		"parent": map[string]interface{}{
			"subscription_details": map[string]interface{}{
				"subscription": map[string]interface{}{"id": subID},
			},
		},
	})
	r := signedWebhookRequest(t, failPayload)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("payment_failed: expected 200, got %d", w.Code)
	}

	// Step 2: customer.subscription.deleted → free + cancelled.
	deletePayload := stripeEventPayload(t, "evt_cancel_delete", "customer.subscription.deleted", map[string]interface{}{
		"id":     subID,
		"status": "canceled",
	})
	r = signedWebhookRequest(t, deletePayload)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("subscription.deleted: expected 200, got %d", w.Code)
	}

	sub, _ := db.GetSubscriptionByUserID(ctx, tx, uid)
	if sub.PlanID != db.PlanFree {
		t.Errorf("after cancellation: expected plan=%s, got %s", db.PlanFree, sub.PlanID)
	}
	if sub.Status != db.SubscriptionStatusCancelled {
		t.Errorf("after cancellation: expected status=cancelled, got %s", sub.Status)
	}
}
