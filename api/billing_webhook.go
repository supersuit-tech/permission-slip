package api

import (
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/db"
	pstripe "github.com/supersuit-tech/permission-slip-web/stripe"
)

// maxWebhookBodyBytes limits the webhook request body to 64 KB.
// Stripe events are typically small; this prevents abuse.
const maxWebhookBodyBytes = 65536

// RegisterBillingWebhookRoutes adds the Stripe webhook endpoint to the mux.
// This endpoint does NOT require authentication — Stripe signs the request
// with a shared secret, which we verify via the Stripe-Signature header.
func RegisterBillingWebhookRoutes(mux *http.ServeMux, deps *Deps) {
	mux.Handle("POST /billing/webhook", handleStripeWebhook(deps))
}

func handleStripeWebhook(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Stripe == nil {
			http.Error(w, "billing not configured", http.StatusServiceUnavailable)
			return
		}

		event, err := pstripe.VerifyWebhook(r, deps.Stripe.WebhookSecret(), maxWebhookBodyBytes)
		if err != nil {
			log.Printf("[%s] StripeWebhook: verify: %v", TraceID(r.Context()), err)
			http.Error(w, "webhook verification failed", http.StatusBadRequest)
			return
		}

		log.Printf("[%s] StripeWebhook: received event %s type=%s", TraceID(r.Context()), event.ID, event.Type)

		switch event.Type {
		case "checkout.session.completed":
			handleCheckoutCompleted(r, deps, event)
		case "customer.subscription.updated":
			handleSubscriptionUpdated(r, deps, event)
		case "customer.subscription.deleted":
			handleSubscriptionDeleted(r, deps, event)
		case "invoice.payment_failed":
			handleInvoicePaymentFailed(r, deps, event)
		default:
			log.Printf("[%s] StripeWebhook: unhandled event type: %s (event %s)", TraceID(r.Context()), event.Type, event.ID)
		}

		// Always return 200 to acknowledge receipt. Returning non-200 causes
		// Stripe to retry, which would reprocess the event.
		w.WriteHeader(http.StatusOK)
	}
}

// handleCheckoutCompleted processes a successful Checkout Session.
// It activates the paid subscription by updating the plan and storing
// Stripe IDs.
func handleCheckoutCompleted(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) {
	customerID, subscriptionID, err := pstripe.ParseCheckoutSessionCompleted(event)
	if err != nil {
		log.Printf("[%s] StripeWebhook: parse checkout.session.completed: %v", TraceID(r.Context()), err)
		return
	}

	// Find the local subscription by Stripe customer ID.
	sub, err := db.GetSubscriptionByStripeCustomerID(r.Context(), deps.DB, customerID)
	if err != nil {
		log.Printf("[%s] StripeWebhook: lookup subscription by customer %s: %v", TraceID(r.Context()), customerID, err)
		return
	}
	if sub == nil {
		log.Printf("[%s] StripeWebhook: no subscription found for customer %s", TraceID(r.Context()), customerID)
		return
	}

	// Store Stripe subscription ID.
	if _, err := db.UpdateSubscriptionStripe(r.Context(), deps.DB, sub.UserID, &customerID, &subscriptionID); err != nil {
		log.Printf("[%s] StripeWebhook: save subscription IDs: %v", TraceID(r.Context()), err)
		return
	}

	// Atomically upgrade to paid plan — only succeeds if user is currently on
	// the free plan. This prevents double-upgrade race conditions when two
	// checkout webhooks arrive concurrently for the same user.
	upgraded, err := db.UpgradeSubscriptionPlan(r.Context(), deps.DB, sub.UserID, db.PlanFree, db.PlanPayAsYouGo)
	if err != nil {
		log.Printf("[%s] StripeWebhook: upgrade plan: %v", TraceID(r.Context()), err)
		return
	}
	if upgraded == nil {
		log.Printf("[%s] StripeWebhook: checkout complete (event %s), user %s already upgraded (idempotent no-op)", TraceID(r.Context()), event.ID, sub.UserID)
		return
	}

	log.Printf("[%s] StripeWebhook: checkout complete (event %s), user %s upgraded to pay_as_you_go", TraceID(r.Context()), event.ID, sub.UserID)
}

// lookupByStripeSubscription parses a subscription event and looks up the
// local subscription by Stripe subscription ID. Returns nil for both values
// (no error) if the subscription cannot be found — callers should return early.
func lookupByStripeSubscription(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) (*db.Subscription, *pstripe.StripeSubscription) {
	stripeSub, err := pstripe.ParseSubscriptionEvent(event)
	if err != nil {
		log.Printf("[%s] StripeWebhook: parse %s (event %s): %v", TraceID(r.Context()), event.Type, event.ID, err)
		return nil, nil
	}

	sub, err := db.GetSubscriptionByStripeSubscriptionID(r.Context(), deps.DB, stripeSub.ID)
	if err != nil {
		log.Printf("[%s] StripeWebhook: lookup subscription %s (event %s): %v", TraceID(r.Context()), stripeSub.ID, event.ID, err)
		return nil, nil
	}
	if sub == nil {
		log.Printf("[%s] StripeWebhook: no subscription found for %s (event %s)", TraceID(r.Context()), stripeSub.ID, event.ID)
		return nil, nil
	}
	return sub, stripeSub
}

// handleSubscriptionUpdated processes subscription status changes.
// Maps Stripe statuses to local subscription statuses.
func handleSubscriptionUpdated(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) {
	sub, stripeSub := lookupByStripeSubscription(r, deps, event)
	if sub == nil {
		return
	}

	// Map Stripe status → local status.
	var status db.SubscriptionStatus
	switch stripeSub.Status {
	case "active", "trialing":
		status = db.SubscriptionStatusActive
	case "past_due", "unpaid":
		status = db.SubscriptionStatusPastDue
	default:
		status = db.SubscriptionStatusCancelled
	}

	if _, err := db.UpdateSubscriptionStatus(r.Context(), deps.DB, sub.UserID, status); err != nil {
		log.Printf("[%s] StripeWebhook: update status for %s: %v", TraceID(r.Context()), sub.UserID, err)
	}
}

// handleSubscriptionDeleted processes subscription cancellation.
// Downgrades the user back to the free plan.
func handleSubscriptionDeleted(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) {
	sub, _ := lookupByStripeSubscription(r, deps, event)
	if sub == nil {
		return
	}

	// Downgrade to free plan and mark cancelled.
	if _, err := db.UpdateSubscriptionPlan(r.Context(), deps.DB, sub.UserID, db.PlanFree); err != nil {
		log.Printf("[%s] StripeWebhook: downgrade plan for %s: %v", TraceID(r.Context()), sub.UserID, err)
		return
	}
	if _, err := db.UpdateSubscriptionStatus(r.Context(), deps.DB, sub.UserID, db.SubscriptionStatusCancelled); err != nil {
		log.Printf("[%s] StripeWebhook: cancel status for %s: %v", TraceID(r.Context()), sub.UserID, err)
		return
	}

	log.Printf("[%s] StripeWebhook: subscription deleted (event %s), user %s downgraded to free", TraceID(r.Context()), event.ID, sub.UserID)
}

// handleInvoicePaymentFailed marks the subscription as past_due when Stripe
// cannot collect payment.
func handleInvoicePaymentFailed(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) {
	inv, err := pstripe.ParseInvoicePaymentFailed(event)
	if err != nil {
		log.Printf("[%s] StripeWebhook: parse invoice.payment_failed: %v", TraceID(r.Context()), err)
		return
	}

	// Extract subscription ID from the parent object (v82+ API structure).
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		log.Printf("[%s] StripeWebhook: invoice.payment_failed (event %s): invoice has no parent subscription, skipping", TraceID(r.Context()), event.ID)
		return
	}
	stripeSubID := inv.Parent.SubscriptionDetails.Subscription.ID

	sub, err := db.GetSubscriptionByStripeSubscriptionID(r.Context(), deps.DB, stripeSubID)
	if err != nil {
		log.Printf("[%s] StripeWebhook: invoice.payment_failed (event %s): lookup subscription %s: %v", TraceID(r.Context()), event.ID, stripeSubID, err)
		return
	}
	if sub == nil {
		log.Printf("[%s] StripeWebhook: invoice.payment_failed (event %s): no subscription found for %s", TraceID(r.Context()), event.ID, stripeSubID)
		return
	}

	if _, err := db.UpdateSubscriptionStatus(r.Context(), deps.DB, sub.UserID, db.SubscriptionStatusPastDue); err != nil {
		log.Printf("[%s] StripeWebhook: set past_due for %s: %v", TraceID(r.Context()), sub.UserID, err)
	}
}
