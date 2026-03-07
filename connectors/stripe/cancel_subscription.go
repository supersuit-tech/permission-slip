package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// cancelSubscriptionAction implements connectors.Action for stripe.cancel_subscription.
// It cancels an active subscription via DELETE /v1/subscriptions/{id}.
// This is a high-risk action — revenue loss and customer impact.
type cancelSubscriptionAction struct {
	conn *StripeConnector
}

type cancelSubscriptionParams struct {
	SubscriptionID    string `json:"subscription_id"`
	CancelAtPeriodEnd *bool  `json:"cancel_at_period_end"`
	ProrationBehavior string `json:"proration_behavior"`
}

func (p *cancelSubscriptionParams) validate() error {
	if p.SubscriptionID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subscription_id"}
	}

	validProrations := map[string]bool{
		"create_prorations":    true,
		"none":                 true,
		"always_invoice":       true,
		"":                     true,
	}
	if !validProrations[p.ProrationBehavior] {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid proration_behavior %q: must be one of create_prorations, none, always_invoice", p.ProrationBehavior),
		}
	}
	return nil
}

// Execute cancels a Stripe subscription and returns the canceled subscription data.
func (a *cancelSubscriptionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params cancelSubscriptionParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// If cancel_at_period_end is true, we use POST to update the subscription
	// with cancel_at_period_end=true instead of DELETE (which cancels immediately).
	if params.CancelAtPeriodEnd != nil && *params.CancelAtPeriodEnd {
		body := map[string]any{
			"cancel_at_period_end": true,
		}
		if params.ProrationBehavior != "" {
			body["proration_behavior"] = params.ProrationBehavior
		}
		formParams := formEncode(body)

		var resp struct {
			ID                string `json:"id"`
			Status            string `json:"status"`
			CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
			CanceledAt        *int64 `json:"canceled_at"`
			CurrentPeriodEnd  int64  `json:"current_period_end"`
		}

		if err := a.conn.doPost(ctx, req.Credentials, "/v1/subscriptions/"+params.SubscriptionID, formParams, &resp, req.ActionType, req.Parameters); err != nil {
			return nil, err
		}
		return connectors.JSONResult(resp)
	}

	// Immediate cancellation via DELETE.
	formParams := map[string]string{}
	if params.ProrationBehavior != "" {
		formParams["proration_behavior"] = params.ProrationBehavior
	}

	var resp struct {
		ID                string `json:"id"`
		Status            string `json:"status"`
		CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
		CanceledAt        *int64 `json:"canceled_at"`
		CurrentPeriodEnd  int64  `json:"current_period_end"`
	}

	path := "/v1/subscriptions/" + params.SubscriptionID
	idempotencyKey := deriveIdempotencyKey(req.ActionType, req.Parameters)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, formParams, &resp, idempotencyKey); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
