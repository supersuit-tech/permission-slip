package paypal

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type getPayoutItemAction struct {
	conn *PayPalConnector
}

type getPayoutItemParams struct {
	PayoutItemID string `json:"payout_item_id"`
}

func (a *getPayoutItemAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getPayoutItemParams
	if err := parseParams(req, &params); err != nil {
		return nil, err
	}
	seg, err := pathSegment("payout_item_id", params.PayoutItemID)
	if err != nil {
		return nil, err
	}
	path := "/v1/payments/payouts-item/" + seg
	var raw json.RawMessage
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &raw, ""); err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
