package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// issueRefundAction implements connectors.Action for stripe.issue_refund.
// It issues a refund via POST /v1/refunds. This is a high-risk action
// that moves real money — the idempotency key is critical to prevent
// double-refunds on retries.
type issueRefundAction struct {
	conn *StripeConnector
}

type issueRefundParams struct {
	PaymentIntentID string         `json:"payment_intent_id"`
	ChargeID        string         `json:"charge_id"`
	Amount          any            `json:"amount"`
	Reason          string         `json:"reason"`
	Metadata        map[string]any `json:"metadata"`
}

func (p *issueRefundParams) validate() error {
	if p.PaymentIntentID == "" && p.ChargeID == "" {
		return &connectors.ValidationError{Message: "either payment_intent_id or charge_id is required"}
	}
	if p.Reason != "" {
		switch p.Reason {
		case "duplicate", "fraudulent", "requested_by_customer":
			// valid
		default:
			return &connectors.ValidationError{
				Message: fmt.Sprintf("invalid reason %q: must be one of duplicate, fraudulent, requested_by_customer", p.Reason),
			}
		}
	}
	return nil
}

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
		body["amount"] = params.Amount
	}
	if params.Reason != "" {
		body["reason"] = params.Reason
	}
	if len(params.Metadata) > 0 {
		body["metadata"] = params.Metadata
	}

	flat := formEncode(body)

	var resp struct {
		ID     string `json:"id"`
		Amount int    `json:"amount"`
		Status string `json:"status"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/refunds", flat, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
