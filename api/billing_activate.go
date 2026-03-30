package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip/db"
)

// ── POST /billing/activate ──────────────────────────────────────────────────

type activateRequest struct {
	SessionID string `json:"session_id"`
}

type activateResponse struct {
	PlanID string `json:"plan_id"`
	Status string `json:"status"`
}

// handleActivateUpgrade verifies a completed Stripe Checkout Session and
// activates the paid plan synchronously. This provides a reliable upgrade
// path that doesn't depend on webhooks, which may be delayed or
// misconfigured (especially in test/dev environments with ngrok).
//
// The webhook remains the primary upgrade mechanism; this endpoint acts as
// a fallback that the frontend calls after returning from Stripe checkout.
func handleActivateUpgrade(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req activateRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if req.SessionID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "session_id is required"))
			return
		}

		// Already on paid plan — nothing to do.
		sub, err := db.GetSubscriptionByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ActivateUpgrade: subscription lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}
		if sub != nil && sub.PlanID == db.PlanPayAsYouGo {
			RespondJSON(w, http.StatusOK, activateResponse{PlanID: sub.PlanID, Status: "already_active"})
			return
		}

		if deps.Stripe == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Billing not configured"))
			return
		}

		// Verify the checkout session with Stripe.
		sess, err := deps.Stripe.RetrieveCheckoutSession(r.Context(), req.SessionID)
		if err != nil {
			log.Printf("[%s] ActivateUpgrade: retrieve session: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to verify checkout session"))
			return
		}

		// Only activate if the session is complete and payment succeeded.
		if sess.Status != "complete" || sess.PaymentStatus != "paid" {
			pendingPlan := db.PlanFree
			if sub != nil && sub.PlanID == db.PlanFreePro {
				pendingPlan = db.PlanFreePro
			}
			RespondJSON(w, http.StatusOK, activateResponse{PlanID: pendingPlan, Status: "pending"})
			return
		}

		// Require a Stripe subscription — a session without one (e.g.,
		// payment mode or wrong product) shouldn't grant plan access.
		if sess.Subscription == nil {
			log.Printf("[%s] ActivateUpgrade: session %s has no subscription", TraceID(r.Context()), req.SessionID)
			CaptureError(r.Context(), fmt.Errorf("ActivateUpgrade: session %s has no subscription", req.SessionID))
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Checkout session has no subscription"))
			return
		}

		// Verify the session belongs to this user's Stripe customer.
		if sub == nil || sub.StripeCustomerID == nil || sess.Customer == nil || *sub.StripeCustomerID != sess.Customer.ID {
			log.Printf("[%s] ActivateUpgrade: customer mismatch for session %s", TraceID(r.Context()), req.SessionID)
			CaptureError(r.Context(), fmt.Errorf("ActivateUpgrade: customer mismatch for session %s", req.SessionID))
			RespondError(w, r, http.StatusForbidden, Forbidden(ErrInvalidRequest, "Checkout session does not belong to this account"))
			return
		}

		// Store the Stripe subscription ID.
		subID := sess.Subscription.ID
		custID := sess.Customer.ID
		if _, err := db.UpdateSubscriptionStripe(r.Context(), deps.DB, profile.ID, &custID, &subID); err != nil {
			log.Printf("[%s] ActivateUpgrade: save stripe IDs: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		}

		// Atomically upgrade — same logic as the webhook handler.
		upgraded, err := db.UpgradePayAsYouGoFromFreeOrFreePro(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ActivateUpgrade: upgrade plan: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to activate plan"))
			return
		}

		planID := db.PlanPayAsYouGo
		status := "activated"
		if upgraded == nil {
			// Already upgraded (e.g., webhook beat us to it) — still success.
			status = "already_active"
		}

		log.Printf("[%s] ActivateUpgrade: user %s plan=%s status=%s session=%s", TraceID(r.Context()), profile.ID, planID, status, req.SessionID)
		RespondJSON(w, http.StatusOK, activateResponse{PlanID: planID, Status: status})
	}
}
