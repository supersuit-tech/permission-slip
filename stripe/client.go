package stripe

import (
	"context"
	"fmt"
	"sync"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/invoice"
	"github.com/stripe/stripe-go/v82/invoiceitem"
	"github.com/stripe/stripe-go/v82/paymentmethod"
	"github.com/stripe/stripe-go/v82/setupintent"
	"github.com/stripe/stripe-go/v82/subscription"
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

// keyMu protects writes to gostripe.Key so parallel tests don't race.
// In production New() is called once, but the mutex is cheap insurance.
var keyMu sync.Mutex

// New creates a new Stripe Client and sets the global API key.
// The Stripe Go SDK uses a global key by default; we set it once at init.
func New(cfg Config) *Client {
	keyMu.Lock()
	gostripe.Key = cfg.SecretKey
	keyMu.Unlock()
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

// OverageCostCents calculates the cost in cents for overage requests at
// $0.005/request (0.5 cents). Uses ceiling division to avoid under-billing
// for odd counts: ceil(overage * 0.5) = (overage*5 + 9) / 10.
// Returns 0 if overage is zero or negative.
func OverageCostCents(overage int) int {
	if overage <= 0 {
		return 0
	}
	return int((int64(overage)*5 + 9) / 10)
}

// CreateUsageInvoiceItem creates a Stripe Invoice Item for billable request
// usage. It calculates the overage beyond the free allowance and charges
// $0.005 per request. Returns nil (no error) if usage is within the free tier.
func (c *Client) CreateUsageInvoiceItem(ctx context.Context, stripeCustomerID string, requestCount int64) (*gostripe.InvoiceItem, error) {
	billable := requestCount - FreeRequestAllowance
	if billable <= 0 {
		return nil, nil // within free tier
	}

	amountCents := int64(OverageCostCents(int(billable)))

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

// InvoiceSummary is a simplified invoice representation for the API response.
type InvoiceSummary struct {
	ID          string `json:"id"`
	Number      string `json:"number,omitempty"`
	Status      string `json:"status"`
	Currency    string `json:"currency"`
	AmountDue   int64  `json:"amount_due"`
	AmountPaid  int64  `json:"amount_paid"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
	Created     int64  `json:"created"`
	HostedURL   string `json:"hosted_invoice_url,omitempty"`
	PDFURL      string `json:"invoice_pdf,omitempty"`
}

// ListInvoices returns up to `limit` invoices for the given Stripe customer,
// filtered to only paid invoices. Returns an empty slice if the customer has
// no invoices. hasMore is true when additional invoices exist beyond the limit.
func (c *Client) ListInvoices(ctx context.Context, stripeCustomerID string, limit int) (invoices []InvoiceSummary, hasMore bool, err error) {
	params := &gostripe.InvoiceListParams{
		Customer: gostripe.String(stripeCustomerID),
		Status:   gostripe.String("paid"),
	}
	// Fetch one extra to detect whether more pages exist.
	params.Limit = gostripe.Int64(int64(limit + 1))
	params.Context = ctx

	iter := invoice.List(params)
	for iter.Next() {
		if len(invoices) == limit {
			hasMore = true
			break
		}
		inv := iter.Invoice()
		summary := InvoiceSummary{
			ID:          inv.ID,
			Number:      inv.Number,
			Status:      string(inv.Status),
			Currency:    string(inv.Currency),
			AmountDue:   inv.AmountDue,
			AmountPaid:  inv.AmountPaid,
			PeriodStart: inv.PeriodStart,
			PeriodEnd:   inv.PeriodEnd,
			Created:     inv.Created,
		}
		if inv.HostedInvoiceURL != "" {
			summary.HostedURL = inv.HostedInvoiceURL
		}
		if inv.InvoicePDF != "" {
			summary.PDFURL = inv.InvoicePDF
		}
		invoices = append(invoices, summary)
	}
	if iterErr := iter.Err(); iterErr != nil {
		return nil, false, fmt.Errorf("stripe: list invoices: %w", iterErr)
	}
	return invoices, hasMore, nil
}

// ── Payment Method operations ─────────────────────────────────────────────

// CreateSetupIntent creates a Stripe SetupIntent for collecting a payment
// method without charging the customer. The client secret is used with
// Stripe Elements to securely collect card details.
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

// CancelSubscription cancels a Stripe subscription immediately.
func (c *Client) CancelSubscription(ctx context.Context, stripeSubscriptionID string) (*gostripe.Subscription, error) {
	params := &gostripe.SubscriptionCancelParams{}
	params.Context = ctx

	sub, err := subscription.Cancel(stripeSubscriptionID, params)
	if err != nil {
		return nil, fmt.Errorf("stripe: cancel subscription: %w", err)
	}
	return sub, nil
}
