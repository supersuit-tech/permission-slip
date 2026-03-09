package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// insertAction implements connectors.Action for supabase.insert.
type insertAction struct {
	conn *SupabaseConnector
}

type insertParams struct {
	Table         string           `json:"table"`
	Rows          []map[string]any `json:"rows"`
	Returning     string           `json:"returning,omitempty"`
	OnConflict    string           `json:"on_conflict,omitempty"`
	AllowedTables []string         `json:"allowed_tables,omitempty"`
}

func (p insertParams) validate() error {
	if err := validateTable(p.Table, p.AllowedTables); err != nil {
		return err
	}
	if len(p.Rows) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: rows"}
	}
	if len(p.Rows) > 1000 {
		return &connectors.ValidationError{Message: fmt.Sprintf("rows must contain at most 1000 items, got %d", len(p.Rows))}
	}
	return nil
}

// Execute inserts rows into a Supabase table via PostgREST POST.
func (a *insertAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params insertParams
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

	// Upsert: on_conflict triggers PostgREST's upsert behavior.
	// PostgREST requires both the on_conflict query param and the
	// Prefer: resolution=merge-duplicates header for upsert to work.
	var extraHeaders map[string]string
	if params.OnConflict != "" {
		q.Set("on_conflict", params.OnConflict)
		extraHeaders = map[string]string{
			"Prefer": "return=representation,resolution=merge-duplicates",
		}
	}

	reqURL += "?" + q.Encode()

	body, err := json.Marshal(params.Rows)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	var rows []map[string]any
	_, err = a.conn.doRequestWithHeaders(ctx, http.MethodPost, reqURL, apiKey, bytes.NewReader(body), &rows, extraHeaders)
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"rows":          rows,
		"rows_affected": len(rows),
	})
}
