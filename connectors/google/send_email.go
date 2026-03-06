package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendEmailAction implements connectors.Action for google.send_email.
// It sends an email via the Gmail API POST /gmail/v1/users/me/messages/send.
type sendEmailAction struct {
	conn *GoogleConnector
}

// sendEmailParams is the user-facing parameter schema.
type sendEmailParams struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (p *sendEmailParams) validate() error {
	if p.To == "" {
		return &connectors.ValidationError{Message: "missing required parameter: to"}
	}
	if p.Subject == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subject"}
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	if containsNewline(p.To) {
		return &connectors.ValidationError{Message: "to must not contain newlines"}
	}
	if containsNewline(p.Subject) {
		return &connectors.ValidationError{Message: "subject must not contain newlines"}
	}
	return nil
}

// containsNewline returns true if s contains CR or LF characters, which
// would allow MIME header injection in RFC 2822 messages.
func containsNewline(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}

// gmailSendRequest is the Gmail API request body for messages.send.
type gmailSendRequest struct {
	Raw string `json:"raw"`
}

// gmailSendResponse is the Gmail API response from messages.send.
type gmailSendResponse struct {
	ID       string   `json:"id"`
	ThreadID string   `json:"threadId"`
	LabelIDs []string `json:"labelIds"`
}

// Execute sends an email via Gmail and returns the message metadata.
func (a *sendEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendEmailParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build RFC 2822 message and base64url-encode it for the Gmail API.
	var msg strings.Builder
	msg.WriteString("To: " + params.To + "\r\n")
	msg.WriteString("Subject: " + params.Subject + "\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(params.Body)

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	body := gmailSendRequest{Raw: raw}
	var resp gmailSendResponse

	url := a.conn.gmailBaseURL + "/gmail/v1/users/me/messages/send"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, url, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":        resp.ID,
		"thread_id": resp.ThreadID,
	})
}
