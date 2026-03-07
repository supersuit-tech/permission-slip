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

type listDealsSort struct {
	PropertyName string `json:"propertyName"`
	Direction    string `json:"direction"`
}

type listDealsParams struct {
	Filters    []searchFilter  `json:"filters"`
	Sorts      []listDealsSort `json:"sorts"`
	Limit      int             `json:"limit"`
	Properties []string        `json:"properties"`
}

// defaultDealProperties are returned when no properties are specified,
// so callers always get a useful response without needing to know
// HubSpot's internal property names.
var defaultDealProperties = []string{
	"dealname", "amount", "dealstage", "pipeline",
	"closedate", "createdate", "hs_lastmodifieddate",
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
			return &connectors.ValidationError{Message: fmt.Sprintf("filters[%d]: unsupported operator %q (supported: EQ, NEQ, LT, LTE, GT, GTE, BETWEEN, CONTAINS_TOKEN, NOT_CONTAINS_TOKEN, HAS_PROPERTY, NOT_HAS_PROPERTY)", i, f.Operator)}
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

	props := params.Properties
	if len(props) == 0 {
		props = defaultDealProperties
	}

	body := listDealsRequest{
		Limit:      limit,
		Sorts:      params.Sorts,
		Properties: props,
	}

	if len(params.Filters) > 0 {
		body.FilterGroups = []filterGroup{{Filters: params.Filters}}
	}

	var resp searchResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/crm/v3/objects/deals/search", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
