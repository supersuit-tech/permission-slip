package stripe

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

// WebhookEvent is a parsed Stripe webhook event with typed data.
type WebhookEvent struct {
	ID   string // Stripe event ID (e.g. "evt_1234…"); useful for logging and idempotency
	Type string
	Raw  gostripe.Event
}

// VerifyWebhook reads the request body, verifies the Stripe webhook signature,
// and returns the parsed event. Returns an error if signature verification
// fails or the body cannot be read.
//
// maxBodyBytes limits the request body size to prevent abuse.
func VerifyWebhook(r *http.Request, webhookSecret string, maxBodyBytes int64) (*WebhookEvent, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("stripe webhook: read body: %w", err)
	}

	sig := r.Header.Get("Stripe-Signature")
	if sig == "" {
		return nil, fmt.Errorf("stripe webhook: missing Stripe-Signature header")
	}

	event, err := webhook.ConstructEvent(body, sig, webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("stripe webhook: verify signature: %w", err)
	}

	return &WebhookEvent{
		ID:   event.ID,
		Type: string(event.Type),
		Raw:  event,
	}, nil
}

// ParseCheckoutSessionCompleted extracts the checkout.session.completed event data.
// Returns the Stripe Customer ID and Subscription ID from the session.
func ParseCheckoutSessionCompleted(event *WebhookEvent) (customerID, subscriptionID string, err error) {
	var session gostripe.CheckoutSession
	if err := json.Unmarshal(event.Raw.Data.Raw, &session); err != nil {
		return "", "", fmt.Errorf("stripe webhook: parse checkout session: %w", err)
	}
	if session.Customer == nil {
		return "", "", fmt.Errorf("stripe webhook: checkout session missing customer")
	}
	if session.Subscription == nil {
		return "", "", fmt.Errorf("stripe webhook: checkout session missing subscription")
	}
	return session.Customer.ID, session.Subscription.ID, nil
}

// ParseSubscriptionUpdated extracts subscription data from
// customer.subscription.updated or customer.subscription.deleted events.
func ParseSubscriptionUpdated(event *WebhookEvent) (*gostripe.Subscription, error) {
	var sub gostripe.Subscription
	if err := json.Unmarshal(event.Raw.Data.Raw, &sub); err != nil {
		return nil, fmt.Errorf("stripe webhook: parse subscription: %w", err)
	}
	return &sub, nil
}

// ParseInvoicePaymentFailed extracts invoice data from invoice.payment_failed events.
func ParseInvoicePaymentFailed(event *WebhookEvent) (*gostripe.Invoice, error) {
	var inv gostripe.Invoice
	if err := json.Unmarshal(event.Raw.Data.Raw, &inv); err != nil {
		return nil, fmt.Errorf("stripe webhook: parse invoice: %w", err)
	}
	return &inv, nil
}
