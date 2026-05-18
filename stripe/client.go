// Package stripe provides a minimal Stripe API client for self-hosted card vault
// flows (SetupIntents and payment method retrieval). It intentionally excludes
// subscription billing and metered invoicing.
package stripe

import (
	"context"
	"fmt"
	"sync"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/paymentmethod"
	"github.com/stripe/stripe-go/v82/setupintent"
)

// Config holds Stripe API credentials.
type Config struct {
	SecretKey string // STRIPE_SECRET_KEY
}

// Client wraps Stripe calls used by the payment-methods API.
type Client struct {
	cfg Config
}

var keyMu sync.Mutex

// New configures the global Stripe API key and returns a client.
func New(cfg Config) *Client {
	keyMu.Lock()
	gostripe.Key = cfg.SecretKey
	keyMu.Unlock()
	return &Client{cfg: cfg}
}

// CreateCustomer creates a Stripe Customer for a Permission Slip user.
func (c *Client) CreateCustomer(ctx context.Context, email, userID string) (*gostripe.Customer, error) {
	params := &gostripe.CustomerParams{
		Metadata: map[string]string{"user_id": userID},
	}
	if email != "" {
		params.Email = gostripe.String(email)
	}
	params.Context = ctx
	cust, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: create customer: %w", err)
	}
	return cust, nil
}

// CreateSetupIntent creates a SetupIntent for adding a card without charging.
func (c *Client) CreateSetupIntent(ctx context.Context, stripeCustomerID string) (*gostripe.SetupIntent, error) {
	params := &gostripe.SetupIntentParams{
		Customer:           gostripe.String(stripeCustomerID),
		PaymentMethodTypes: []*string{gostripe.String("card")},
	}
	params.Context = ctx
	si, err := setupintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: create setup intent: %w", err)
	}
	return si, nil
}

// GetPaymentMethod retrieves a Stripe PaymentMethod by ID.
func (c *Client) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*gostripe.PaymentMethod, error) {
	params := &gostripe.PaymentMethodParams{}
	params.Context = ctx
	pm, err := paymentmethod.Get(paymentMethodID, params)
	if err != nil {
		return nil, fmt.Errorf("stripe: get payment method: %w", err)
	}
	return pm, nil
}

// DetachPaymentMethod detaches a payment method from its customer.
func (c *Client) DetachPaymentMethod(ctx context.Context, paymentMethodID string) (*gostripe.PaymentMethod, error) {
	params := &gostripe.PaymentMethodDetachParams{}
	params.Context = ctx
	pm, err := paymentmethod.Detach(paymentMethodID, params)
	if err != nil {
		return nil, fmt.Errorf("stripe: detach payment method: %w", err)
	}
	return pm, nil
}
