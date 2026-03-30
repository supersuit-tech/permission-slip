package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateContactAction implements connectors.Action for intercom.update_contact.
// It updates a contact via PUT /contacts/{id}.
type updateContactAction struct {
	conn *IntercomConnector
}

type updateContactParams struct {
	ContactID        string         `json:"contact_id"`
	Email            string         `json:"email"`
	Phone            string         `json:"phone"`
	Name             string         `json:"name"`
	CustomAttributes map[string]any `json:"custom_attributes"`
}

func (p *updateContactParams) validate() error {
	if p.ContactID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: contact_id"}
	}
	if !isValidIntercomID(p.ContactID) {
		return &connectors.ValidationError{Message: "contact_id contains invalid characters"}
	}
	if p.Email == "" && p.Phone == "" && p.Name == "" && len(p.CustomAttributes) == 0 {
		return &connectors.ValidationError{Message: "at least one of email, phone, name, or custom_attributes must be provided"}
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

	body := map[string]any{}
	if params.Email != "" {
		body["email"] = params.Email
	}
	if params.Phone != "" {
		body["phone"] = params.Phone
	}
	if params.Name != "" {
		body["name"] = params.Name
	}
	if len(params.CustomAttributes) > 0 {
		body["custom_attributes"] = params.CustomAttributes
	}

	var resp intercomContact
	path := fmt.Sprintf("/contacts/%s", params.ContactID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
