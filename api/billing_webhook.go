package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	gostripe "github.com/stripe/stripe-go/v82"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/notify"
	pstripe "github.com/supersuit-tech/permission-slip/stripe"
)

// maxWebhookBodyBytes limits the webhook request body to 64 KB.
// Stripe events are typically small; this prevents abuse.
const maxWebhookBodyBytes = 65536

// RegisterBillingWebhookRoutes adds the Stripe webhook endpoint to the mux.
// This should be called on the top-level mux (NOT inside the v1 router) so the
// endpoint bypasses auth and rate-limiting middleware. Stripe authenticates
// requests via the Stripe-Signature header, verified against STRIPE_WEBHOOK_SECRET.
func RegisterBillingWebhookRoutes(mux *http.ServeMux, deps *Deps) {
	mux.Handle("POST /api/webhooks/stripe", handleStripeWebhook(deps))
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
			CaptureError(r.Context(), err)
			http.Error(w, "webhook verification failed", http.StatusBadRequest)
			return
		}

		log.Printf("[%s] StripeWebhook: received event %s type=%s", TraceID(r.Context()), event.ID, event.Type)

		// Idempotency: skip events we've already processed. This prevents
		// duplicate processing when Stripe retries after a timeout.
		already, err := db.IsStripeEventProcessed(r.Context(), deps.DB, event.ID)
		if err != nil {
			log.Printf("[%s] StripeWebhook: idempotency check failed (event %s): %v", TraceID(r.Context()), event.ID, err)
			CaptureError(r.Context(), err)
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
		case "invoice.paid":
			handlerErr = handleInvoicePaid(r, deps, event)
		case "invoice.payment_failed":
			handlerErr = handleInvoicePaymentFailed(r, deps, event)
		case "customer.subscription.updated":
			handlerErr = handleSubscriptionUpdated(r, deps, event)
		case "customer.subscription.deleted":
			handlerErr = handleSubscriptionDeleted(r, deps, event)
		default:
			log.Printf("[%s] StripeWebhook: unhandled event type: %s (event %s)", TraceID(r.Context()), event.Type, event.ID)
		}

		if handlerErr != nil {
			log.Printf("[%s] StripeWebhook: handler failed for event %s: %v", TraceID(r.Context()), event.ID, handlerErr)
			CaptureError(r.Context(), handlerErr)
			// Return 500 so Stripe retries. Don't record the event.
			http.Error(w, "processing failed", http.StatusInternalServerError)
			return
		}

		// Record successful processing. If this fails, the worst case is that
		// Stripe retries and we reprocess idempotently.
		if _, err := db.RecordStripeEvent(r.Context(), deps.DB, event.ID, event.Type); err != nil {
			log.Printf("[%s] StripeWebhook: failed to record event %s: %v", TraceID(r.Context()), event.ID, err)
			CaptureError(r.Context(), err)
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
		CaptureError(r.Context(), err)
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
	upgraded, err := db.UpgradePayAsYouGoFromFreeOrFreePro(r.Context(), deps.DB, sub.UserID)
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

// lookupSubscriptionFromInvoice extracts the subscription ID from a Stripe
// invoice's parent object and looks up the local subscription. Returns an
// error for DB failures (retryable). Returns (nil, nil) when the invoice
// has no parent subscription or the subscription is unknown (not retryable).
//
// This is shared by handleInvoicePaid and handleInvoicePaymentFailed.
func lookupSubscriptionFromInvoice(r *http.Request, deps *Deps, event *pstripe.WebhookEvent, inv *gostripe.Invoice) (*db.Subscription, error) {
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		log.Printf("[%s] StripeWebhook: %s (event %s): invoice has no parent subscription, skipping", TraceID(r.Context()), event.Type, event.ID)
		return nil, nil
	}
	stripeSubID := inv.Parent.SubscriptionDetails.Subscription.ID

	sub, err := db.GetSubscriptionByStripeSubscriptionID(r.Context(), deps.DB, stripeSubID)
	if err != nil {
		return nil, fmt.Errorf("lookup subscription %s: %w", stripeSubID, err)
	}
	if sub == nil {
		log.Printf("[%s] StripeWebhook: %s (event %s): no subscription found for %s", TraceID(r.Context()), event.Type, event.ID, stripeSubID)
		return nil, nil
	}
	return sub, nil
}

// handleInvoicePaid processes a successful invoice payment. It updates the
// subscription status to active (clearing any past_due state from previous
// failed payments) and syncs the billing period dates from Stripe.
func handleInvoicePaid(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) error {
	inv, err := pstripe.ParseInvoicePaid(event)
	if err != nil {
		log.Printf("[%s] StripeWebhook: parse invoice.paid: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), err)
		return nil // malformed event, don't retry
	}

	sub, err := lookupSubscriptionFromInvoice(r, deps, event, inv)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil
	}

	// Update status to active (this clears past_due after a successful retry).
	if _, err := db.UpdateSubscriptionStatus(r.Context(), deps.DB, sub.UserID, db.SubscriptionStatusActive); err != nil {
		return fmt.Errorf("set active for %s: %w", sub.UserID, err)
	}

	// Sync billing period dates from the invoice.
	if inv.PeriodStart > 0 && inv.PeriodEnd > 0 {
		start := time.Unix(inv.PeriodStart, 0)
		end := time.Unix(inv.PeriodEnd, 0)
		if _, err := db.UpdateSubscriptionPeriod(r.Context(), deps.DB, sub.UserID, start, end); err != nil {
			return fmt.Errorf("update period for %s: %w", sub.UserID, err)
		}
	}

	log.Printf("[%s] StripeWebhook: invoice.paid (event %s), user %s status=active", TraceID(r.Context()), event.ID, sub.UserID)
	return nil
}

// lookupByStripeSubscription parses a subscription event and looks up the
// local subscription by Stripe subscription ID. Returns an error for DB
// failures (retryable). Returns nil subscription when not found (not retryable).
func lookupByStripeSubscription(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) (*db.Subscription, *pstripe.StripeSubscription, error) {
	stripeSub, err := pstripe.ParseSubscriptionEvent(event)
	if err != nil {
		log.Printf("[%s] StripeWebhook: parse %s (event %s): %v", TraceID(r.Context()), event.Type, event.ID, err)
		CaptureError(r.Context(), err)
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

	log.Printf("[%s] StripeWebhook: subscription.updated (event %s), user %s status=%s (stripe=%s)", TraceID(r.Context()), event.ID, sub.UserID, status, stripeSub.Status)
	return nil
}

// handleSubscriptionDeleted processes subscription cancellation.
// Downgrades the user back to the free plan.
func handleSubscriptionDeleted(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) error {
	sub, stripeSub, err := lookupByStripeSubscription(r, deps, event)
	if err != nil {
		return err
	}
	if sub == nil || stripeSub == nil {
		return nil
	}

	tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
	if err != nil {
		return fmt.Errorf("begin tx for subscription deleted: %w", err)
	}
	if owned {
		defer db.RollbackTx(r.Context(), tx)
	}

	// Re-read by Stripe subscription ID inside the transaction (same snapshot as cur).
	sub2, err := db.GetSubscriptionByStripeSubscriptionID(r.Context(), tx, stripeSub.ID)
	if err != nil {
		return fmt.Errorf("reload subscription by stripe id: %w", err)
	}
	if sub2 == nil {
		return nil
	}
	cur, err := db.GetSubscriptionByUserID(r.Context(), tx, sub2.UserID)
	if err != nil {
		return fmt.Errorf("reload subscription for %s: %w", sub2.UserID, err)
	}
	targetPlan := db.PlanFree
	if cur != nil && cur.PlanID == db.PlanFreePro {
		targetPlan = db.PlanFreePro
	}
	nextStatus := db.SubscriptionStatusCancelled
	if targetPlan == db.PlanFreePro {
		nextStatus = db.SubscriptionStatusActive
	}
	var quotaPlanID *string
	var quotaUntil *time.Time
	if targetPlan == db.PlanFree && cur != nil && cur.PlanID == db.PlanPayAsYouGo {
		qp := db.PlanPayAsYouGo
		quotaPlanID = &qp
		if u, ok := pstripe.SubscriptionCurrentPeriodEndUnix(event.Raw.Data.Raw); ok {
			t := time.Unix(u, 0)
			quotaUntil = &t
		} else {
			pe := cur.CurrentPeriodEnd
			quotaUntil = &pe
		}
	}
	if _, err := db.ApplyStripeSubscriptionDeletedToFree(r.Context(), tx, sub2.UserID, targetPlan, nextStatus, quotaPlanID, quotaUntil); err != nil {
		return fmt.Errorf("downgrade plan for %s: %w", sub2.UserID, err)
	}
	if owned {
		if err := db.CommitTx(r.Context(), tx); err != nil {
			return fmt.Errorf("commit subscription deleted: %w", err)
		}
	}

	log.Printf("[%s] StripeWebhook: subscription deleted (event %s), user %s plan=%s", TraceID(r.Context()), event.ID, sub2.UserID, targetPlan)
	return nil
}

// handleInvoicePaymentFailed marks the subscription as past_due when Stripe
// cannot collect payment, and notifies the user so they can update their
// payment method. Stripe will continue to retry per its Smart Retries schedule;
// if all retries fail, a customer.subscription.deleted event will follow.
func handleInvoicePaymentFailed(r *http.Request, deps *Deps, event *pstripe.WebhookEvent) error {
	inv, err := pstripe.ParseInvoicePaymentFailed(event)
	if err != nil {
		log.Printf("[%s] StripeWebhook: parse invoice.payment_failed: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), err)
		return nil // malformed event, don't retry
	}

	sub, err := lookupSubscriptionFromInvoice(r, deps, event, inv)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil
	}

	if _, err := db.UpdateSubscriptionStatus(r.Context(), deps.DB, sub.UserID, db.SubscriptionStatusPastDue); err != nil {
		return fmt.Errorf("set past_due for %s: %w", sub.UserID, err)
	}

	log.Printf("[%s] StripeWebhook: invoice.payment_failed (event %s), user %s set to past_due", TraceID(r.Context()), event.ID, sub.UserID)

	// Notify the user about the payment failure so they can update their
	// payment method. This is fire-and-forget; notification failures don't
	// affect the webhook response.
	notifyPaymentFailure(r, deps, sub.UserID, event)

	return nil
}

// notifyPaymentFailure sends a notification to the user about a failed payment.
// It is fire-and-forget: errors are logged, not returned, so the webhook handler
// is not blocked by delivery latency or notification failures.
func notifyPaymentFailure(r *http.Request, deps *Deps, userID string, event *pstripe.WebhookEvent) {
	if deps.Notifier == nil {
		log.Printf("[%s] StripeWebhook: notifier not configured, skipping payment failure notification for user %s", TraceID(r.Context()), userID)
		return
	}

	profile, err := db.GetProfileByUserID(r.Context(), deps.DB, userID)
	if err != nil {
		log.Printf("[%s] StripeWebhook: notify payment failure: profile lookup error for %s: %v", TraceID(r.Context()), userID, err)
		CaptureError(r.Context(), err)
		return
	}
	if profile == nil {
		log.Printf("[%s] StripeWebhook: notify payment failure: no profile found for user %s", TraceID(r.Context()), userID)
		CaptureError(r.Context(), fmt.Errorf("notify payment failure: no profile found for user %s", userID))
		return
	}

	// Build a billing settings URL so the user can update their payment method.
	billingURL := ""
	if deps.BaseURL != "" {
		billingURL = deps.BaseURL + "/settings?tab=billing"
	}

	notif := notify.Approval{
		ApprovalID:  "payment_failed_" + event.ID,
		ApprovalURL: billingURL,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(72 * time.Hour), // Stripe retries for 72h
		Type:        notify.NotificationTypePaymentFailed,
	}

	recipient := notify.Recipient{
		UserID:   profile.ID,
		Username: profile.Username,
		Email:    profile.Email,
		Phone:    profile.Phone,
	}

	deps.Notifier.Dispatch(r.Context(), notif, recipient)
	log.Printf("[%s] StripeWebhook: dispatched payment failure notification for user %s", TraceID(r.Context()), userID)
}
