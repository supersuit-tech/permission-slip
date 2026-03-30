package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// searchOrdersAction implements connectors.Action for square.search_orders.
// It searches orders across locations via POST /v2/orders/search.
type searchOrdersAction struct {
	conn *SquareConnector
}

type searchOrdersParams struct {
	LocationIDs []string        `json:"location_ids"`
	Query       json.RawMessage `json:"query,omitempty"`
	Limit       int             `json:"limit,omitempty"`
	Cursor      string          `json:"cursor,omitempty"`
}

func (p *searchOrdersParams) validate() error {
	if len(p.LocationIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: location_ids (must have at least one)"}
	}
	if p.Limit < 0 || p.Limit > 500 {
		return &connectors.ValidationError{Message: "limit must be between 0 and 500 (0 or omitted uses Square's default)"}
	}
	if len(p.Query) > 0 {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(p.Query, &obj); err != nil {
			return &connectors.ValidationError{Message: "query must be a JSON object (e.g. {\"filter\": {\"state_filter\": {\"states\": [\"OPEN\"]}}})"}
		}
	}
	return nil
}

func (a *searchOrdersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchOrdersParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"location_ids": params.LocationIDs,
	}
	if len(params.Query) > 0 {
		body["query"] = json.RawMessage(params.Query)
	}
	if params.Limit > 0 {
		body["limit"] = params.Limit
	}
	if params.Cursor != "" {
		body["cursor"] = params.Cursor
	}

	var resp struct {
		Orders json.RawMessage `json:"orders"`
		Cursor string          `json:"cursor,omitempty"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/orders/search", body, &resp); err != nil {
		return nil, err
	}

	orders := json.RawMessage(resp.Orders)
	if len(orders) == 0 || string(orders) == "null" {
		orders = json.RawMessage("[]")
	}
	result := map[string]interface{}{
		"orders": orders,
	}
	if resp.Cursor != "" {
		result["cursor"] = resp.Cursor
	}

	return connectors.JSONResult(result)
}
