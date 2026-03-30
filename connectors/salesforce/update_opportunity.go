package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateOpportunityAction implements connectors.Action for salesforce.update_opportunity.
type updateOpportunityAction struct {
	conn *SalesforceConnector
}

type updateOpportunityParams struct {
	RecordID    string   `json:"record_id"`
	StageName   string   `json:"stage_name,omitempty"`
	Amount      *float64 `json:"amount,omitempty"`
	CloseDate   string   `json:"close_date,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
}

func (p *updateOpportunityParams) validate() error {
	if p.RecordID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: record_id"}
	}
	if err := validateRecordID(p.RecordID, "record_id"); err != nil {
		return err
	}
	if p.StageName == "" && p.Amount == nil && p.CloseDate == "" && p.Name == "" && p.Description == "" {
		return &connectors.ValidationError{Message: "at least one field to update is required (stage_name, amount, close_date, name, or description)"}
	}
	if p.CloseDate != "" {
		if err := validateDate(p.CloseDate, "close_date"); err != nil {
			return err
		}
	}
	if p.Amount != nil && *p.Amount < 0 {
		return &connectors.ValidationError{Message: "invalid amount: must be non-negative"}
	}
	return nil
}

func (a *updateOpportunityAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateOpportunityParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	fields := map[string]any{}
	if params.StageName != "" {
		fields["StageName"] = params.StageName
	}
	if params.Amount != nil {
		fields["Amount"] = *params.Amount
	}
	if params.CloseDate != "" {
		fields["CloseDate"] = params.CloseDate
	}
	if params.Name != "" {
		fields["Name"] = params.Name
	}
	if params.Description != "" {
		fields["Description"] = params.Description
	}

	baseURL, err := a.conn.apiBaseURL(req.Credentials)
	if err != nil {
		return nil, err
	}

	apiURL := baseURL + "/sobjects/Opportunity/" + url.PathEscape(params.RecordID)

	// Salesforce PATCH returns 204 No Content on success.
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPatch, apiURL, fields, nil); err != nil {
		return nil, err
	}

	result := map[string]any{
		"record_id": params.RecordID,
		"success":   true,
	}
	if url := recordURL(req.Credentials, params.RecordID); url != "" {
		result["record_url"] = url
	}
	return connectors.JSONResult(result)
}
