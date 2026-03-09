package supabase

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// readAction implements connectors.Action for supabase.read.
type readAction struct {
	conn *SupabaseConnector
}

type readParams struct {
	Table         string            `json:"table"`
	Select        string            `json:"select,omitempty"`
	Filters       map[string]string `json:"filters,omitempty"`
	Order         string            `json:"order,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Offset        int               `json:"offset,omitempty"`
	CountTotal    bool              `json:"count_total,omitempty"`
	AllowedTables []string          `json:"allowed_tables,omitempty"`
}

func (p readParams) validate() error {
	if err := validateTable(p.Table, p.AllowedTables); err != nil {
		return err
	}
	if p.Limit < 0 {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be non-negative, got %d", p.Limit)}
	}
	if p.Offset < 0 {
		return &connectors.ValidationError{Message: fmt.Sprintf("offset must be non-negative, got %d", p.Offset)}
	}
	return validateFilters(p.Filters)
}

// Execute reads rows from a Supabase table via PostgREST GET.
func (a *readAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params readParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	baseURL, apiKey, err := resolveConfig(req.Credentials)
	if err != nil {
		return nil, err
	}

	reqURL := restURL(baseURL, params.Table)

	q := url.Values{}

	// Column selection.
	sel := params.Select
	if sel == "" {
		sel = "*"
	}
	q.Set("select", sel)

	// Apply filters.
	if err := applyFilters(q, params.Filters); err != nil {
		return nil, err
	}

	// Ordering.
	if params.Order != "" {
		q.Set("order", params.Order)
	}

	// Pagination.
	limit := params.Limit
	if limit <= 0 {
		limit = defaultMaxRows
	}
	q.Set("limit", strconv.Itoa(limit))

	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}

	reqURL += "?" + q.Encode()

	// Build extra headers for exact count if requested.
	var extraHeaders map[string]string
	if params.CountTotal {
		extraHeaders = map[string]string{"Prefer": "count=exact"}
	}

	var rows []map[string]any
	contentRange, err := a.conn.doRequestWithHeaders(ctx, "GET", reqURL, apiKey, nil, &rows, extraHeaders)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"rows":  rows,
		"count": len(rows),
	}

	// Parse total count from Content-Range header (e.g., "0-9/42").
	if params.CountTotal && contentRange != "" {
		if total := parseTotalFromContentRange(contentRange); total >= 0 {
			result["total_count"] = total
		}
	}

	return connectors.JSONResult(result)
}
