package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listCustomersAction implements connectors.Action for square.list_customers.
// It lists customer profiles via GET /v2/customers (no query) or searches
// via POST /v2/customers/search (when a query string is provided).
type listCustomersAction struct {
	conn *SquareConnector
}

type listSquareCustomersParams struct {
	Query  string `json:"query,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (p *listSquareCustomersParams) validate() error {
	if p.Limit < 0 {
		return &connectors.ValidationError{Message: "limit must be a non-negative integer"}
	}
	if p.Limit > 100 {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be at most 100, got %d", p.Limit)}
	}
	return nil
}

// Execute lists or searches customer profiles from Square.
func (a *listCustomersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listSquareCustomersParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var path string
	var body interface{}

	if params.Query != "" {
		// Use the search endpoint when a query string is provided.
		path = "/customers/search"
		body = map[string]interface{}{
			"query": map[string]interface{}{
				"filter": map[string]interface{}{
					"email_address": map[string]interface{}{
						"fuzzy": params.Query,
					},
				},
			},
		}
		if params.Cursor != "" {
			reqMap := body.(map[string]interface{})
			reqMap["cursor"] = params.Cursor
		}
		if params.Limit > 0 {
			reqMap := body.(map[string]interface{})
			reqMap["limit"] = params.Limit
		}
	} else {
		// Use the list endpoint without filters.
		path = "/customers"
		if params.Cursor != "" || params.Limit > 0 {
			sep := "?"
			if params.Limit > 0 {
				path += fmt.Sprintf("%slimit=%d", sep, params.Limit)
				sep = "&"
			}
			if params.Cursor != "" {
				path += sep + "cursor=" + url.QueryEscape(params.Cursor)
			}
		}
	}

	var method string
	if params.Query != "" {
		method = http.MethodPost
	} else {
		method = http.MethodGet
		body = nil
	}

	var resp struct {
		Customers json.RawMessage `json:"customers"`
		Cursor    string          `json:"cursor,omitempty"`
	}
	if err := a.conn.do(ctx, req.Credentials, method, path, body, &resp); err != nil {
		return nil, err
	}

	customers := json.RawMessage(resp.Customers)
	if len(customers) == 0 || string(customers) == "null" {
		customers = json.RawMessage("[]")
	}

	result := map[string]interface{}{
		"customers": customers,
	}
	if resp.Cursor != "" {
		result["cursor"] = resp.Cursor
	}

	return connectors.JSONResult(result)
}
