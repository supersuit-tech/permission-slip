package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createPaymentAction implements connectors.Action for square.create_payment.
// It processes a payment via POST /v2/payments. High risk — charges real money.
type createPaymentAction struct {
	conn *SquareConnector
}

type createPaymentParams struct {
	SourceID    string `json:"source_id"`
	AmountMoney money  `json:"amount_money"`
	OrderID     string `json:"order_id,omitempty"`
	CustomerID  string `json:"customer_id,omitempty"`
	Note        string `json:"note,omitempty"`
	ReferenceID string `json:"reference_id,omitempty"`
}

func (p *createPaymentParams) validate() error {
	if p.SourceID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: source_id"}
	}
	if p.AmountMoney.Currency == "" {
		return &connectors.ValidationError{Message: "missing required parameter: amount_money.currency"}
	}
	if p.AmountMoney.Amount <= 0 {
		return &connectors.ValidationError{Message: "amount_money.amount must be greater than 0 (in smallest currency unit, e.g. cents)"}
	}
	return nil
}

func (a *createPaymentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPaymentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"idempotency_key": newIdempotencyKey(),
		"source_id":       params.SourceID,
		"amount_money":    params.AmountMoney,
	}
	if params.OrderID != "" {
		body["order_id"] = params.OrderID
	}
	if params.CustomerID != "" {
		body["customer_id"] = params.CustomerID
	}
	if params.Note != "" {
		body["note"] = params.Note
	}
	if params.ReferenceID != "" {
		body["reference_id"] = params.ReferenceID
	}

	var resp struct {
		Payment json.RawMessage `json:"payment"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/payments", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp.Payment))
}
