package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// ── Response types ──────────────────────────────────────────────────────────

type subscriptionResponse struct {
	PlanID             string     `json:"plan_id"`
	PlanName           string     `json:"plan_name"`
	Status             string     `json:"status"`
	CurrentPeriodStart time.Time  `json:"current_period_start"`
	CurrentPeriodEnd   time.Time  `json:"current_period_end"`
	HasPaymentMethod   bool       `json:"has_payment_method"`
	Usage              *usageInfo `json:"usage,omitempty"`
}

type usageInfo struct {
	RequestCount int  `json:"request_count"`
	SMSCount     int  `json:"sms_count"`
	RequestLimit *int `json:"request_limit"`
	OverLimit    bool `json:"over_limit"`
}

type checkoutResponse struct {
	URL string `json:"url"`
}

// ── Route registration ──────────────────────────────────────────────────────

// RegisterBillingRoutes adds billing endpoints to the mux.
// These are only registered when billing is enabled.
func RegisterBillingRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /billing/subscription", requireProfile(handleGetSubscription(deps)))
	mux.Handle("POST /billing/checkout", requireProfile(handleCreateCheckout(deps)))
}

// ── GET /billing/subscription ───────────────────────────────────────────────

func handleGetSubscription(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		sub, err := db.GetSubscriptionWithPlan(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] GetSubscription: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}
		if sub == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrSubscriptionNotFound, "No subscription found"))
			return
		}

		resp := subscriptionResponse{
			PlanID:             sub.PlanID,
			PlanName:           sub.Plan.Name,
			Status:             string(sub.Status),
			CurrentPeriodStart: sub.CurrentPeriodStart,
			CurrentPeriodEnd:   sub.CurrentPeriodEnd,
			HasPaymentMethod:   sub.StripeCustomerID != nil,
		}

		// Attach current period usage if available.
		usage, err := db.GetCurrentPeriodUsage(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] GetSubscription: usage lookup: %v", TraceID(r.Context()), err)
		}
		if usage != nil {
			ui := &usageInfo{
				RequestCount: usage.RequestCount,
				SMSCount:     usage.SMSCount,
				RequestLimit: sub.Plan.MaxRequestsPerMonth,
			}
			if sub.Plan.MaxRequestsPerMonth != nil && usage.RequestCount > *sub.Plan.MaxRequestsPerMonth {
				ui.OverLimit = true
			}
			resp.Usage = ui
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// ── POST /billing/checkout ──────────────────────────────────────────────────

func handleCreateCheckout(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		if deps.Stripe == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Billing not configured"))
			return
		}

		// Look up existing subscription to check for existing Stripe customer.
		sub, err := db.GetSubscriptionByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] CreateCheckout: subscription lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}

		// If already on paid plan, reject.
		if sub != nil && sub.PlanID == db.PlanPayAsYouGo {
			RespondError(w, r, http.StatusConflict, Conflict(ErrAlreadySubscribed, "Already subscribed to a paid plan"))
			return
		}

		var stripeCustomerID string

		if sub != nil && sub.StripeCustomerID != nil {
			// Reuse existing Stripe customer.
			stripeCustomerID = *sub.StripeCustomerID
		} else {
			// Create a new Stripe customer.
			email := ""
			if profile.Email != nil {
				email = *profile.Email
			}

			cust, err := deps.Stripe.CreateCustomer(r.Context(), email, profile.ID)
			if err != nil {
				log.Printf("[%s] CreateCheckout: create customer: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to create Stripe customer"))
				return
			}
			stripeCustomerID = cust.ID

			// Persist Stripe customer ID.
			if _, err := db.UpdateSubscriptionStripe(r.Context(), deps.DB, profile.ID, &stripeCustomerID, nil); err != nil {
				log.Printf("[%s] CreateCheckout: save customer ID: %v", TraceID(r.Context()), err)
			}
		}

		// Build success/cancel URLs.
		successURL := deps.BaseURL + "/settings?checkout=success"
		cancelURL := deps.BaseURL + "/settings?checkout=cancel"

		sess, err := deps.Stripe.CreateCheckoutSession(r.Context(), stripeCustomerID, successURL, cancelURL)
		if err != nil {
			log.Printf("[%s] CreateCheckout: create session: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to create checkout session"))
			return
		}

		RespondJSON(w, http.StatusOK, checkoutResponse{URL: sess.URL})
	}
}

// upstreamError returns a 502 ErrorResponse for upstream Stripe failures.
func upstreamError(message string) ErrorResponse {
	return newErrorResponse(ErrUpstreamError, message, true)
}

// ReportPeriodUsage creates Stripe Invoice Items for a user's billing period
// usage (requests + SMS). Called by the webhook handler when a billing period
// ends or by a background cron.
func ReportPeriodUsage(ctx context.Context, deps *Deps, userID string, usage *db.UsagePeriod) {
	if deps.Stripe == nil {
		return
	}

	sub, err := db.GetSubscriptionByUserID(ctx, deps.DB, userID)
	if err != nil || sub == nil || sub.StripeCustomerID == nil {
		return
	}

	stripeCustomerID := *sub.StripeCustomerID

	// Report request usage.
	if usage.RequestCount > 0 {
		if _, err := deps.Stripe.CreateUsageInvoiceItem(ctx, stripeCustomerID, int64(usage.RequestCount)); err != nil {
			log.Printf("billing: report request usage for %s: %v", userID, err)
		}
	}

	// Report SMS usage. For now, we bill all SMS at the base "us_ca" rate.
	// A future iteration can track per-destination counts in the breakdown.
	if usage.SMSCount > 0 {
		if _, err := deps.Stripe.CreateSMSInvoiceItem(ctx, stripeCustomerID, "us_ca", int64(usage.SMSCount)); err != nil {
			log.Printf("billing: report SMS usage for %s: %v", userID, err)
		}
	}
}
