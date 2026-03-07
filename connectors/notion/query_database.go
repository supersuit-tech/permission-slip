package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// queryDatabaseAction implements connectors.Action for notion.query_database.
// It queries a database with filters and sorts via POST /v1/databases/{database_id}/query.
type queryDatabaseAction struct {
	conn *NotionConnector
}

// queryDatabaseParams is the user-facing parameter schema.
type queryDatabaseParams struct {
	DatabaseID string          `json:"database_id"`
	Filter     json.RawMessage `json:"filter,omitempty"`
	Sorts      json.RawMessage `json:"sorts,omitempty"`
	PageSize   int             `json:"page_size,omitempty"`
}

func (p *queryDatabaseParams) validate() error {
	if p.DatabaseID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: database_id"}
	}
	if p.PageSize < 0 || p.PageSize > 100 {
		return &connectors.ValidationError{Message: "page_size must be between 0 and 100"}
	}
	return nil
}

// Execute queries a Notion database and returns the matching pages.
func (a *queryDatabaseAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params queryDatabaseParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := make(map[string]any)
	if len(params.Filter) > 0 {
		var filter any
		if err := json.Unmarshal(params.Filter, &filter); err == nil {
			body["filter"] = filter
		}
	}
	if len(params.Sorts) > 0 {
		var sorts any
		if err := json.Unmarshal(params.Sorts, &sorts); err == nil {
			body["sorts"] = sorts
		}
	}
	pageSize := params.PageSize
	if pageSize == 0 {
		pageSize = 100
	}
	body["page_size"] = pageSize

	var resp map[string]any
	if err := a.conn.do(ctx, http.MethodPost, "/v1/databases/"+params.DatabaseID+"/query", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
