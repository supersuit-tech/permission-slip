package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchContactsAction implements connectors.Action for intercom.search_contacts.
// It searches contacts via POST /contacts/search.
type searchContactsAction struct {
	conn *IntercomConnector
}

type searchContactsParams struct {
	Query intercomSearchQuery `json:"query"`
	Limit int                 `json:"limit"`
}

type intercomSearchQuery struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type contactsSearchResponse struct {
	Type       string           `json:"type"`
	TotalCount int              `json:"total_count"`
	Data       []intercomContact `json:"data"`
}

var validIntercomContactOperators = map[string]bool{
	"=":            true,
	"!=":           true,
	"IN":           true,
	"NIN":          true,
	">":            true,
	"<":            true,
	"~":            true,
	"!~":           true,
	"^":            true,
	"$":            true,
}

const (
	defaultContactSearchLimit = 20
	maxContactSearchLimit     = 150
)

func (p *searchContactsParams) validate() error {
	if p.Query.Field == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query.field"}
	}
	if p.Query.Operator == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query.operator"}
	}
	if !validIntercomContactOperators[p.Query.Operator] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid operator %q: must be =, !=, IN, NIN, >, <, ~, !~, ^, or $", p.Query.Operator)}
	}
	if p.Query.Value == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query.value"}
	}
	return nil
}

func (a *searchContactsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchContactsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = defaultContactSearchLimit
	}
	if limit > maxContactSearchLimit {
		limit = maxContactSearchLimit
	}

	body := map[string]any{
		"query": map[string]string{
			"field":    params.Query.Field,
			"operator": params.Query.Operator,
			"value":    params.Query.Value,
		},
		"pagination": map[string]int{
			"per_page": limit,
		},
	}

	var resp contactsSearchResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/contacts/search", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
