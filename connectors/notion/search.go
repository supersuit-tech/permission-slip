package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchAction implements connectors.Action for notion.search.
// It performs full-text search across shared pages and databases via POST /v1/search.
type searchAction struct {
	conn *NotionConnector
}

// searchParams is the user-facing parameter schema.
type searchParams struct {
	Query       string          `json:"query"`
	Filter      json.RawMessage `json:"filter,omitempty"`
	PageSize    int             `json:"page_size,omitempty"`
	StartCursor string          `json:"start_cursor,omitempty"`
}

func (p *searchParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query"}
	}
	if p.PageSize < 0 || p.PageSize > 100 {
		return &connectors.ValidationError{Message: "page_size must be between 1 and 100 (or 0 for default)"}
	}
	return nil
}

// Execute searches Notion and returns matching pages and databases.
func (a *searchAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := map[string]any{
		"query": params.Query,
	}
	if len(params.Filter) > 0 {
		var filter any
		if err := json.Unmarshal(params.Filter, &filter); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid filter JSON: %v", err)}
		}
		body["filter"] = filter
	}
	applyPagination(body, params.PageSize, params.StartCursor)

	var resp map[string]any
	if err := a.conn.do(ctx, http.MethodPost, "/v1/search", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
