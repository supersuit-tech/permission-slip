package paypal

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type createVenmoPayoutBatchAction struct {
	conn *PayPalConnector
}

type createVenmoPayoutBatchParams struct {
	PayoutBatch json.RawMessage `json:"payout_batch"`
}

// Execute creates a payout batch via POST /v1/payments/payouts.
// The payout_batch body must follow PayPal's schema; for Venmo recipients use
// recipient_type PHONE or USER_HANDLE and recipient_wallet VENMO in each item.
func (a *createVenmoPayoutBatchAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createVenmoPayoutBatchParams
	if err := parseParams(req, &params); err != nil {
		return nil, err
	}
	body, err := readJSONBody(params.PayoutBatch, "payout_batch")
	if err != nil {
		return nil, err
	}

	reqID := deriveRequestID(req.ActionType, req.Parameters)
	raw, err := a.conn.doJSONRaw(ctx, req.Credentials, http.MethodPost, "/v1/payments/payouts", body, reqID)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
