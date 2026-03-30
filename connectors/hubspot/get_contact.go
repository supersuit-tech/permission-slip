package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getContactAction implements connectors.Action for hubspot.get_contact.
// It fetches a single contact via GET /crm/v3/objects/contacts/{contact_id}.
type getContactAction struct {
	conn *HubSpotConnector
}

type getContactParams struct {
	ContactID  string   `json:"contact_id"`
	Properties []string `json:"properties"`
}

func (p *getContactParams) validate() error {
	if p.ContactID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: contact_id"}
	}
	if !isValidHubSpotID(p.ContactID) {
		return &connectors.ValidationError{Message: "contact_id must be a numeric HubSpot ID"}
	}
	return nil
}

func (a *getContactAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getContactParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	props := params.Properties
	if len(props) == 0 {
		props = defaultContactProperties
	}

	q := url.Values{}
	for _, p := range props {
		q.Add("properties", p)
	}
	path := fmt.Sprintf("/crm/v3/objects/contacts/%s?%s", params.ContactID, q.Encode())
	var resp hubspotObjectResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
