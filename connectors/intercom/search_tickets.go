package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchTicketsAction implements connectors.Action for intercom.search_tickets.
// It searches tickets via POST /tickets/search.
type searchTicketsAction struct {
	conn *IntercomConnector
}

type searchTicketsParams struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

var validOperators = map[string]bool{
	"=": true, "!=": true, ">": true, "<": true, "~": true, "IN": true, "NIN": true,
}

func (p *searchTicketsParams) validate() error {
	if p.Field == "" {
		return &connectors.ValidationError{Message: "missing required parameter: field"}
	}
	if p.Operator == "" {
		return &connectors.ValidationError{Message: "missing required parameter: operator"}
	}
	if !validOperators[p.Operator] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid operator %q: must be =, !=, >, <, ~, IN, or NIN", p.Operator)}
	}
	if p.Value == "" {
		return &connectors.ValidationError{Message: "missing required parameter: value"}
	}
	return nil
}

func (a *searchTicketsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchTicketsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"query": map[string]any{
			"field":    params.Field,
			"operator": params.Operator,
			"value":    params.Value,
		},
	}

	var resp searchTicketsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/tickets/search", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
