package expedia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// priceCheckAction implements connectors.Action for expedia.price_check.
// It confirms real-time pricing and availability for a room from search results
// via GET /v3/properties/availability/{room_id}/price-check.
type priceCheckAction struct {
	conn *ExpediaConnector
}

// priceCheckParams are the parameters parsed from ActionRequest.Parameters.
type priceCheckParams struct {
	RoomID string `json:"room_id"`
}

func (p *priceCheckParams) validate() error {
	if p.RoomID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: room_id"}
	}
	return nil
}

// Execute confirms pricing and availability for a specific room.
func (a *priceCheckAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params priceCheckParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v3/properties/availability/%s/price-check", url.PathEscape(params.RoomID))

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, defaultCustomerIP, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
