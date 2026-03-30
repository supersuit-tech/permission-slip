package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// addToListAction implements connectors.Action for sendgrid.add_to_list.
// It adds a contact to a SendGrid marketing contact list using the
// PUT /marketing/contacts endpoint.
type addToListAction struct {
	conn *SendGridConnector
}

type addToListParams struct {
	ListID    string `json:"list_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (p *addToListParams) validate() error {
	if p.ListID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: list_id"}
	}
	if p.Email == "" {
		return &connectors.ValidationError{Message: "missing required parameter: email"}
	}
	if !emailPattern.MatchString(p.Email) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid email address: %q", p.Email)}
	}
	return nil
}

// Execute adds a contact to the specified list.
func (a *addToListAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addToListParams
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

	body := map[string]any{
		"list_ids": []string{params.ListID},
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
