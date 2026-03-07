package doordash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// cancelDeliveryAction implements connectors.Action for doordash.cancel_delivery.
// It cancels an active delivery via PUT /drive/v2/deliveries/{delivery_id}/cancel.
// May incur a cancellation fee depending on delivery status.
type cancelDeliveryAction struct {
	conn *DoorDashConnector
}

type cancelDeliveryParams struct {
	DeliveryID string `json:"delivery_id"`
}

func (p *cancelDeliveryParams) validate() error {
	if p.DeliveryID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: delivery_id"}
	}
	return nil
}

func (a *cancelDeliveryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params cancelDeliveryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, "/drive/v2/deliveries/"+params.DeliveryID+"/cancel", nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
