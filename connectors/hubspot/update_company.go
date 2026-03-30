package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateCompanyAction implements connectors.Action for hubspot.update_company.
// It updates properties on an existing company via PATCH /crm/v3/objects/companies/{company_id}.
type updateCompanyAction struct {
	conn *HubSpotConnector
}

type updateCompanyParams struct {
	CompanyID  string            `json:"company_id"`
	Name       string            `json:"name"`
	Domain     string            `json:"domain"`
	Phone      string            `json:"phone"`
	City       string            `json:"city"`
	Country    string            `json:"country"`
	Industry   string            `json:"industry"`
	Properties map[string]string `json:"properties"`
}

func (p *updateCompanyParams) validate() error {
	if p.CompanyID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: company_id"}
	}
	if !isValidHubSpotID(p.CompanyID) {
		return &connectors.ValidationError{Message: "company_id must be a numeric HubSpot ID"}
	}
	// At least one field to update must be provided.
	if p.Name == "" && p.Domain == "" && p.Phone == "" && p.City == "" && p.Country == "" && p.Industry == "" && len(p.Properties) == 0 {
		return &connectors.ValidationError{Message: "at least one field to update must be provided (name, domain, phone, city, country, industry, or properties)"}
	}
	return nil
}

func (p *updateCompanyParams) toAPIProperties() map[string]string {
	overrides := map[string]string{}
	if p.Name != "" {
		overrides["name"] = p.Name
	}
	if p.Domain != "" {
		overrides["domain"] = p.Domain
	}
	if p.Phone != "" {
		overrides["phone"] = p.Phone
	}
	if p.City != "" {
		overrides["city"] = p.City
	}
	if p.Country != "" {
		overrides["country"] = p.Country
	}
	if p.Industry != "" {
		overrides["industry"] = p.Industry
	}
	return mergeProperties(p.Properties, overrides)
}

func (a *updateCompanyAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateCompanyParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := hubspotObjectRequest{Properties: params.toAPIProperties()}
	var resp hubspotObjectResponse
	path := fmt.Sprintf("/crm/v3/objects/companies/%s", params.CompanyID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPatch, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
