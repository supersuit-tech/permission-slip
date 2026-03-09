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

	// Column selection for returned rows.
	ret := params.Returning
	if ret == "" {
		ret = "*"
	}
	q.Set("select", ret)

	// Upsert: on_conflict triggers PostgREST's upsert behavior.
	if params.OnConflict != "" {
		q.Set("on_conflict", params.OnConflict)
	}

	reqURL += "?" + q.Encode()

	body, err := json.Marshal(params.Rows)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	method := http.MethodPost

	var rows []map[string]any
	if err := a.conn.doRequest(ctx, method, reqURL, apiKey, bytes.NewReader(body), &rows); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"rows":          rows,
		"rows_affected": len(rows),
	})
}
