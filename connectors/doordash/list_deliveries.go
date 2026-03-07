package doordash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listDeliveriesAction implements connectors.Action for doordash.list_deliveries.
// It lists recent deliveries via GET /drive/v2/deliveries.
type listDeliveriesAction struct {
	conn *DoorDashConnector
}

// validStatuses is the set of delivery statuses accepted by the DoorDash API.
var validStatuses = map[string]bool{
	"created":              true,
	"confirmed":            true,
	"enroute_to_pickup":    true,
	"arrived_at_pickup":    true,
	"picked_up":            true,
	"enroute_to_dropoff":   true,
	"arrived_at_dropoff":   true,
	"delivered":            true,
	"cancelled":            true,
	"enroute_to_return":    true,
	"returned":             true,
}

type listDeliveriesParams struct {
	Limit         *int   `json:"limit,omitempty"`
	StartingAfter string `json:"starting_after,omitempty"`
	Status        string `json:"status,omitempty"`
}

func (p *listDeliveriesParams) validate() error {
	if p.Limit != nil && *p.Limit < 1 {
		return &connectors.ValidationError{Message: "limit must be at least 1"}
	}
	if p.Status != "" && !validStatuses[p.Status] {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid status %q — valid values: created, confirmed, enroute_to_pickup, arrived_at_pickup, picked_up, enroute_to_dropoff, arrived_at_dropoff, delivered, cancelled, enroute_to_return, returned", p.Status),
		}
	}
	return nil
}

func (a *listDeliveriesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listDeliveriesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := url.Values{}
	if params.Limit != nil {
		query.Set("limit", strconv.Itoa(*params.Limit))
	}
	if params.StartingAfter != "" {
		query.Set("starting_after", params.StartingAfter)
	}
	if params.Status != "" {
		query.Set("status", params.Status)
	}

	path := "/drive/v2/deliveries"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
