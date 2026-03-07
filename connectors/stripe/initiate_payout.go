package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// initiatePayoutAction implements connectors.Action for stripe.initiate_payout.
// It triggers a payout to a connected bank account via POST /v1/payouts.
// This is the highest-risk action — it moves real money out of the Stripe account.
type initiatePayoutAction struct {
	conn *StripeConnector
}

type initiatePayoutParams struct {
	Amount      int64          `json:"amount"`
	Currency    string         `json:"currency"`
	Description string         `json:"description"`
	Destination string         `json:"destination"`
	Metadata    map[string]any `json:"metadata"`
}

func (p *initiatePayoutParams) validate() error {
	if p.Amount <= 0 {
		return &connectors.ValidationError{Message: "amount must be positive (in smallest currency unit)"}
	}
	if p.Currency == "" {
		return &connectors.ValidationError{Message: "missing required parameter: currency"}
	}
	if err := validateCurrency(p.Currency); err != nil {
		return err
	}
	return validateMetadata(p.Metadata)
}

// Execute initiates a Stripe payout and returns the payout data.
func (a *initiatePayoutAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params initiatePayoutParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"amount":   params.Amount,
		"currency": params.Currency,
	}
	if params.Description != "" {
		body["description"] = params.Description
	}
	if params.Destination != "" {
		body["destination"] = params.Destination
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	formParams := formEncode(body)

	var resp struct {
		ID          string `json:"id"`
		Amount      int64  `json:"amount"`
		Currency    string `json:"currency"`
		Status      string `json:"status"`
		Method      string `json:"method"`
		ArrivalDate int64  `json:"arrival_date"`
		Destination string `json:"destination"`
		Description string `json:"description"`
		Created     int64  `json:"created"`
	}

	if err := a.conn.doPost(ctx, req.Credentials, "/v1/payouts", formParams, &resp, req.ActionType, req.Parameters); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
