package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// issueRefundAction implements connectors.Action for stripe.issue_refund.
// It creates a refund via POST /v1/refunds. This is a high-risk action
// that moves real money — idempotency keys are mandatory to prevent
// double-refunds.
type issueRefundAction struct {
	conn *StripeConnector
}

type issueRefundParams struct {
	PaymentIntentID string         `json:"payment_intent_id"`
	ChargeID        string         `json:"charge_id"`
	Amount          *int64         `json:"amount"`
	Reason          string         `json:"reason"`
	Metadata        map[string]any `json:"metadata"`
}

func (p *issueRefundParams) validate() error {
	if p.PaymentIntentID == "" && p.ChargeID == "" {
		return &connectors.ValidationError{
			Message: "either payment_intent_id or charge_id is required",
		}
	}
	if p.PaymentIntentID != "" && p.ChargeID != "" {
		return &connectors.ValidationError{
			Message: "provide either payment_intent_id or charge_id, not both",
		}
	}
	if p.Amount != nil && *p.Amount <= 0 {
		return &connectors.ValidationError{
			Message: "amount must be positive (in cents)",
		}
	}

	validReasons := map[string]bool{
		"duplicate": true, "fraudulent": true,
		"requested_by_customer": true, "": true,
	}
	if !validReasons[p.Reason] {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid reason %q: must be one of duplicate, fraudulent, requested_by_customer", p.Reason),
		}
	}
	return nil
}

// Execute issues a Stripe refund and returns the refund data.
func (a *issueRefundAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params issueRefundParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{}
	if params.PaymentIntentID != "" {
		body["payment_intent"] = params.PaymentIntentID
	}
	if params.ChargeID != "" {
		body["charge"] = params.ChargeID
	}
	if params.Amount != nil {
		body["amount"] = *params.Amount
	}
	if params.Reason != "" {
		body["reason"] = params.Reason
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)

	var resp struct {
		ID     string `json:"id"`
		Amount int64  `json:"amount"`
		Status string `json:"status"`
		Reason string `json:"reason"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/refunds", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
