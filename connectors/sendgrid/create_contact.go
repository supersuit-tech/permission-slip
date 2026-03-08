package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createContactAction implements connectors.Action for sendgrid.create_contact.
// It upserts a contact in SendGrid without requiring a list, using the
// PUT /marketing/contacts endpoint with no list_ids.
type createContactAction struct {
	conn *SendGridConnector
}

type createContactParams struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone_number"`
	City      string `json:"city"`
	Country   string `json:"country"`
}

func (p *createContactParams) validate() error {
	if p.Email == "" {
		return &connectors.ValidationError{Message: "missing required parameter: email"}
	}
	if !emailPattern.MatchString(p.Email) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid email address: %q", p.Email)}
	}
	return nil
}

// Execute upserts a contact via PUT /marketing/contacts.
func (a *createContactAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createContactParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	contact := map[string]string{"email": params.Email}
	if params.FirstName != "" {
		contact["first_name"] = params.FirstName
	}
	if params.LastName != "" {
		contact["last_name"] = params.LastName
	}
	if params.Phone != "" {
		contact["phone_number"] = params.Phone
	}
	if params.City != "" {
		contact["city"] = params.City
	}
	if params.Country != "" {
		contact["country"] = params.Country
	}

	body := map[string]any{
		"contacts": []map[string]string{contact},
	}

	var resp struct {
		JobID string `json:"job_id"`
	}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPut, "/marketing/contacts", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"job_id": resp.JobID,
		"email":  params.Email,
		"status": "accepted",
	})
}
