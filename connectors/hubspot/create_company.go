package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createCompanyAction implements connectors.Action for hubspot.create_company.
// It creates a new company via POST /crm/v3/objects/companies.
type createCompanyAction struct {
	conn *HubSpotConnector
}

type createCompanyParams struct {
	Name       string            `json:"name"`
	Domain     string            `json:"domain"`
	Phone      string            `json:"phone"`
	City       string            `json:"city"`
	Country    string            `json:"country"`
	Industry   string            `json:"industry"`
	Properties map[string]string `json:"properties"`
}

func (p *createCompanyParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

func (p *createCompanyParams) toAPIProperties() map[string]string {
	overrides := map[string]string{"name": p.Name}
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

func (a *createCompanyAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCompanyParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := hubspotObjectRequest{Properties: params.toAPIProperties()}
	var resp hubspotObjectResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/crm/v3/objects/companies", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
