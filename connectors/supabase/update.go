package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateAction implements connectors.Action for supabase.update.
type updateAction struct {
	conn *SupabaseConnector
}

type updateParams struct {
	Table         string            `json:"table"`
	Set           map[string]any    `json:"set"`
	Filters       map[string]string `json:"filters"`
	Returning     string            `json:"returning,omitempty"`
	AllowedTables []string          `json:"allowed_tables,omitempty"`
}

func (p updateParams) validate() error {
	if err := validateTable(p.Table, p.AllowedTables); err != nil {
		return err
	}
	if len(p.Set) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: set"}
	}
	if len(p.Filters) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: filters — at least one filter is required to prevent accidental full-table updates"}
	}
	return validateFilters(p.Filters)
}

// Execute updates rows in a Supabase table via PostgREST PATCH.
func (a *updateAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateParams
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

	// Apply filters to scope the update.
	if err := applyFilters(q, params.Filters); err != nil {
		return nil, err
	}

	reqURL += "?" + q.Encode()

	body, err := json.Marshal(params.Set)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	var rows []map[string]any
	if err := a.conn.doRequest(ctx, http.MethodPatch, reqURL, apiKey, bytes.NewReader(body), &rows); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"rows":          rows,
		"rows_affected": len(rows),
	})
}
