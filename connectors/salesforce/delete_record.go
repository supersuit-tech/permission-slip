package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteRecordAction implements connectors.Action for salesforce.delete_record.
type deleteRecordAction struct {
	conn *SalesforceConnector
}

type deleteRecordParams struct {
	SObjectType string `json:"sobject_type"`
	RecordID    string `json:"record_id"`
}

func (p *deleteRecordParams) validate() error {
	if p.SObjectType == "" {
		return &connectors.ValidationError{Message: "missing required parameter: sobject_type"}
	}
	if err := validateSObjectType(p.SObjectType, "sobject_type"); err != nil {
		return err
	}
	if p.RecordID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: record_id"}
	}
	if err := validateRecordID(p.RecordID, "record_id"); err != nil {
		return err
	}
	return nil
}

func (a *deleteRecordAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteRecordParams
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

	// Salesforce DELETE returns 204 No Content on success.
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodDelete, apiURL, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"record_id":    params.RecordID,
		"sobject_type": params.SObjectType,
		"success":      true,
	})
}
