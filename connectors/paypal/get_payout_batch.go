package paypal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type getPayoutBatchAction struct {
	conn *PayPalConnector
}

type getPayoutBatchParams struct {
	PayoutBatchID string `json:"payout_batch_id"`
}

func (a *getPayoutBatchAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getPayoutBatchParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := validatePayPalPathID("payout_batch_id", params.PayoutBatchID); err != nil {
		return nil, err
	}
	path := "/v1/payments/payouts/" + params.PayoutBatchID
	var raw json.RawMessage
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &raw, ""); err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
