package stripe

import (
	"context"
	"fmt"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/invoiceitem"
)

// Config holds Stripe configuration from environment variables.
type Config struct {
	SecretKey      string // STRIPE_SECRET_KEY
	WebhookSecret  string // STRIPE_WEBHOOK_SECRET
	PriceIDRequest string // STRIPE_PRICE_ID_REQUEST — metered price for per-request billing
}

// Client wraps the Stripe API for billing operations.
// All methods are context-aware and return typed errors.
type Client struct {
	cfg Config
}

// New creates a new Stripe Client and sets the global API key.
// The Stripe Go SDK uses a global key by default; we set it once at init.
func New(cfg Config) *Client {
	gostripe.Key = cfg.SecretKey
	return &Client{cfg: cfg}
}

// WebhookSecret returns the webhook signing secret for signature verification.
func (c *Client) WebhookSecret() string {
	return c.cfg.WebhookSecret
}

// CreateCustomer creates a new Stripe Customer linked to a user.
func (c *Client) CreateCustomer(ctx context.Context, email, userID string) (*gostripe.Customer, error) {
	params := &gostripe.CustomerParams{
		Email: gostripe.String(email),
		Metadata: map[string]string{
			"user_id": userID,
		},
	}
	params.Context = ctx

	cust, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: create customer: %w", err)
	}
	return cust, nil
}

// CreateCheckoutSession creates a Stripe Checkout Session for upgrading to paid.
// The session collects payment method and creates a subscription with a metered price.
//
// successURL and cancelURL are the URLs the user is redirected to after checkout.
// They should include {CHECKOUT_SESSION_ID} placeholder for the session ID.
func (c *Client) CreateCheckoutSession(ctx context.Context, stripeCustomerID, successURL, cancelURL string) (*gostripe.CheckoutSession, error) {
	params := &gostripe.CheckoutSessionParams{
		Customer: gostripe.String(stripeCustomerID),
		Mode:     gostripe.String(string(gostripe.CheckoutSessionModeSubscription)),
		LineItems: []*gostripe.CheckoutSessionLineItemParams{
			{
				Price: gostripe.String(c.cfg.PriceIDRequest),
			},
		},
		SuccessURL: gostripe.String(successURL),
		CancelURL:  gostripe.String(cancelURL),
	}
	params.Context = ctx

	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: create checkout session: %w", err)
	}
	return sess, nil
}

// FreeRequestAllowance is the number of requests included for free each
// billing period before per-request charges apply.
const FreeRequestAllowance = 1000

// PricePerRequestCents is the per-request price in cents ($0.005 = 0.5 cents).
// We track this as a float to handle sub-cent pricing. Stripe Invoice Items
// accept amounts in the smallest currency unit (cents for USD).
const PricePerRequestCents = 0.5

// CreateUsageInvoiceItem creates a Stripe Invoice Item for billable request
// usage. It calculates the overage beyond the free allowance and charges
// $0.005 per request. Returns nil (no error) if usage is within the free tier.
func (c *Client) CreateUsageInvoiceItem(ctx context.Context, stripeCustomerID string, requestCount int64) (*gostripe.InvoiceItem, error) {
	billable := requestCount - FreeRequestAllowance
	if billable <= 0 {
		return nil, nil // within free tier
	}

	// $0.005/request = 0.5 cents/request. Stripe amounts are in cents (int64),
	// so we round up to avoid under-billing for odd counts.
	amountCents := (billable*5 + 9) / 10 // equivalent to ceil(billable * 0.5)

	params := &gostripe.InvoiceItemParams{
		Customer:    gostripe.String(stripeCustomerID),
		Amount:      gostripe.Int64(amountCents),
		Currency:    gostripe.String(string(gostripe.CurrencyUSD)),
		Description: gostripe.String(fmt.Sprintf("API requests: %d billable (after %d free)", billable, FreeRequestAllowance)),
	}
	params.Context = ctx

	item, err := invoiceitem.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: create usage invoice item: %w", err)
	}
	return item, nil
}

// SMSRate defines per-destination SMS pricing.
type SMSRate struct {
	Description string // e.g., "SMS - US/CA"
	UnitAmount  int64  // amount in cents
}

// SMSRates maps destination region identifiers to their pricing.
var SMSRates = map[string]SMSRate{
	"us_ca":         {Description: "SMS - US/CA", UnitAmount: 1},         // $0.01
	"uk_eu":         {Description: "SMS - UK/EU", UnitAmount: 4},         // $0.04
	"international": {Description: "SMS - International", UnitAmount: 5}, // $0.05 (base rate)
}

// CreateSMSInvoiceItem creates a Stripe Invoice Item for SMS charges.
// count is the number of SMS messages, region determines the per-SMS rate.
func (c *Client) CreateSMSInvoiceItem(ctx context.Context, stripeCustomerID, region string, count int64) (*gostripe.InvoiceItem, error) {
	rate, ok := SMSRates[region]
	if !ok {
		return nil, fmt.Errorf("stripe: unknown SMS region: %q", region)
	}

	params := &gostripe.InvoiceItemParams{
		Customer:    gostripe.String(stripeCustomerID),
		Amount:      gostripe.Int64(rate.UnitAmount * count),
		Currency:    gostripe.String(string(gostripe.CurrencyUSD)),
		Description: gostripe.String(fmt.Sprintf("%s (%d messages)", rate.Description, count)),
	}
	params.Context = ctx

	item, err := invoiceitem.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: create SMS invoice item: %w", err)
	}
	return item, nil
}
