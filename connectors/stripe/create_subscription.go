package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createSubscriptionAction implements connectors.Action for stripe.create_subscription.
// It creates a recurring subscription via POST /v1/subscriptions.
type createSubscriptionAction struct {
	conn *StripeConnector
}

type subscriptionItem struct {
	Price    string `json:"price"`
	Quantity *int64 `json:"quantity"`
}

type createSubscriptionParams struct {
	Customer        string             `json:"customer"`
	Items           []subscriptionItem `json:"items"`
	TrialPeriodDays *int64             `json:"trial_period_days"`
	PaymentBehavior string             `json:"payment_behavior"`
	Metadata        map[string]any     `json:"metadata"`
}

// maxSubscriptionItems caps the number of items per subscription.
// Stripe allows up to 20 items on a subscription.
const maxSubscriptionItems = 20

func (p *createSubscriptionParams) validate() error {
	if p.Customer == "" {
		return &connectors.ValidationError{Message: "missing required parameter: customer"}
	}
	if len(p.Items) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: items"}
	}
	if len(p.Items) > maxSubscriptionItems {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("too many items: %d (max %d)", len(p.Items), maxSubscriptionItems),
		}
	}
	for i, item := range p.Items {
		if item.Price == "" {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("items[%d].price is required", i),
			}
		}
		if item.Quantity != nil && *item.Quantity <= 0 {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("items[%d].quantity must be positive", i),
			}
		}
	}
	if p.TrialPeriodDays != nil && *p.TrialPeriodDays < 0 {
		return &connectors.ValidationError{Message: "trial_period_days must be non-negative"}
	}
	if err := validateEnum(p.PaymentBehavior, "payment_behavior", []string{
		"default_incomplete", "error_if_incomplete", "allow_incomplete", "pending_if_incomplete",
	}); err != nil {
		return err
	}
	return validateMetadata(p.Metadata)
}

// Execute creates a Stripe subscription and returns the subscription data.
func (a *createSubscriptionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createSubscriptionParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"customer": params.Customer,
	}

	// Build items array for bracket notation encoding.
	items := make([]any, len(params.Items))
	for i, item := range params.Items {
		entry := map[string]any{"price": item.Price}
		if item.Quantity != nil {
			entry["quantity"] = *item.Quantity
		}
		items[i] = entry
	}
	body["items"] = items

	if params.TrialPeriodDays != nil {
		body["trial_period_days"] = *params.TrialPeriodDays
	}
	if params.PaymentBehavior != "" {
		body["payment_behavior"] = params.PaymentBehavior
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)

	var resp struct {
		ID                 string `json:"id"`
		Status             string `json:"status"`
		Customer           string `json:"customer"`
		CurrentPeriodStart int64  `json:"current_period_start"`
		CurrentPeriodEnd   int64  `json:"current_period_end"`
		TrialStart         *int64 `json:"trial_start"`
		TrialEnd           *int64 `json:"trial_end"`
		LatestInvoice      string `json:"latest_invoice"`
		Created            int64  `json:"created"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/subscriptions", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
