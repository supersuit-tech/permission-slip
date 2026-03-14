package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// archiveEmailAction implements connectors.Action for google.archive_email.
// It archives a Gmail thread by removing the INBOX label from all messages
// via POST /gmail/v1/users/me/threads/{id}/modify, matching Gmail's built-in
// Archive button behavior.
type archiveEmailAction struct {
	conn *GoogleConnector
}

// archiveEmailParams is the user-facing parameter schema.
type archiveEmailParams struct {
	ThreadID string `json:"thread_id"`
}

func (p *archiveEmailParams) validate() error {
	if p.ThreadID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: thread_id"}
	}
	return nil
}

// gmailModifyRequest is the Gmail API request body for threads.modify / messages.modify.
type gmailModifyRequest struct {
	RemoveLabelIDs []string `json:"removeLabelIds"`
}

// gmailThreadModifyResponse is the Gmail API response from threads.modify.
type gmailThreadModifyResponse struct {
	ID string `json:"id"`
}

// Execute archives a Gmail thread by removing the INBOX label from all messages.
func (a *archiveEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params archiveEmailParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	modifyURL := a.conn.gmailBaseURL + "/gmail/v1/users/me/threads/" + url.PathEscape(params.ThreadID) + "/modify"
	body := gmailModifyRequest{
		RemoveLabelIDs: []string{"INBOX"},
	}

	var resp gmailThreadModifyResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, modifyURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"thread_id": resp.ID,
	})
}
