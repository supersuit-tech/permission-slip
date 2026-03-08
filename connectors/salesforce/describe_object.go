package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// describeObjectAction implements connectors.Action for salesforce.describe_object.
type describeObjectAction struct {
	conn *SalesforceConnector
}

type describeObjectParams struct {
	SObjectType string `json:"sobject_type"`
}

func (p *describeObjectParams) validate() error {
	if p.SObjectType == "" {
		return &connectors.ValidationError{Message: "missing required parameter: sobject_type"}
	}
	return nil
}

func (a *describeObjectAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params describeObjectParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	baseURL, err := a.conn.apiBaseURL(req.Credentials)
	if err != nil {
		return nil, err
	}

	apiURL := baseURL + "/sobjects/" + url.PathEscape(params.SObjectType) + "/describe"

	var result json.RawMessage
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, apiURL, nil, &result); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: result}, nil
}
