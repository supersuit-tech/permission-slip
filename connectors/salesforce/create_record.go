package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createRecordAction implements connectors.Action for salesforce.create_record.
type createRecordAction struct {
	conn *SalesforceConnector
}

type createRecordParams struct {
	SObjectType string         `json:"sobject_type"`
	Fields      map[string]any `json:"fields"`
}

func (p *createRecordParams) validate() error {
	if p.SObjectType == "" {
		return &connectors.ValidationError{Message: "missing required parameter: sobject_type"}
	}
	if len(p.Fields) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: fields"}
	}
	return nil
}

// sfCreateResponse is the Salesforce response from sObject create.
type sfCreateResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

func (a *createRecordAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createRecordParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	baseURL, err := a.conn.apiBaseURL(req.Credentials)
	if err != nil {
		return nil, err
	}

	apiURL := baseURL + "/sobjects/" + url.PathEscape(params.SObjectType) + "/"

	var resp sfCreateResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, apiURL, params.Fields, &resp); err != nil {
		return nil, err
	}

	result := map[string]any{
		"id":           resp.ID,
		"sobject_type": params.SObjectType,
		"success":      resp.Success,
	}
	if url := recordURL(req.Credentials, resp.ID); url != "" {
		result["record_url"] = url
	}
	return connectors.JSONResult(result)
}
