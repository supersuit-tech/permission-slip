package api

import (
	"fmt"
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

		// Idempotency: skip events we've already processed. This prevents
		// duplicate processing when Stripe retries after a timeout.
		already, err := db.IsStripeEventProcessed(r.Context(), deps.DB, event.ID)
		if err != nil {
			log.Printf("[%s] StripeWebhook: idempotency check failed (event %s): %v", TraceID(r.Context()), event.ID, err)
			// DB error — return 500 so Stripe retries.
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if already {
			log.Printf("[%s] StripeWebhook: event %s already processed, skipping", TraceID(r.Context()), event.ID)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Dispatch to the appropriate handler. Handlers return an error if
		// the event should be retried (DB failure), or nil on success (including
		// when the event is a no-op, e.g. unknown subscription).
		var handlerErr error
		switch event.Type {
		case "checkout.session.completed":
			handlerErr = handleCheckoutCompleted(r, deps, event)
		case "customer.subscription.updated":
			handlerErr = handleSubscriptionUpdated(r, deps, event)
		case "customer.subscription.deleted":
			handlerErr = handleSubscriptionDeleted(r, deps, event)
		case "invoice.payment_failed":
			handlerErr = handleInvoicePaymentFailed(r, deps, event)
		default:
			log.Printf("[%s] StripeWebhook: unhandled event type: %s (event %s)", TraceID(r.Context()), event.Type, event.ID)
		}

		if handlerErr != nil {
			log.Printf("[%s] StripeWebhook: handler failed for event %s: %v", TraceID(r.Context()), event.ID, handlerErr)
			// Return 500 so Stripe retries. Don't record the event.
			http.Error(w, "processing failed", http.StatusInternalServerError)
			return
		}

		// Record successful processing. If this fails, the worst case is that
		// Stripe retries and we reprocess idempotently.
		if _, err := db.RecordStripeEvent(r.Context(), deps.DB, event.ID, event.Type); err != nil {
			log.Printf("[%s] StripeWebhook: failed to record event %s: %v", TraceID(r.Context()), event.ID, err)
		}

		w.WriteHeader(http.StatusOK)
	}
}

// handleCheckoutCompleted processes a successful Checkout Session.
// It activates the paid subscription by updating the plan and storing
// Stripe IDs. Returns an error only for retryable failures (DB errors).
func handleCheckoutCompleted(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) error {
	customerID, subscriptionID, err := pstripe.ParseCheckoutSessionCompleted(event)
	if err != nil {
		log.Printf("[%s] StripeWebhook: parse checkout.session.completed: %v", TraceID(r.Context()), err)
		return nil // malformed event, don't retry
	}

	// Find the local subscription by Stripe customer ID.
	sub, err := db.GetSubscriptionByStripeCustomerID(r.Context(), deps.DB, customerID)
	if err != nil {
		return fmt.Errorf("lookup subscription by customer %s: %w", customerID, err)
	}
	if sub == nil {
		log.Printf("[%s] StripeWebhook: no subscription found for customer %s", TraceID(r.Context()), customerID)
		return nil // no matching user, don't retry
	}

	// Store Stripe subscription ID.
	if _, err := db.UpdateSubscriptionStripe(r.Context(), deps.DB, sub.UserID, &customerID, &subscriptionID); err != nil {
		return fmt.Errorf("save subscription IDs for %s: %w", sub.UserID, err)
	}

	// Atomically upgrade to paid plan — only succeeds if user is currently on
	// the free plan. This prevents double-upgrade race conditions when two
	// checkout webhooks arrive concurrently for the same user.
	upgraded, err := db.UpgradeSubscriptionPlan(r.Context(), deps.DB, sub.UserID, db.PlanFree, db.PlanPayAsYouGo)
	if err != nil {
		return fmt.Errorf("upgrade plan for %s: %w", sub.UserID, err)
	}
	if upgraded == nil {
		log.Printf("[%s] StripeWebhook: checkout complete (event %s), user %s already upgraded (idempotent no-op)", TraceID(r.Context()), event.ID, sub.UserID)
		return nil
	}

	log.Printf("[%s] StripeWebhook: checkout complete (event %s), user %s upgraded to pay_as_you_go", TraceID(r.Context()), event.ID, sub.UserID)
	return nil
}

// lookupByStripeSubscription parses a subscription event and looks up the
// local subscription by Stripe subscription ID. Returns an error for DB
// failures (retryable). Returns nil subscription when not found (not retryable).
func lookupByStripeSubscription(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) (*db.Subscription, *pstripe.StripeSubscription, error) {
	stripeSub, err := pstripe.ParseSubscriptionEvent(event)
	if err != nil {
		log.Printf("[%s] StripeWebhook: parse %s (event %s): %v", TraceID(r.Context()), event.Type, event.ID, err)
		return nil, nil, nil // malformed event, don't retry
	}

	sub, err := db.GetSubscriptionByStripeSubscriptionID(r.Context(), deps.DB, stripeSub.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup subscription %s: %w", stripeSub.ID, err)
	}
	if sub == nil {
		log.Printf("[%s] StripeWebhook: no subscription found for %s (event %s)", TraceID(r.Context()), stripeSub.ID, event.ID)
		return nil, nil, nil // unknown subscription, don't retry
	}
	return sub, stripeSub, nil
}

// handleSubscriptionUpdated processes subscription status changes.
// Maps Stripe statuses to local subscription statuses.
func handleSubscriptionUpdated(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) error {
	sub, stripeSub, err := lookupByStripeSubscription(r, deps, event)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil
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
		return fmt.Errorf("update status for %s: %w", sub.UserID, err)
	}
	return nil
}

// handleSubscriptionDeleted processes subscription cancellation.
// Downgrades the user back to the free plan.
func handleSubscriptionDeleted(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) error {
	sub, _, err := lookupByStripeSubscription(r, deps, event)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil
	}

	// Downgrade to free plan and mark cancelled.
	if _, err := db.UpdateSubscriptionPlan(r.Context(), deps.DB, sub.UserID, db.PlanFree); err != nil {
		return fmt.Errorf("downgrade plan for %s: %w", sub.UserID, err)
	}
	if _, err := db.UpdateSubscriptionStatus(r.Context(), deps.DB, sub.UserID, db.SubscriptionStatusCancelled); err != nil {
		return fmt.Errorf("cancel status for %s: %w", sub.UserID, err)
	}

	log.Printf("[%s] StripeWebhook: subscription deleted (event %s), user %s downgraded to free", TraceID(r.Context()), event.ID, sub.UserID)
	return nil
}

// handleInvoicePaymentFailed marks the subscription as past_due when Stripe
// cannot collect payment.
func handleInvoicePaymentFailed(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) error {
	inv, err := pstripe.ParseInvoicePaymentFailed(event)
	if err != nil {
		log.Printf("[%s] StripeWebhook: parse invoice.payment_failed: %v", TraceID(r.Context()), err)
		return nil // malformed event, don't retry
	}

	// Extract subscription ID from the parent object (v82+ API structure).
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		log.Printf("[%s] StripeWebhook: invoice.payment_failed (event %s): invoice has no parent subscription, skipping", TraceID(r.Context()), event.ID)
		return nil // not subscription-related, don't retry
	}
	stripeSubID := inv.Parent.SubscriptionDetails.Subscription.ID

	sub, err := db.GetSubscriptionByStripeSubscriptionID(r.Context(), deps.DB, stripeSubID)
	if err != nil {
		return fmt.Errorf("lookup subscription %s: %w", stripeSubID, err)
	}
	if sub == nil {
		log.Printf("[%s] StripeWebhook: invoice.payment_failed (event %s): no subscription found for %s", TraceID(r.Context()), event.ID, stripeSubID)
		return nil // unknown subscription, don't retry
	}

	if _, err := db.UpdateSubscriptionStatus(r.Context(), deps.DB, sub.UserID, db.SubscriptionStatusPastDue); err != nil {
		return fmt.Errorf("set past_due for %s: %w", sub.UserID, err)
	}
	return nil
}
