package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createPriceAction implements connectors.Action for stripe.create_price.
// It creates a price for a product via POST /v1/prices.
// Prices can be one-time or recurring (required for subscriptions).
type createPriceAction struct {
	conn *StripeConnector
}

type createPriceRecurring struct {
	Interval      string `json:"interval"`
	IntervalCount *int64 `json:"interval_count"`
}

type createPriceParams struct {
	Currency    string                `json:"currency"`
	Product     string                `json:"product"`
	UnitAmount  *int64                `json:"unit_amount"`
	Recurring   *createPriceRecurring `json:"recurring"`
	Nickname    string                `json:"nickname"`
	Active      *bool                 `json:"active"`
	// TaxBehavior controls whether tax is included in the displayed price
	// (inclusive), added on top (exclusive), or unspecified (inherits from
	// the product's tax_behavior setting). Required for Stripe Tax.
	TaxBehavior string         `json:"tax_behavior"`
	Metadata    map[string]any `json:"metadata"`
}

func (p *createPriceParams) validate() error {
	if p.Currency == "" {
		return &connectors.ValidationError{Message: "missing required parameter: currency"}
	}
	if err := validateCurrency(p.Currency); err != nil {
		return err
	}
	if p.Product == "" {
		return &connectors.ValidationError{Message: "missing required parameter: product"}
	}
	if p.UnitAmount == nil {
		return &connectors.ValidationError{Message: "missing required parameter: unit_amount"}
	}
	if *p.UnitAmount < 0 {
		return &connectors.ValidationError{Message: "unit_amount must be non-negative"}
	}
	if p.Recurring != nil {
		if err := validateEnum(p.Recurring.Interval, "recurring.interval", []string{
			"day", "week", "month", "year",
		}); err != nil {
			return err
		}
		if p.Recurring.Interval == "" {
			return &connectors.ValidationError{Message: "recurring.interval is required when recurring is specified"}
		}
		if p.Recurring.IntervalCount != nil && *p.Recurring.IntervalCount < 1 {
			return &connectors.ValidationError{Message: "recurring.interval_count must be at least 1"}
		}
	}
	if err := validateEnum(p.TaxBehavior, "tax_behavior", []string{
		"inclusive", "exclusive", "unspecified",
	}); err != nil {
		return err
	}
	return validateMetadata(p.Metadata)
}

// Execute creates a Stripe price and returns the created price data.
func (a *createPriceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPriceParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"currency":    params.Currency,
		"product":     params.Product,
		"unit_amount": *params.UnitAmount,
	}
	if params.Recurring != nil {
		recurring := map[string]any{
			"interval": params.Recurring.Interval,
		}
		if params.Recurring.IntervalCount != nil {
			recurring["interval_count"] = *params.Recurring.IntervalCount
		}
		body["recurring"] = recurring
	}
	if params.Nickname != "" {
		body["nickname"] = params.Nickname
	}
	if params.Active != nil {
		body["active"] = *params.Active
	}
	if params.TaxBehavior != "" {
		body["tax_behavior"] = params.TaxBehavior
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)

	var resp struct {
		ID          string `json:"id"`
		Currency    string `json:"currency"`
		Product     string `json:"product"`
		UnitAmount  int64  `json:"unit_amount"`
		Active      bool   `json:"active"`
		Type        string `json:"type"`
		TaxBehavior string `json:"tax_behavior"`
		Created     int64  `json:"created"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/prices", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
