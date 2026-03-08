package stripe

import (
	"context"
	"encoding/json"
	"fmt"

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
	Mode              string             `json:"mode"`
	LineItems         []checkoutLineItem `json:"line_items"`
	SuccessURL        string             `json:"success_url"`
	CancelURL         string             `json:"cancel_url"`
	Customer          string             `json:"customer"`
	CustomerEmail     string             `json:"customer_email"`
	AllowPromoCode    bool               `json:"allow_promotion_codes"`
	Metadata          map[string]any     `json:"metadata"`
}

const maxCheckoutLineItems = 20

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
		ID         string `json:"id"`
		URL        string `json:"url"`
		Status     string `json:"status"`
		Mode       string `json:"mode"`
		Customer   string `json:"customer"`
		AmountTotal *int64 `json:"amount_total"`
		Currency   string `json:"currency"`
		Created    int64  `json:"created"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/checkout/sessions", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
