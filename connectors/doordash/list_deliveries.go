package doordash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listDeliveriesAction implements connectors.Action for doordash.list_deliveries.
// It lists recent deliveries via GET /drive/v2/deliveries.
type listDeliveriesAction struct {
	conn *DoorDashConnector
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

	path := "/drive/v2/deliveries"
	sep := "?"
	if params.Limit != nil {
		path += sep + "limit=" + strconv.Itoa(*params.Limit)
		sep = "&"
	}
	if params.StartingAfter != "" {
		path += sep + "starting_after=" + params.StartingAfter
		sep = "&"
	}
	if params.Status != "" {
		path += sep + "status=" + params.Status
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
