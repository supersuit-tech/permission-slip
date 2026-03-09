package supabase

import (
	"context"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// deleteAction implements connectors.Action for supabase.delete.
type deleteAction struct {
	conn *SupabaseConnector
}

type deleteParams struct {
	Table         string            `json:"table"`
	Filters       map[string]string `json:"filters"`
	Returning     string            `json:"returning,omitempty"`
	AllowedTables []string          `json:"allowed_tables,omitempty"`
}

func (p deleteParams) validate() error {
	if err := validateTable(p.Table, p.AllowedTables); err != nil {
		return err
	}
	if len(p.Filters) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: filters — at least one filter is required to prevent accidental full-table deletes"}
	}
	return validateFilters(p.Filters)
}

// Execute deletes rows from a Supabase table via PostgREST DELETE.
func (a *deleteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	baseURL, apiKey, err := resolveConfig(req.Credentials)
	if err != nil {
		return nil, err
	}

	reqURL := restURL(baseURL, params.Table)

	q := url.Values{}

	returningSelect(q, params.Returning)

	// Apply filters to scope the delete.
	if err := applyFilters(q, params.Filters); err != nil {
		return nil, err
	}

	reqURL += "?" + q.Encode()

	var rows []map[string]any
	if err := a.conn.doRequest(ctx, http.MethodDelete, reqURL, apiKey, nil, &rows); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"rows":          rows,
		"rows_affected": len(rows),
	})
}
