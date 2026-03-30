package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listContactsAction implements connectors.Action for hubspot.list_contacts.
// It searches contacts via POST /crm/v3/objects/contacts/search.
type listContactsAction struct {
	conn *HubSpotConnector
}

type listContactsParams struct {
	Filters    []searchFilter `json:"filters"`
	Limit      int            `json:"limit"`
	Properties []string       `json:"properties"`
}

// defaultContactProperties are returned when no properties are specified.
var defaultContactProperties = []string{
	"firstname", "lastname", "email", "phone", "company",
	"createdate", "hs_lastmodifieddate",
}

type listContactsRequest struct {
	FilterGroups []filterGroup  `json:"filterGroups,omitempty"`
	Limit        int            `json:"limit"`
	Properties   []string       `json:"properties,omitempty"`
}

func (a *listContactsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listContactsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	for i, f := range params.Filters {
		if f.PropertyName == "" {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("filters[%d]: missing propertyName", i)}
		}
		if f.Operator == "" {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("filters[%d]: missing operator", i)}
		}
		if !validSingleValueOperators[f.Operator] {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("filters[%d]: unsupported operator %q", i, f.Operator)}
		}
		if f.Value == "" {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("filters[%d]: missing value", i)}
		}
	}

	limit := params.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	props := params.Properties
	if len(props) == 0 {
		props = defaultContactProperties
	}

	body := listContactsRequest{
		Limit:      limit,
		Properties: props,
	}
	if len(params.Filters) > 0 {
		body.FilterGroups = []filterGroup{{Filters: params.Filters}}
	}

	var resp searchResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/crm/v3/objects/contacts/search", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
