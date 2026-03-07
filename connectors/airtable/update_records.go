package airtable

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateRecordsAction implements connectors.Action for airtable.update_records.
type updateRecordsAction struct {
	conn *AirtableConnector
}

type updateRecordsParams struct {
	BaseID  string              `json:"base_id"`
	Table   string              `json:"table"`
	Records []updateRecordInput `json:"records"`
}

type updateRecordInput struct {
	ID     string         `json:"id"`
	Fields map[string]any `json:"fields"`
}

func (p *updateRecordsParams) validate() error {
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
		if err := validateRecordID(r.ID); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("records[%d].id: %s", i, err.Error())}
		}
		if r.Fields == nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("records[%d].fields is required", i)}
		}
	}
	return nil
}

type updateRecordsRequest struct {
	Records []updateRecordInput `json:"records"`
}

type updateRecordsResponse struct {
	Records []recordEntry `json:"records"`
}

// Execute updates one or more records using PATCH (partial update).
func (a *updateRecordsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateRecordsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/%s/%s", a.conn.baseURL, params.BaseID, url.PathEscape(params.Table))

	body, err := json.Marshal(updateRecordsRequest{Records: params.Records})
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	var resp updateRecordsResponse
	if err := a.conn.doRequest(ctx, "PATCH", reqURL, req.Credentials, bytes.NewReader(body), &resp); err != nil {
		return nil, err
	}

	result := listRecordsResult{
		Records: make([]recordSummary, 0, len(resp.Records)),
	}
	for _, r := range resp.Records {
		result.Records = append(result.Records, recordSummary{
			ID:          r.ID,
			CreatedTime: r.CreatedTime,
			Fields:      r.Fields,
		})
	}

	return connectors.JSONResult(result)
}
