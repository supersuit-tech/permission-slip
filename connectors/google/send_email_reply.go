package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// sendEmailReplyAction implements connectors.Action for google.send_email_reply.
// It replies to an existing Gmail thread by fetching the original message
// headers and sending a new message in the same thread.
type sendEmailReplyAction struct {
	conn *GoogleConnector
}

// sendEmailReplyParams is the user-facing parameter schema.
type sendEmailReplyParams struct {
	ThreadID  string `json:"thread_id"`
	MessageID string `json:"message_id"`
	Body      string `json:"body"`
	HTML      *bool  `json:"html,omitempty"`
}

func (p *sendEmailReplyParams) validate() error {
	if p.ThreadID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: thread_id"}
	}
	if p.MessageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message_id"}
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

// gmailSendReplyRequest is the Gmail API request body for sending a reply.
type gmailSendReplyRequest struct {
	Raw      string `json:"raw"`
	ThreadID string `json:"threadId"`
}

// stripHeaderNewlines removes CR and LF from a Gmail-sourced header value to
// prevent MIME header injection. Unlike user-supplied headers (which are
// rejected on newlines), Gmail-provided values are silently sanitized because
// the agent cannot control the original sender's content.
func stripHeaderNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r", ""), "\n", "")
}

// Execute fetches the original message headers and sends a reply in the same thread.
func (a *sendEmailReplyAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendEmailReplyParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Fetch the original message to extract From, Subject, and Message-ID headers.
	// Request both Message-Id and Message-ID since the casing varies by provider.
	var origMsg gmailMessageResponse
	msgURL := a.conn.gmailBaseURL + "/gmail/v1/users/me/messages/" + url.PathEscape(params.MessageID) +
		"?format=metadata&metadataHeaders=From&metadataHeaders=Subject&metadataHeaders=Message-Id&metadataHeaders=Message-ID"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, msgURL, nil, &origMsg); err != nil {
		return nil, err
	}

	// Verify the message belongs to the expected thread.
	if origMsg.ThreadID != params.ThreadID {
		return nil, &connectors.ValidationError{Message: "message_id does not belong to specified thread_id"}
	}

	var origFrom, origSubject, origMessageID string
	for _, h := range origMsg.Payload.Headers {
		// Parse header names case-insensitively to handle provider variations.
		switch strings.ToLower(h.Name) {
		case "from":
			origFrom = h.Value
		case "subject":
			origSubject = h.Value
		case "message-id":
			origMessageID = h.Value
		}
	}

	if origFrom == "" {
		return nil, &connectors.ExternalError{Message: "could not determine reply-to address from original message; check message_id"}
	}

	// Sanitize Gmail-sourced header values to prevent MIME header injection.
	origFrom = stripHeaderNewlines(origFrom)
	origSubject = stripHeaderNewlines(origSubject)
	origMessageID = stripHeaderNewlines(origMessageID)

	// Build Re: subject if not already prefixed.
	replySubject := origSubject
	if !strings.HasPrefix(strings.ToLower(replySubject), "re:") {
		replySubject = "Re: " + replySubject
	}

	// Build and encode the RFC 2822 reply using the shared helper.
	raw := buildGmailRaw(origFrom, replySubject, params.Body, emailHTMLDefault(params.HTML), [][2]string{
		{"In-Reply-To", origMessageID},
		{"References", origMessageID},
	})

	body := gmailSendReplyRequest{
		Raw:      raw,
		ThreadID: params.ThreadID,
	}

	var resp gmailSendResponse
	sendURL := a.conn.gmailBaseURL + "/gmail/v1/users/me/messages/send"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, sendURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":        resp.ID,
		"thread_id": resp.ThreadID,
		"subject":   replySubject,
		"to":        origFrom,
	})
}
