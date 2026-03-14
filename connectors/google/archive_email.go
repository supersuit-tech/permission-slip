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
// It archives a Gmail message by removing the INBOX label via
// POST /gmail/v1/users/me/messages/{id}/modify.
type archiveEmailAction struct {
	conn *GoogleConnector
}

// archiveEmailParams is the user-facing parameter schema.
type archiveEmailParams struct {
	MessageID string `json:"message_id"`
}

func (p *archiveEmailParams) validate() error {
	if p.MessageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message_id"}
	}
	return nil
}

// gmailModifyRequest is the Gmail API request body for messages.modify.
type gmailModifyRequest struct {
	RemoveLabelIDs []string `json:"removeLabelIds"`
}

// gmailModifyResponse is the Gmail API response from messages.modify.
type gmailModifyResponse struct {
	ID       string   `json:"id"`
	ThreadID string   `json:"threadId"`
	LabelIDs []string `json:"labelIds"`
}

// Execute archives a Gmail message by removing the INBOX label.
func (a *archiveEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params archiveEmailParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	modifyURL := a.conn.gmailBaseURL + "/gmail/v1/users/me/messages/" + url.PathEscape(params.MessageID) + "/modify"
	body := gmailModifyRequest{
		RemoveLabelIDs: []string{"INBOX"},
	}

	var resp gmailModifyResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, modifyURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"id":        resp.ID,
		"thread_id": resp.ThreadID,
		"labels":    resp.LabelIDs,
	})
}
