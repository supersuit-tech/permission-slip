package airtable

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listRecordsAction implements connectors.Action for airtable.list_records.
type listRecordsAction struct {
	conn *AirtableConnector
}

type listRecordsParams struct {
	BaseID          string       `json:"base_id"`
	Table           string       `json:"table"`
	Fields          []string     `json:"fields,omitempty"`
	FilterByFormula string       `json:"filter_by_formula,omitempty"`
	MaxRecords      int          `json:"max_records,omitempty"`
	PageSize        int          `json:"page_size,omitempty"`
	Sort            []sortField  `json:"sort,omitempty"`
	View            string       `json:"view,omitempty"`
	Offset          string       `json:"offset,omitempty"`
}

type sortField struct {
	Field     string `json:"field"`
	Direction string `json:"direction,omitempty"`
}

func (p *listRecordsParams) validate() error {
	if err := validateBaseID(p.BaseID); err != nil {
		return err
	}
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if p.MaxRecords < 0 {
		return &connectors.ValidationError{Message: fmt.Sprintf("max_records must be non-negative, got %d", p.MaxRecords)}
	}
	if p.PageSize != 0 && (p.PageSize < 1 || p.PageSize > 100) {
		return &connectors.ValidationError{Message: fmt.Sprintf("page_size must be between 1 and 100, got %d", p.PageSize)}
	}
	return nil
}

type listRecordsResponse struct {
	Records []recordEntry `json:"records"`
	Offset  string        `json:"offset,omitempty"`
}

type recordEntry struct {
	ID          string         `json:"id"`
	CreatedTime string         `json:"createdTime"`
	Fields      map[string]any `json:"fields"`
}

type listRecordsResult struct {
	Records []recordSummary `json:"records"`
	Offset  string          `json:"offset,omitempty"`
}

type recordSummary struct {
	ID          string         `json:"id"`
	CreatedTime string         `json:"created_time"`
	Fields      map[string]any `json:"fields"`
}

// Execute lists records from an Airtable table using Airtable's standard GET
// list endpoint with query parameters for filtering, sorting, and pagination.
func (a *listRecordsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listRecordsParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	// Use GET with query params for simple requests to match Airtable's standard API.
	reqURL := fmt.Sprintf("%s/%s/%s", a.conn.baseURL, url.PathEscape(params.BaseID), url.PathEscape(params.Table))

	q := url.Values{}
	if params.FilterByFormula != "" {
		q.Set("filterByFormula", params.FilterByFormula)
	}
	if params.MaxRecords > 0 {
		q.Set("maxRecords", strconv.Itoa(params.MaxRecords))
	}
	if params.PageSize > 0 {
		q.Set("pageSize", strconv.Itoa(params.PageSize))
	}
	if params.View != "" {
		q.Set("view", params.View)
	}
	if params.Offset != "" {
		q.Set("offset", params.Offset)
	}
	for _, f := range params.Fields {
		q.Add("fields[]", f)
	}
	for i, s := range params.Sort {
		q.Set(fmt.Sprintf("sort[%d][field]", i), s.Field)
		if s.Direction != "" {
			q.Set(fmt.Sprintf("sort[%d][direction]", i), s.Direction)
		}
	}

	if encoded := q.Encode(); encoded != "" {
		reqURL += "?" + encoded
	}

	var resp listRecordsResponse
	if err := a.conn.doRequest(ctx, "GET", reqURL, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(toListRecordsResult(resp))
}

// recordEntriesToSummaries converts a slice of recordEntry (API response) to
// recordSummary (client-facing result). Shared by list, create, update, and search.
func recordEntriesToSummaries(entries []recordEntry) []recordSummary {
	out := make([]recordSummary, 0, len(entries))
	for _, r := range entries {
		out = append(out, recordSummary{
			ID:          r.ID,
			CreatedTime: r.CreatedTime,
			Fields:      r.Fields,
		})
	}
	return out
}

func toListRecordsResult(resp listRecordsResponse) listRecordsResult {
	result := listRecordsResult{
		Records: recordEntriesToSummaries(resp.Records),
		Offset:  resp.Offset,
	}
	return result
}
