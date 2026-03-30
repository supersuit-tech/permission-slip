package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateContactAction implements connectors.Action for hubspot.update_contact.
// It updates properties on an existing contact via PATCH /crm/v3/objects/contacts/{contact_id}.
type updateContactAction struct {
	conn *HubSpotConnector
}

type updateContactParams struct {
	ContactID  string            `json:"contact_id"`
	Properties map[string]string `json:"properties"`
}

func (p *updateContactParams) validate() error {
	if p.ContactID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: contact_id"}
	}
	if !isValidHubSpotID(p.ContactID) {
		return &connectors.ValidationError{Message: "contact_id must be a numeric HubSpot ID"}
	}
	if len(p.Properties) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: properties"}
	}
	return nil
}

func (a *updateContactAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateContactParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := hubspotObjectRequest{Properties: params.Properties}
	var resp hubspotObjectResponse
	path := fmt.Sprintf("/crm/v3/objects/contacts/%s", params.ContactID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPatch, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
