package doordash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getDeliveryAction implements connectors.Action for doordash.get_delivery.
// It retrieves delivery status via GET /drive/v2/deliveries/{delivery_id}.
type getDeliveryAction struct {
	conn *DoorDashConnector
}

type getDeliveryParams struct {
	DeliveryID string `json:"delivery_id"`
}

func (p *getDeliveryParams) validate() error {
	if p.DeliveryID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: delivery_id"}
	}
	return nil
}

func (a *getDeliveryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getDeliveryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/drive/v2/deliveries/"+url.PathEscape(params.DeliveryID), nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
