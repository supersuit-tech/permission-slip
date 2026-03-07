package airtable

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// deleteRecordsAction implements connectors.Action for airtable.delete_records.
type deleteRecordsAction struct {
	conn *AirtableConnector
}

type deleteRecordsParams struct {
	BaseID    string   `json:"base_id"`
	Table     string   `json:"table"`
	RecordIDs []string `json:"record_ids"`
}

func (p *deleteRecordsParams) validate() error {
	if err := validateBaseID(p.BaseID); err != nil {
		return err
	}
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if len(p.RecordIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: record_ids"}
	}
	if len(p.RecordIDs) > 10 {
		return &connectors.ValidationError{Message: "record_ids must contain at most 10 items"}
	}
	for i, id := range p.RecordIDs {
		if err := validateRecordID(id); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("record_ids[%d]: %s", i, err.Error())}
		}
	}
	return nil
}

type deleteRecordsResponse struct {
	Records []deleteRecordEntry `json:"records"`
}

type deleteRecordEntry struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type deleteRecordsResult struct {
	Deleted []deleteRecordSummary `json:"deleted"`
}

type deleteRecordSummary struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// Execute deletes one or more records from an Airtable table.
func (a *deleteRecordsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteRecordsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Airtable DELETE uses query params: ?records[]=rec1&records[]=rec2
	reqURL := fmt.Sprintf("%s/%s/%s", a.conn.baseURL, params.BaseID, url.PathEscape(params.Table))

	queryParts := make([]string, 0, len(params.RecordIDs))
	for _, id := range params.RecordIDs {
		queryParts = append(queryParts, "records[]="+url.QueryEscape(id))
	}
	reqURL += "?" + strings.Join(queryParts, "&")

	var resp deleteRecordsResponse
	if err := a.conn.doRequest(ctx, "DELETE", reqURL, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	result := deleteRecordsResult{
		Deleted: make([]deleteRecordSummary, 0, len(resp.Records)),
	}
	for _, r := range resp.Records {
		result.Deleted = append(result.Deleted, deleteRecordSummary{
			ID:      r.ID,
			Deleted: r.Deleted,
		})
	}

	return connectors.JSONResult(result)
}
