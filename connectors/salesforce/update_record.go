package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateRecordAction implements connectors.Action for salesforce.update_record.
type updateRecordAction struct {
	conn *SalesforceConnector
}

type updateRecordParams struct {
	SObjectType string         `json:"sobject_type"`
	RecordID    string         `json:"record_id"`
	Fields      map[string]any `json:"fields"`
}

func (p *updateRecordParams) validate() error {
	if p.SObjectType == "" {
		return &connectors.ValidationError{Message: "missing required parameter: sobject_type"}
	}
	if p.RecordID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: record_id"}
	}
	if len(p.Fields) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: fields"}
	}
	return nil
}

func (a *updateRecordAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateRecordParams
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

	apiURL := baseURL + "/sobjects/" + url.PathEscape(params.SObjectType) + "/" + url.PathEscape(params.RecordID)

	// Salesforce PATCH returns 204 No Content on success.
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPatch, apiURL, params.Fields, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"record_id": params.RecordID,
		"success":   true,
	})
}
