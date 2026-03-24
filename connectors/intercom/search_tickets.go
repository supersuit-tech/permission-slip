package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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
	// Optional bounds for ticket search — combined with the primary filter via AND.
	CreatedAtAfter  string `json:"created_at_after,omitempty"`
	CreatedAtBefore string `json:"created_at_before,omitempty"`
	UpdatedAtAfter  string `json:"updated_at_after,omitempty"`
	UpdatedAtBefore string `json:"updated_at_before,omitempty"`
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

	predicates := []map[string]any{
		{
			"field":    params.Field,
			"operator": params.Operator,
			"value":    params.Value,
		},
	}

	appendTimePredicate := func(field, op, raw string) error {
		if raw == "" {
			return nil
		}
		sec, err := connectors.ParseUnixTimestampOrRFC3339(raw)
		if err != nil {
			return err
		}
		v := strconv.FormatInt(sec, 10)
		if err != nil {
			return err
		}
		predicates = append(predicates, map[string]any{
			"field":    field,
			"operator": op,
			"value":    v,
		})
		return nil
	}
	if err := appendTimePredicate("created_at", ">", params.CreatedAtAfter); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid created_at_after: %v", err)}
	}
	if err := appendTimePredicate("created_at", "<", params.CreatedAtBefore); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid created_at_before: %v", err)}
	}
	if err := appendTimePredicate("updated_at", ">", params.UpdatedAtAfter); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid updated_at_after: %v", err)}
	}
	if err := appendTimePredicate("updated_at", "<", params.UpdatedAtBefore); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid updated_at_before: %v", err)}
	}

	var query any
	if len(predicates) == 1 {
		query = predicates[0]
	} else {
		query = map[string]any{
			"operator": "AND",
			"value":    predicates,
		}
	}

	body := map[string]any{"query": query}

	var resp searchTicketsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/tickets/search", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
