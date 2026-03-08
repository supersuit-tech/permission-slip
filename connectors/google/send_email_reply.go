package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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

// Execute fetches the original message headers and sends a reply in the same thread.
func (a *sendEmailReplyAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendEmailReplyParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Fetch the original message to extract From, Subject, and Message-Id headers.
	var origMsg gmailMessageResponse
	msgURL := a.conn.gmailBaseURL + "/gmail/v1/users/me/messages/" + url.PathEscape(params.MessageID) +
		"?format=metadata&metadataHeaders=From&metadataHeaders=Subject&metadataHeaders=Message-Id"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, msgURL, nil, &origMsg); err != nil {
		return nil, err
	}

	var origFrom, origSubject, origMessageID string
	for _, h := range origMsg.Payload.Headers {
		switch h.Name {
		case "From":
			origFrom = h.Value
		case "Subject":
			origSubject = h.Value
		case "Message-Id":
			origMessageID = h.Value
		}
	}

	if origFrom == "" {
		return nil, &connectors.ExternalError{Message: "could not determine reply-to address from original message; check message_id"}
	}

	// Strip newlines from header values sourced from the original message to
	// prevent MIME header injection if the original sender crafted a malicious From/Subject.
	origFrom = strings.ReplaceAll(strings.ReplaceAll(origFrom, "\r", ""), "\n", "")
	origSubject = strings.ReplaceAll(strings.ReplaceAll(origSubject, "\r", ""), "\n", "")
	origMessageID = strings.ReplaceAll(strings.ReplaceAll(origMessageID, "\r", ""), "\n", "")

	// Build Re: subject if not already prefixed.
	replySubject := origSubject
	if !strings.HasPrefix(strings.ToLower(replySubject), "re:") {
		replySubject = "Re: " + replySubject
	}

	// Build RFC 2822 reply message.
	var msg strings.Builder
	msg.WriteString("To: " + origFrom + "\r\n")
	msg.WriteString("Subject: " + replySubject + "\r\n")
	if origMessageID != "" {
		msg.WriteString("In-Reply-To: " + origMessageID + "\r\n")
		msg.WriteString("References: " + origMessageID + "\r\n")
	}
	msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(params.Body)

	raw := base64.RawURLEncoding.EncodeToString([]byte(msg.String()))

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
