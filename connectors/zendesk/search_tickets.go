package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// searchTicketsAction implements connectors.Action for zendesk.search_tickets.
// It searches tickets via GET /search.json?query=type:ticket {query}.
type searchTicketsAction struct {
	conn *ZendeskConnector
}

type searchTicketsParams struct {
	Query     string `json:"query"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
	Page      int    `json:"page"`
	PerPage   int    `json:"per_page"`
}

var validSortBy = map[string]bool{
	"updated_at": true, "created_at": true, "priority": true, "status": true, "relevance": true,
}

var validSortOrder = map[string]bool{
	"asc": true, "desc": true,
}

func (p *searchTicketsParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query"}
	}
	if p.SortBy != "" && !validSortBy[p.SortBy] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_by %q: must be updated_at, created_at, priority, status, or relevance", p.SortBy)}
	}
	if p.SortOrder != "" && !validSortOrder[p.SortOrder] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_order %q: must be asc or desc", p.SortOrder)}
	}
	if p.Page < 0 {
		return &connectors.ValidationError{Message: "page must be a positive integer"}
	}
	if p.PerPage < 0 || p.PerPage > 100 {
		return &connectors.ValidationError{Message: "per_page must be between 1 and 100"}
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
	if params.SortOrder != "" {
		q.Set("sort_order", params.SortOrder)
	}
	if params.Page > 0 {
		q.Set("page", strconv.Itoa(params.Page))
	}
	if params.PerPage > 0 {
		q.Set("per_page", strconv.Itoa(params.PerPage))
	}

	var resp searchResponse
	path := "/search.json?" + q.Encode()
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
