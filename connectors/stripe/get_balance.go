package stripe

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getBalanceAction implements connectors.Action for stripe.get_balance.
// It retrieves the current account balance via GET /v1/balance.
type getBalanceAction struct {
	conn *StripeConnector
}

// Execute retrieves the Stripe account balance.
func (a *getBalanceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var resp struct {
		Available []struct {
			Amount   int64  `json:"amount"`
			Currency string `json:"currency"`
		} `json:"available"`
		Pending []struct {
			Amount   int64  `json:"amount"`
			Currency string `json:"currency"`
		} `json:"pending"`
		Livemode bool   `json:"livemode"`
		Object   string `json:"object"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/v1/balance", nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
