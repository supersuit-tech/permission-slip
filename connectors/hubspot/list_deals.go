package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listDealsAction implements connectors.Action for hubspot.list_deals.
// It searches deals via POST /crm/v3/objects/deals/search.
type listDealsAction struct {
	conn *HubSpotConnector
}

type listDealsFilter struct {
	PropertyName string `json:"propertyName"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
}

type listDealsSort struct {
	PropertyName string `json:"propertyName"`
	Direction    string `json:"direction"`
}

type listDealsParams struct {
	Filters    []listDealsFilter `json:"filters"`
	Sorts      []listDealsSort   `json:"sorts"`
	Limit      int               `json:"limit"`
	Properties []string          `json:"properties"`
}

const (
	defaultListDealsLimit = 10
	maxListDealsLimit     = 200 // HubSpot API maximum
)

var validSortDirections = map[string]bool{
	"ASCENDING":  true,
	"DESCENDING": true,
}

func (p *listDealsParams) validate() error {
	for i, f := range p.Filters {
		if f.PropertyName == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("filters[%d]: missing propertyName", i)}
		}
		if f.Operator == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("filters[%d]: missing operator", i)}
		}
		if !validSearchOperators[f.Operator] {
			return &connectors.ValidationError{Message: fmt.Sprintf("filters[%d]: unsupported operator %q", i, f.Operator)}
		}
	}
	for i, s := range p.Sorts {
		if s.PropertyName == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("sorts[%d]: missing propertyName", i)}
		}
		if s.Direction != "" && !validSortDirections[s.Direction] {
			return &connectors.ValidationError{Message: fmt.Sprintf("sorts[%d]: invalid direction %q (must be ASCENDING or DESCENDING)", i, s.Direction)}
		}
	}
	return nil
}

type listDealsRequest struct {
	FilterGroups []filterGroup   `json:"filterGroups,omitempty"`
	Sorts        []listDealsSort `json:"sorts,omitempty"`
	Limit        int             `json:"limit"`
	Properties   []string        `json:"properties,omitempty"`
}

func (a *listDealsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listDealsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = defaultListDealsLimit
	}
	if limit > maxListDealsLimit {
		limit = maxListDealsLimit
	}

	body := listDealsRequest{
		Limit:      limit,
		Sorts:      params.Sorts,
		Properties: params.Properties,
	}

	if len(params.Filters) > 0 {
		filters := make([]searchFilter, len(params.Filters))
		for i, f := range params.Filters {
			filters[i] = searchFilter{
				PropertyName: f.PropertyName,
				Operator:     f.Operator,
				Value:        f.Value,
			}
		}
		body.FilterGroups = []filterGroup{{Filters: filters}}
	}

	var resp searchResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/crm/v3/objects/deals/search", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
