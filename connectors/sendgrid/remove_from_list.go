package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// removeFromListAction implements connectors.Action for sendgrid.remove_from_list.
// It removes a contact from a SendGrid contact list using
// DELETE /marketing/lists/{list_id}/contacts?contact_ids={id}.
type removeFromListAction struct {
	conn *SendGridConnector
}

type removeFromListParams struct {
	ListID    string `json:"list_id"`
	ContactID string `json:"contact_id"`
}

func (p *removeFromListParams) validate() error {
	if p.ListID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: list_id"}
	}
	if p.ContactID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: contact_id"}
	}
	return nil
}

// Execute removes a contact from the specified list.
func (a *removeFromListAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params removeFromListParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/marketing/lists/" + url.PathEscape(params.ListID) + "/contacts?contact_ids=" + url.QueryEscape(params.ContactID)

	// SendGrid returns 202 Accepted with a job_id for async deletion.
	var resp struct {
		JobID string `json:"job_id"`
	}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodDelete, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"job_id":     resp.JobID,
		"contact_id": params.ContactID,
		"status":     "accepted",
	})
}
