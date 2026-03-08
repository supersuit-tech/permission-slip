package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateCompanyAction implements connectors.Action for hubspot.update_company.
// It updates properties on an existing company via PATCH /crm/v3/objects/companies/{company_id}.
type updateCompanyAction struct {
	conn *HubSpotConnector
}

type updateCompanyParams struct {
	CompanyID  string            `json:"company_id"`
	Properties map[string]string `json:"properties"`
}

func (p *updateCompanyParams) validate() error {
	if p.CompanyID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: company_id"}
	}
	if !isValidHubSpotID(p.CompanyID) {
		return &connectors.ValidationError{Message: "company_id must be a numeric HubSpot ID"}
	}
	if len(p.Properties) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: properties"}
	}
	return nil
}

func (a *updateCompanyAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateCompanyParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := hubspotObjectRequest{Properties: params.Properties}
	var resp hubspotObjectResponse
	path := fmt.Sprintf("/crm/v3/objects/companies/%s", params.CompanyID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPatch, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
