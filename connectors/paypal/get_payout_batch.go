package paypal

import (
	"context"
	"encoding/json"
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
	if err := parseParams(req, &params); err != nil {
		return nil, err
	}
	seg, err := pathSegment("payout_batch_id", params.PayoutBatchID)
	if err != nil {
		return nil, err
	}
	path := "/v1/payments/payouts/" + seg
	var raw json.RawMessage
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &raw, ""); err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
