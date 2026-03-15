package stripe

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/invoice"
	"github.com/stripe/stripe-go/v82/invoiceitem"
	"github.com/stripe/stripe-go/v82/paymentmethod"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/setupintent"
	"github.com/stripe/stripe-go/v82/subscription"

	"github.com/supersuit-tech/permission-slip-web/config"
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

	// requestPrice caches the per-request price fetched from Stripe at init.
	// Zero means Stripe was unavailable; callers should fall back to plans.json.
	requestPriceMu sync.RWMutex
	requestPrice   *RequestPrice
}

// RequestPrice holds the per-request pricing fetched from Stripe.
type RequestPrice struct {
	// UnitAmountDecimal is the price in cents (e.g. 0.5 for $0.005).
	UnitAmountDecimal float64
	// DisplayAmount is a human-readable string like "$0.005".
	DisplayAmount string
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
	c := &Client{cfg: cfg}
	cachedClientMu.Lock()
	cachedClient = c
	cachedClientMu.Unlock()
	return c
}

// WebhookSecret returns the webhook signing secret for signature verification.
func (c *Client) WebhookSecret() string {
	return c.cfg.WebhookSecret
}

// FetchRequestPrice fetches the per-request price from Stripe and caches it.
// Should be called once at startup. Uses a 10-second timeout to avoid blocking
// startup if Stripe is unreachable. If Stripe is unavailable, the cached value
// stays nil and GetRequestPrice returns the fallback from plans.json.
func (c *Client) FetchRequestPrice() {
	if c.cfg.PriceIDRequest == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	params := &gostripe.PriceParams{}
	params.Context = ctx
	p, err := price.Get(c.cfg.PriceIDRequest, params)
	if err != nil {
		log.Printf("stripe: failed to fetch request price %s: %v", c.cfg.PriceIDRequest, err)
		return
	}

	// Guard against zero/null unit_amount_decimal (e.g. tiered pricing).
	if p.UnitAmountDecimal <= 0 {
		log.Printf("stripe: request price %s has zero/null unit_amount_decimal — using fallback", c.cfg.PriceIDRequest)
		return
	}

	rp := &RequestPrice{
		UnitAmountDecimal: p.UnitAmountDecimal,
	}
	// Format as "$X.XXX" from cents decimal (e.g. 0.5 cents = $0.005).
	dollars := p.UnitAmountDecimal / 100.0
	rp.DisplayAmount = fmt.Sprintf("$%.3f", dollars)

	c.requestPriceMu.Lock()
	c.requestPrice = rp
	c.requestPriceMu.Unlock()
}

// GetRequestPrice returns the cached per-request price from Stripe.
// Returns nil if Stripe was unavailable; callers should fall back to plans.json.
func (c *Client) GetRequestPrice() *RequestPrice {
	c.requestPriceMu.RLock()
	defer c.requestPriceMu.RUnlock()
	return c.requestPrice
}

// RequestPriceDisplay returns the display string for the per-request price.
// Falls back to plans.json if Stripe price is not cached.
func (c *Client) RequestPriceDisplay() string {
	if rp := c.GetRequestPrice(); rp != nil {
		return rp.DisplayAmount
	}
	return fallbackPriceDisplay()
}

// cachedClient holds a reference to the last-created Client for the package-level
// RequestPriceDisplay function. Set by New().
var (
	cachedClientMu sync.RWMutex
	cachedClient   *Client
)

// RequestPriceDisplay is a package-level function that returns the display string
// for the per-request price. Uses the cached Stripe price if available, otherwise
// falls back to plans.json. Safe to call even when no Stripe client is configured.
func RequestPriceDisplay() string {
	cachedClientMu.RLock()
	c := cachedClient
	cachedClientMu.RUnlock()
	if c != nil {
		return c.RequestPriceDisplay()
	}
	return fallbackPriceDisplay()
}

// fallbackPriceDisplay derives the price display from plans.json.
func fallbackPriceDisplay() string {
	p := config.GetPlan(config.PlanPayAsYouGo)
	if p != nil && p.PricePerRequestMillicents > 0 {
		dollars := float64(p.PricePerRequestMillicents) / 100_000.0
		return fmt.Sprintf("$%.3f", dollars)
	}
	return "$0.005"
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

// freeRequestAllowance is the number of requests included for free each
// billing period before per-request charges apply. Derived from config/plans.json.
var freeRequestAllowance = func() int64 {
	p := config.GetPlan(config.PlanFree)
	if p != nil && p.MaxRequestsPerMonth != nil {
		return int64(*p.MaxRequestsPerMonth)
	}
	return 250 // fallback
}()

// FreeRequestAllowance returns the number of free requests per billing period.
func FreeRequestAllowance() int64 { return freeRequestAllowance }

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
	allowance := FreeRequestAllowance()
	billable := requestCount - allowance
	if billable <= 0 {
		return nil, nil // within free tier
	}

	amountCents := int64(OverageCostCents(int(billable)))

	params := &gostripe.InvoiceItemParams{
		Customer:    gostripe.String(stripeCustomerID),
		Amount:      gostripe.Int64(amountCents),
		Currency:    gostripe.String(string(gostripe.CurrencyUSD)),
		Description: gostripe.String(fmt.Sprintf("API requests: %d billable (after %d free)", billable, allowance)),
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

// RetrieveCheckoutSession fetches an existing Checkout Session by ID.
// Used to verify payment status when the webhook hasn't arrived yet.
func (c *Client) RetrieveCheckoutSession(ctx context.Context, sessionID string) (*gostripe.CheckoutSession, error) {
	params := &gostripe.CheckoutSessionParams{}
	params.Context = ctx

	sess, err := session.Get(sessionID, params)
	if err != nil {
		return nil, fmt.Errorf("stripe: retrieve checkout session: %w", err)
	}
	return sess, nil
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
