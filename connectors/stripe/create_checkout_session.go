package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createCheckoutSessionAction implements connectors.Action for stripe.create_checkout_session.
// It creates a Stripe Checkout session via POST /v1/checkout/sessions.
// This is the most common Stripe integration for SaaS payment flows.
type createCheckoutSessionAction struct {
	conn *StripeConnector
}

type checkoutLineItem struct {
	Price    string `json:"price"`
	Quantity int64  `json:"quantity"`
}

type createCheckoutSessionParams struct {
	Mode           string             `json:"mode"`
	LineItems      []checkoutLineItem `json:"line_items"`
	SuccessURL     string             `json:"success_url"`
	CancelURL      string             `json:"cancel_url"`
	Customer       string             `json:"customer"`
	CustomerEmail  string             `json:"customer_email"`
	AllowPromoCode bool               `json:"allow_promotion_codes"`
	Metadata       map[string]any     `json:"metadata"`
}

// maxCheckoutLineItems mirrors the Stripe API limit of 20 line items per
// Checkout session. Enforcing it client-side gives a clearer error message
// than letting Stripe reject the request.
const maxCheckoutLineItems = 20

// validateCheckoutURL checks that a redirect URL is safe to use with Stripe Checkout.
// Stripe requires success_url and cancel_url to use https in production (except localhost
// for development). We enforce https broadly to prevent insecure data exposure in redirect
// parameters (e.g. session IDs passed via {CHECKOUT_SESSION_ID} template variable).
func validateCheckoutURL(fieldName, rawURL string) error {
	if rawURL == "" {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("%s is not a valid URL: %v", fieldName, err)}
	}
	if u.Scheme != "https" {
		return &connectors.ValidationError{Message: fmt.Sprintf("%s must use https scheme", fieldName)}
	}
	if u.Host == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("%s must include a host", fieldName)}
	}
	return nil
}

func (p *createCheckoutSessionParams) validate() error {
	if err := validateEnum(p.Mode, "mode", []string{"payment", "subscription", "setup"}); err != nil {
		return err
	}
	if p.Mode == "" {
		return &connectors.ValidationError{Message: "missing required parameter: mode"}
	}
	if len(p.LineItems) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: line_items"}
	}
	if len(p.LineItems) > maxCheckoutLineItems {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("too many line_items: %d (max %d)", len(p.LineItems), maxCheckoutLineItems),
		}
	}
	for i, item := range p.LineItems {
		if item.Price == "" {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("line_items[%d].price is required", i),
			}
		}
		if item.Quantity < 1 {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("line_items[%d].quantity must be at least 1", i),
			}
		}
	}
	if p.Customer != "" && p.CustomerEmail != "" {
		return &connectors.ValidationError{
			Message: "provide either customer or customer_email, not both",
		}
	}
	if err := validateCheckoutURL("success_url", p.SuccessURL); err != nil {
		return err
	}
	if err := validateCheckoutURL("cancel_url", p.CancelURL); err != nil {
		return err
	}
	return validateMetadata(p.Metadata)
}

// Execute creates a Stripe Checkout session and returns the session URL and ID.
func (a *createCheckoutSessionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCheckoutSessionParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"mode": params.Mode,
	}

	items := make([]any, len(params.LineItems))
	for i, item := range params.LineItems {
		items[i] = map[string]any{
			"price":    item.Price,
			"quantity": item.Quantity,
		}
	}
	body["line_items"] = items

	if params.SuccessURL != "" {
		body["success_url"] = params.SuccessURL
	}
	if params.CancelURL != "" {
		body["cancel_url"] = params.CancelURL
	}
	if params.Customer != "" {
		body["customer"] = params.Customer
	}
	if params.CustomerEmail != "" {
		body["customer_email"] = params.CustomerEmail
	}
	if params.AllowPromoCode {
		body["allow_promotion_codes"] = true
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)

	var resp struct {
		ID          string `json:"id"`
		URL         string `json:"url"`
		Status      string `json:"status"`
		Mode        string `json:"mode"`
		Customer    string `json:"customer"`
		AmountTotal *int64 `json:"amount_total"`
		Currency    string `json:"currency"`
		ExpiresAt   int64  `json:"expires_at"`
		Created     int64  `json:"created"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/checkout/sessions", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
