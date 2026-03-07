package airtable

import (
	"context"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getRecordAction implements connectors.Action for airtable.get_record.
type getRecordAction struct {
	conn *AirtableConnector
}

type getRecordParams struct {
	BaseID   string `json:"base_id"`
	Table    string `json:"table"`
	RecordID string `json:"record_id"`
}

func (p getRecordParams) validate() error {
	if err := validateBaseID(p.BaseID); err != nil {
		return err
	}
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	return validateRecordID(p.RecordID)
}

// Execute gets a single record by ID.
func (a *getRecordAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getRecordParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/%s/%s/%s", a.conn.baseURL, url.PathEscape(params.BaseID), url.PathEscape(params.Table), url.PathEscape(params.RecordID))

	var resp recordEntry
	if err := a.conn.doRequest(ctx, "GET", reqURL, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(recordSummary{
		ID:          resp.ID,
		CreatedTime: resp.CreatedTime,
		Fields:      resp.Fields,
	})
}
