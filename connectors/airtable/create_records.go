package airtable

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createRecordsAction implements connectors.Action for airtable.create_records.
type createRecordsAction struct {
	conn *AirtableConnector
}

type createRecordsParams struct {
	BaseID  string              `json:"base_id"`
	Table   string              `json:"table"`
	Records []createRecordInput `json:"records"`
}

type createRecordInput struct {
	Fields map[string]any `json:"fields"`
}

func (p *createRecordsParams) validate() error {
	if err := validateBaseID(p.BaseID); err != nil {
		return err
	}
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if len(p.Records) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: records"}
	}
	if len(p.Records) > 10 {
		return &connectors.ValidationError{Message: "records must contain at most 10 items"}
	}
	for i, r := range p.Records {
		if r.Fields == nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("records[%d].fields is required", i)}
		}
	}
	return nil
}

type createRecordsRequest struct {
	Records []createRecordInput `json:"records"`
}

type createRecordsResponse struct {
	Records []recordEntry `json:"records"`
}

// Execute creates one or more records in an Airtable table.
func (a *createRecordsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createRecordsParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/%s/%s", a.conn.baseURL, url.PathEscape(params.BaseID), url.PathEscape(params.Table))

	body, err := json.Marshal(createRecordsRequest{Records: params.Records})
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	var resp createRecordsResponse
	if err := a.conn.doRequest(ctx, "POST", reqURL, req.Credentials, bytes.NewReader(body), &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(listRecordsResult{
		Records: recordEntriesToSummaries(resp.Records),
	})
}
