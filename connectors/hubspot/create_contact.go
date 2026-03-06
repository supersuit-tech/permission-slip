package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createContactAction implements connectors.Action for hubspot.create_contact.
// It creates a new contact via POST /crm/v3/objects/contacts.
type createContactAction struct {
	conn *HubSpotConnector
}

type createContactParams struct {
	Email      string            `json:"email"`
	FirstName  string            `json:"firstname"`
	LastName   string            `json:"lastname"`
	Phone      string            `json:"phone"`
	Company    string            `json:"company"`
	Properties map[string]string `json:"properties"`
}

func (p *createContactParams) validate() error {
	if p.Email == "" {
		return &connectors.ValidationError{Message: "missing required parameter: email"}
	}
	return nil
}

// toAPIProperties merges explicit fields and the additional properties map
// into the flat properties object that HubSpot expects.
func (p *createContactParams) toAPIProperties() map[string]string {
	props := make(map[string]string)
	// Copy additional properties first so explicit fields take precedence.
	for k, v := range p.Properties {
		props[k] = v
	}
	props["email"] = p.Email
	if p.FirstName != "" {
		props["firstname"] = p.FirstName
	}
	if p.LastName != "" {
		props["lastname"] = p.LastName
	}
	if p.Phone != "" {
		props["phone"] = p.Phone
	}
	if p.Company != "" {
		props["company"] = p.Company
	}
	return props
}

type hubspotObjectRequest struct {
	Properties map[string]string `json:"properties"`
}

type hubspotObjectResponse struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties"`
}

func (a *createContactAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createContactParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := hubspotObjectRequest{Properties: params.toAPIProperties()}
	var resp hubspotObjectResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/crm/v3/objects/contacts", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
