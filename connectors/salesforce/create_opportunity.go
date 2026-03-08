package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createOpportunityAction implements connectors.Action for salesforce.create_opportunity.
type createOpportunityAction struct {
	conn *SalesforceConnector
}

type createOpportunityParams struct {
	Name        string  `json:"name"`
	StageName   string  `json:"stage_name"`
	CloseDate   string  `json:"close_date"`
	Amount      float64 `json:"amount,omitempty"`
	AccountID   string  `json:"account_id,omitempty"`
	Description string  `json:"description,omitempty"`
}

func (p *createOpportunityParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.StageName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: stage_name"}
	}
	if p.CloseDate == "" {
		return &connectors.ValidationError{Message: "missing required parameter: close_date"}
	}
	if p.AccountID != "" {
		if err := validateRecordID(p.AccountID, "account_id"); err != nil {
			return err
		}
	}
	return nil
}

func (a *createOpportunityAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createOpportunityParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	fields := map[string]any{
		"Name":      params.Name,
		"StageName": params.StageName,
		"CloseDate": params.CloseDate,
	}
	if params.Amount != 0 {
		fields["Amount"] = params.Amount
	}
	if params.AccountID != "" {
		fields["AccountId"] = params.AccountID
	}
	if params.Description != "" {
		fields["Description"] = params.Description
	}

	baseURL, err := a.conn.apiBaseURL(req.Credentials)
	if err != nil {
		return nil, err
	}

	apiURL := baseURL + "/sobjects/Opportunity/"

	var resp sfCreateResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, apiURL, fields, &resp); err != nil {
		return nil, err
	}

	result := map[string]any{
		"id":      resp.ID,
		"success": resp.Success,
	}
	if url := recordURL(req.Credentials, resp.ID); url != "" {
		result["record_url"] = url
	}
	return connectors.JSONResult(result)
}
