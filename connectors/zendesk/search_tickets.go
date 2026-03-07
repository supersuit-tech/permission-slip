package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchTicketsAction implements connectors.Action for zendesk.search_tickets.
// It searches tickets via GET /search.json?query=type:ticket {query}.
type searchTicketsAction struct {
	conn *ZendeskConnector
}

type searchTicketsParams struct {
	Query   string `json:"query"`
	SortBy  string `json:"sort_by"`
	SortDir string `json:"sort_order"`
}

var validSortBy = map[string]bool{
	"updated_at": true, "created_at": true, "priority": true, "status": true, "relevance": true,
}

var validSortDir = map[string]bool{
	"asc": true, "desc": true,
}

func (p *searchTicketsParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query"}
	}
	if p.SortBy != "" && !validSortBy[p.SortBy] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_by %q: must be updated_at, created_at, priority, status, or relevance", p.SortBy)}
	}
	if p.SortDir != "" && !validSortDir[p.SortDir] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_order %q: must be asc or desc", p.SortDir)}
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

	q := url.Values{}
	q.Set("query", "type:ticket "+params.Query)
	if params.SortBy != "" {
		q.Set("sort_by", params.SortBy)
	}
	if params.SortDir != "" {
		q.Set("sort_order", params.SortDir)
	}

	var resp searchResponse
	path := "/search.json?" + q.Encode()
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
