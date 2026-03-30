package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createPaymentLinkAction implements connectors.Action for stripe.create_payment_link.
// It creates a shareable payment link via POST /v1/payment_links.
type createPaymentLinkAction struct {
	conn *StripeConnector
}

type paymentLinkLineItem struct {
	PriceID  string `json:"price_id"`
	Quantity int64  `json:"quantity"`
}

type createPaymentLinkParams struct {
	LineItems           []paymentLinkLineItem `json:"line_items"`
	AfterCompletion     string                `json:"after_completion"`
	AllowPromotionCodes *bool                 `json:"allow_promotion_codes"`
	Metadata            map[string]any        `json:"metadata"`
}

// maxPaymentLinkLineItems caps the number of line items per payment link.
// Stripe allows up to 20 line items on a payment link.
const maxPaymentLinkLineItems = 20

func (p *createPaymentLinkParams) validate() error {
	if len(p.LineItems) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: line_items"}
	}
	if len(p.LineItems) > maxPaymentLinkLineItems {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("too many line items: %d (max %d)", len(p.LineItems), maxPaymentLinkLineItems),
		}
	}
	for i, item := range p.LineItems {
		if item.PriceID == "" {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("line_items[%d].price_id is required", i),
			}
		}
		if item.Quantity <= 0 {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("line_items[%d].quantity must be positive", i),
			}
		}
	}
	if p.AfterCompletion != "" {
		u, err := url.Parse(p.AfterCompletion)
		if err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("after_completion is not a valid URL: %v", err)}
		}
		if u.Scheme != "https" {
			return &connectors.ValidationError{Message: "after_completion must use https scheme"}
		}
		if u.Host == "" {
			return &connectors.ValidationError{Message: "after_completion must include a host"}
		}
	}
	return validateMetadata(p.Metadata)
}

// Execute creates a Stripe payment link and returns the link URL.
func (a *createPaymentLinkAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPaymentLinkParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{}

	// Build line_items array for bracket notation encoding.
	items := make([]any, len(params.LineItems))
	for i, item := range params.LineItems {
		items[i] = map[string]any{
			"price":    item.PriceID,
			"quantity": item.Quantity,
		}
	}
	body["line_items"] = items

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
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)

	var resp struct {
		ID     string `json:"id"`
		URL    string `json:"url"`
		Active bool   `json:"active"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/payment_links", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
