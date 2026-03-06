package stripe

import (
	"context"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getBalanceAction implements connectors.Action for stripe.get_balance.
// It retrieves the account balance via GET /v1/balance.
type getBalanceAction struct {
	conn *StripeConnector
}

func (a *getBalanceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var resp struct {
		Available []json.RawMessage `json:"available"`
		Pending   []json.RawMessage `json:"pending"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/v1/balance", nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
