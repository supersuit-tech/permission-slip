package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createContactAction implements connectors.Action for intercom.create_contact.
// It creates a contact (user or lead) via POST /contacts.
type createContactAction struct {
	conn *IntercomConnector
}

type createContactParams struct {
	Email    string            `json:"email"`
	Phone    string            `json:"phone"`
	Name     string            `json:"name"`
	Role     string            `json:"role"` // "user" or "lead"
	CustomAttributes map[string]any `json:"custom_attributes"`
}

var validContactRoles = map[string]bool{
	"user": true,
	"lead": true,
}

func (p *createContactParams) validate() error {
	if p.Email == "" && p.Phone == "" && p.Name == "" {
		return &connectors.ValidationError{Message: "at least one of email, phone, or name is required"}
	}
	if p.Role != "" && !validContactRoles[p.Role] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid role %q: must be user or lead", p.Role)}
	}
	return nil
}

func (a *createContactAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createContactParams
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
	if params.Role != "" {
		body["role"] = params.Role
	}
	if len(params.CustomAttributes) > 0 {
		body["custom_attributes"] = params.CustomAttributes
	}

	var resp intercomContact
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/contacts", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
