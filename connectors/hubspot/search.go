package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchAction implements connectors.Action for hubspot.search.
// It searches CRM objects via POST /crm/v3/objects/{object_type}/search.
type searchAction struct {
	conn *HubSpotConnector
}

type searchFilter struct {
	PropertyName string `json:"propertyName"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
}

type searchParams struct {
	ObjectType string         `json:"object_type"` // contacts, deals, tickets, companies
	Filters    []searchFilter `json:"filters"`
	Limit      int            `json:"limit"`
}

const defaultSearchLimit = 10

var validSearchObjectTypes = map[string]bool{
	"contacts":  true,
	"deals":     true,
	"tickets":   true,
	"companies": true,
}

func (p *searchParams) validate() error {
	if p.ObjectType == "" {
		return &connectors.ValidationError{Message: "missing required parameter: object_type"}
	}
	if !validSearchObjectTypes[p.ObjectType] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid object_type %q: must be contacts, deals, tickets, or companies", p.ObjectType)}
	}
	if len(p.Filters) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: filters"}
	}
	for i, f := range p.Filters {
		if f.PropertyName == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("filter[%d]: missing propertyName", i)}
		}
		if f.Operator == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("filter[%d]: missing operator", i)}
		}
	}
	return nil
}

type searchRequest struct {
	FilterGroups []filterGroup `json:"filterGroups"`
	Limit        int           `json:"limit"`
}

type filterGroup struct {
	Filters []searchFilter `json:"filters"`
}

type searchResponse struct {
	Total   int                     `json:"total"`
	Results []hubspotObjectResponse `json:"results"`
}

func (a *searchAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	body := searchRequest{
		FilterGroups: []filterGroup{
			{Filters: params.Filters},
		},
		Limit: limit,
	}

	var resp searchResponse
	path := fmt.Sprintf("/crm/v3/objects/%s/search", params.ObjectType)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
