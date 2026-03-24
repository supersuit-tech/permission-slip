package paypal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type getOrderAction struct {
	conn *PayPalConnector
}

type getOrderParams struct {
	OrderID string `json:"order_id"`
}

func (a *getOrderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getOrderParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := validatePayPalPathID("order_id", params.OrderID); err != nil {
		return nil, err
	}
	path := "/v2/checkout/orders/" + params.OrderID
	var raw json.RawMessage
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &raw, ""); err != nil {
		return nil, err
	}
	return connectors.JSONResult(raw)
}
