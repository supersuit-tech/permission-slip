package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateSubscriptionAction implements connectors.Action for stripe.update_subscription.
// It updates an existing subscription via POST /v1/subscriptions/{id}.
// Supports plan upgrades/downgrades, quantity changes, adding coupons,
// and trial management (extend or end trials early).
type updateSubscriptionAction struct {
	conn *StripeConnector
}

type updateSubscriptionItem struct {
	ID       string `json:"id"`
	Price    string `json:"price"`
	Quantity *int64 `json:"quantity"`
	Deleted  bool   `json:"deleted"`
}

type updateSubscriptionParams struct {
	SubscriptionID    string                   `json:"subscription_id"`
	Items             []updateSubscriptionItem `json:"items"`
	CouponID          string                   `json:"coupon"`
	ProrationBehavior string                   `json:"proration_behavior"`
	// TrialEnd extends or ends a trial. Accepts a Unix timestamp string (e.g.
	// "1893456000") to set a new trial end date, or the string "now" to end
	// the trial immediately and begin billing.
	TrialEnd string `json:"trial_end"`
	// CancelAt schedules the subscription to cancel at a future Unix timestamp.
	// Must be a positive integer — 0 or negative values are rejected.
	CancelAt *int64 `json:"cancel_at"`
	Metadata map[string]any `json:"metadata"`
}

func (p *updateSubscriptionParams) validate() error {
	if p.SubscriptionID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subscription_id"}
	}
	for i, item := range p.Items {
		if !item.Deleted && item.Price == "" && item.ID == "" {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("items[%d]: must provide id (to update existing item) or price (to add new item)", i),
			}
		}
		if item.Quantity != nil && *item.Quantity < 1 {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("items[%d].quantity must be at least 1", i),
			}
		}
	}
	if err := validateEnum(p.ProrationBehavior, "proration_behavior", []string{
		"create_prorations", "none", "always_invoice",
	}); err != nil {
		return err
	}
	if p.TrialEnd != "" && p.TrialEnd != "now" {
		// Use strconv.ParseInt rather than fmt.Sscanf: Sscanf stops at the first
		// non-numeric character and returns success (e.g. "123abc" parses as 123).
		// ParseInt requires the entire string to be a valid integer.
		ts, err := strconv.ParseInt(p.TrialEnd, 10, 64)
		if err != nil || ts <= 0 {
			return &connectors.ValidationError{
				Message: `trial_end must be a positive Unix timestamp string (e.g. "1893456000") or "now" to end the trial immediately`,
			}
		}
	}
	if p.CancelAt != nil && *p.CancelAt <= 0 {
		return &connectors.ValidationError{Message: "cancel_at must be a positive Unix timestamp (seconds since epoch)"}
	}
	return validateMetadata(p.Metadata)
}

// Execute updates a Stripe subscription and returns the updated subscription data.
func (a *updateSubscriptionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateSubscriptionParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{}

	if len(params.Items) > 0 {
		items := make([]any, len(params.Items))
		for i, item := range params.Items {
			entry := map[string]any{}
			if item.ID != "" {
				entry["id"] = item.ID
			}
			if item.Price != "" {
				entry["price"] = item.Price
			}
			if item.Quantity != nil {
				entry["quantity"] = *item.Quantity
			}
			if item.Deleted {
				entry["deleted"] = true
			}
			items[i] = entry
		}
		body["items"] = items
	}
	if params.CouponID != "" {
		body["coupon"] = params.CouponID
	}
	if params.ProrationBehavior != "" {
		body["proration_behavior"] = params.ProrationBehavior
	}
	if params.TrialEnd != "" {
		body["trial_end"] = params.TrialEnd
	}
	if params.CancelAt != nil && *params.CancelAt > 0 {
		body["cancel_at"] = *params.CancelAt
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)
	escapedID := url.PathEscape(params.SubscriptionID)

	var resp struct {
		ID                 string `json:"id"`
		Status             string `json:"status"`
		Customer           string `json:"customer"`
		CurrentPeriodStart int64  `json:"current_period_start"`
		CurrentPeriodEnd   int64  `json:"current_period_end"`
		TrialEnd           *int64 `json:"trial_end"`
		CancelAt           *int64 `json:"cancel_at"`
		LatestInvoice      string `json:"latest_invoice"`
		Created            int64  `json:"created"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/subscriptions/"+escapedID, formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
