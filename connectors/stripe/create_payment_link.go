package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createPaymentLinkAction implements connectors.Action for stripe.create_payment_link.
// It creates a payment link via POST /v1/payment_links.
type createPaymentLinkAction struct {
	conn *StripeConnector
}

type paymentLinkLineItem struct {
	PriceID  string `json:"price_id"`
	Quantity any    `json:"quantity"`
}

type createPaymentLinkParams struct {
	LineItems           []paymentLinkLineItem `json:"line_items"`
	AfterCompletion     string                `json:"after_completion"`
	AllowPromotionCodes *bool                 `json:"allow_promotion_codes"`
	Metadata            map[string]any        `json:"metadata"`
}

func (p *createPaymentLinkParams) validate() error {
	if len(p.LineItems) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: line_items"}
	}
	for i, item := range p.LineItems {
		if item.PriceID == "" {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("line_items[%d]: missing required field: price_id", i),
			}
		}
	}
	return nil
}

func (a *createPaymentLinkAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPaymentLinkParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{}

	// Build line_items in the format Stripe expects: line_items[0][price]=..., line_items[0][quantity]=...
	// Must use []any (not []map[string]any) so formEncode's type switch handles arrays correctly.
	lineItems := make([]any, len(params.LineItems))
	for i, item := range params.LineItems {
		li := map[string]any{
			"price": item.PriceID,
		}
		if item.Quantity != nil {
			li["quantity"] = item.Quantity
		} else {
			li["quantity"] = "1"
		}
		lineItems[i] = li
	}
	body["line_items"] = lineItems

	if params.AfterCompletion != "" {
		body["after_completion"] = map[string]any{
			"type": "redirect",
			"redirect": map[string]any{
				"url": params.AfterCompletion,
			},
		}
	}
	if params.AllowPromotionCodes != nil {
		body["allow_promotion_codes"] = *params.AllowPromotionCodes
	}
	if len(params.Metadata) > 0 {
		body["metadata"] = params.Metadata
	}

	flat := formEncode(body)

	var resp struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/payment_links", flat, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
