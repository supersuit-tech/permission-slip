package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// issueRefundAction implements connectors.Action for square.issue_refund.
// It refunds a payment via POST /v2/refunds. High risk — returns real money
// and is irreversible.
type issueRefundAction struct {
	conn *SquareConnector
}

type issueRefundParams struct {
	PaymentID   string `json:"payment_id"`
	AmountMoney *money `json:"amount_money,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

func (p *issueRefundParams) validate() error {
	if p.PaymentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: payment_id"}
	}
	if p.AmountMoney != nil {
		if p.AmountMoney.Amount <= 0 {
			return &connectors.ValidationError{Message: "amount_money.amount must be greater than 0 (in smallest currency unit, e.g. cents)"}
		}
		if p.AmountMoney.Currency == "" {
			return &connectors.ValidationError{Message: "missing required parameter: amount_money.currency"}
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

	body := map[string]interface{}{
		"idempotency_key": newIdempotencyKey(),
		"payment_id":      params.PaymentID,
	}
	if params.AmountMoney != nil {
		body["amount_money"] = params.AmountMoney
	}
	if params.Reason != "" {
		body["reason"] = params.Reason
	}

	var resp struct {
		Refund json.RawMessage `json:"refund"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/refunds", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp.Refund))
}
