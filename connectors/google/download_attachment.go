package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// maxAttachmentBytes is the maximum decoded Gmail attachment we will return
// (10 MB). Larger attachments are rejected with a validation error rather
// than being truncated.
const maxAttachmentBytes = 10 * 1024 * 1024

// maxAttachmentJSONBytes covers a 10 MB attachment after base64 expansion
// plus JSON envelope overhead for messages.attachments.get.
const maxAttachmentJSONBytes = int64(maxAttachmentBytes)*4/3 + 64*1024

// downloadAttachmentAction implements connectors.Action for google.download_attachment.
// It fetches a single Gmail attachment by message ID and attachment ID.
type downloadAttachmentAction struct {
	conn *GoogleConnector
}

type downloadAttachmentParams struct {
	MessageID    string `json:"message_id"`
	AttachmentID string `json:"attachment_id"`
}

func (p *downloadAttachmentParams) validate() error {
	p.MessageID = strings.TrimSpace(p.MessageID)
	if p.MessageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message_id"}
	}
	p.AttachmentID = strings.TrimSpace(p.AttachmentID)
	if p.AttachmentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: attachment_id"}
	}
	return nil
}

type downloadAttachmentResult struct {
	MessageID     string `json:"message_id"`
	AttachmentID  string `json:"attachment_id"`
	Filename      string `json:"filename,omitempty"`
	MimeType      string `json:"mime_type"`
	Size          int    `json:"size"`
	ContentBase64 string `json:"content_base64"`
}

// gmailAttachment is the Gmail API messages.attachments.get response.
type gmailAttachment struct {
	Size int    `json:"size"`
	Data string `json:"data"`
}

// Execute downloads one Gmail attachment and returns its bytes as standard base64.
func (a *downloadAttachmentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params downloadAttachmentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var msg gmailFullMessage
	msgURL := a.conn.gmailBaseURL + "/gmail/v1/users/me/messages/" + url.PathEscape(params.MessageID) + "?format=full"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, msgURL, nil, &msg); err != nil {
		return nil, err
	}

	info := findGmailAttachment(&msg.Payload, params.AttachmentID)
	if info == nil || info.AttachmentID == "" {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("attachment %q not found on message %q", params.AttachmentID, params.MessageID),
		}
	}
	if info.Size > maxAttachmentBytes {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("attachment %q exceeds maximum size of 10 MB", params.AttachmentID),
		}
	}

	var att gmailAttachment
	attURL := a.conn.gmailBaseURL + "/gmail/v1/users/me/messages/" + url.PathEscape(params.MessageID) + "/attachments/" + url.PathEscape(info.AttachmentID)
	if err := a.conn.doJSONLimit(ctx, req.Credentials, http.MethodGet, attURL, nil, &att, maxAttachmentJSONBytes); err != nil {
		return nil, err
	}
	if att.Size > maxAttachmentBytes {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("attachment %q exceeds maximum size of 10 MB", params.AttachmentID),
		}
	}

	decoded, err := decodeBase64Bytes(att.Data)
	if err != nil {
		return nil, &connectors.ExternalError{Message: "failed to decode Gmail attachment data"}
	}
	if len(decoded) > maxAttachmentBytes {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("attachment %q exceeds maximum size of 10 MB", params.AttachmentID),
		}
	}

	mimeType := info.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return connectors.JSONResult(downloadAttachmentResult{
		MessageID:     params.MessageID,
		AttachmentID:  info.AttachmentID,
		Filename:      info.Filename,
		MimeType:      mimeType,
		Size:          len(decoded),
		ContentBase64: base64.StdEncoding.EncodeToString(decoded),
	})
}

// findGmailAttachment locates attachment metadata by Gmail attachment_id or MIME part_id.
func findGmailAttachment(part *gmailMessagePart, id string) *gmailAttachmentInfo {
	for _, att := range extractAttachments(part) {
		if att.AttachmentID == id || att.PartID == id {
			found := att
			return &found
		}
	}
	return nil
}
