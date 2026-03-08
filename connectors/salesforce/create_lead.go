package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createLeadAction implements connectors.Action for salesforce.create_lead.
type createLeadAction struct {
	conn *SalesforceConnector
}

type createLeadParams struct {
	LastName   string `json:"last_name"`
	Company    string `json:"company"`
	FirstName  string `json:"first_name,omitempty"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Title      string `json:"title,omitempty"`
	LeadSource string `json:"lead_source,omitempty"`
	Status     string `json:"status,omitempty"`
	Website    string `json:"website,omitempty"`
	Industry   string `json:"industry,omitempty"`
}

func (p *createLeadParams) validate() error {
	if p.LastName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: last_name"}
	}
	if p.Company == "" {
		return &connectors.ValidationError{Message: "missing required parameter: company"}
	}
	if p.Email != "" && !strings.Contains(p.Email, "@") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid email: %q does not appear to be a valid email address", p.Email)}
	}
	return nil
}

func (a *createLeadAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createLeadParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	fields := map[string]any{
		"LastName": params.LastName,
		"Company":  params.Company,
	}
	if params.FirstName != "" {
		fields["FirstName"] = params.FirstName
	}
	if params.Email != "" {
		fields["Email"] = params.Email
	}
	if params.Phone != "" {
		fields["Phone"] = params.Phone
	}
	if params.Title != "" {
		fields["Title"] = params.Title
	}
	if params.LeadSource != "" {
		fields["LeadSource"] = params.LeadSource
	}
	if params.Status != "" {
		fields["Status"] = params.Status
	}
	if params.Website != "" {
		fields["Website"] = params.Website
	}
	if params.Industry != "" {
		fields["Industry"] = params.Industry
	}

	baseURL, err := a.conn.apiBaseURL(req.Credentials)
	if err != nil {
		return nil, err
	}

	apiURL := baseURL + "/sobjects/Lead/"

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
