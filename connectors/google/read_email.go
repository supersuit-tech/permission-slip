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

// maxEmailBodySize caps the decoded email body at 1 MB to prevent memory
// issues with large messages. Bodies exceeding this are truncated.
const maxEmailBodySize = 1024 * 1024

// readEmailAction implements connectors.Action for google.read_email.
// It fetches a single email by message ID and returns the full body,
// headers, and attachment metadata.
type readEmailAction struct {
	conn *GoogleConnector
}

// readEmailParams is the user-facing parameter schema.
type readEmailParams struct {
	MessageID string `json:"message_id"`
}

func (p *readEmailParams) validate() error {
	if p.MessageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message_id"}
	}
	return nil
}

// gmailFullMessage is the Gmail API response from messages.get with format=full.
type gmailFullMessage struct {
	ID           string           `json:"id"`
	ThreadID     string           `json:"threadId"`
	LabelIDs     []string         `json:"labelIds"`
	Snippet      string           `json:"snippet"`
	InternalDate string           `json:"internalDate"`
	Payload      gmailMessagePart `json:"payload"`
}

// gmailMessagePart represents a MIME message part in the Gmail API response.
type gmailMessagePart struct {
	MimeType string `json:"mimeType"`
	Headers  []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"headers"`
	Body struct {
		AttachmentID string `json:"attachmentId"`
		Size         int    `json:"size"`
		Data         string `json:"data"`
	} `json:"body"`
	Parts []gmailMessagePart `json:"parts"`
}

// emailFullDetail is the shape returned to the agent.
type emailFullDetail struct {
	ID          string               `json:"id"`
	ThreadID    string               `json:"thread_id"`
	From        string               `json:"from,omitempty"`
	To          string               `json:"to,omitempty"`
	Cc          string               `json:"cc,omitempty"`
	Subject     string               `json:"subject,omitempty"`
	Date        string               `json:"date,omitempty"`
	Labels      []string             `json:"labels,omitempty"`
	ContentType string               `json:"content_type"`
	Body        string               `json:"body"`
	Attachments []gmailAttachmentInfo `json:"attachments,omitempty"`
}

// gmailAttachmentInfo describes an attachment without including its content.
type gmailAttachmentInfo struct {
	Filename    string `json:"filename"`
	MimeType    string `json:"mime_type"`
	Size        int    `json:"size"`
	PartID      string `json:"part_id,omitempty"`
}

// Execute fetches a single email by message ID and returns its full content.
func (a *readEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params readEmailParams
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

	detail := emailFullDetail{
		ID:       msg.ID,
		ThreadID: msg.ThreadID,
		Labels:   msg.LabelIDs,
	}

	// Extract headers from the top-level payload.
	for _, h := range msg.Payload.Headers {
		switch h.Name {
		case "From":
			detail.From = h.Value
		case "To":
			detail.To = h.Value
		case "Cc":
			detail.Cc = h.Value
		case "Subject":
			detail.Subject = h.Value
		case "Date":
			detail.Date = h.Value
		}
	}

	// Extract body and attachments from the MIME tree.
	body, contentType := extractBody(&msg.Payload)
	detail.Body = body
	detail.ContentType = contentType
	detail.Attachments = extractAttachments(&msg.Payload)

	return connectors.JSONResult(detail)
}

// extractBody walks the MIME part tree and returns the best text body.
// It prefers text/plain over text/html. For multipart messages it recurses
// into alternative/mixed/related parts.
func extractBody(part *gmailMessagePart) (body, contentType string) {
	// Single-part message with body data.
	if part.Body.Data != "" && strings.HasPrefix(part.MimeType, "text/") {
		decoded := decodeBase64URL(part.Body.Data)
		return truncateEmailBody(decoded), part.MimeType
	}

	// Multipart: recurse into parts.
	var htmlBody string
	for i := range part.Parts {
		p := &part.Parts[i]
		if strings.HasPrefix(p.MimeType, "multipart/") {
			b, ct := extractBody(p)
			if b != "" {
				if ct == "text/plain" {
					return b, ct
				}
				htmlBody = b
				contentType = ct
			}
			continue
		}
		if p.Body.Data != "" && p.MimeType == "text/plain" {
			return truncateEmailBody(decodeBase64URL(p.Body.Data)), "text/plain"
		}
		if p.Body.Data != "" && p.MimeType == "text/html" {
			htmlBody = truncateEmailBody(decodeBase64URL(p.Body.Data))
			contentType = "text/html"
		}
	}

	if htmlBody != "" {
		return htmlBody, contentType
	}
	return "", "text/plain"
}

// extractAttachments collects attachment metadata from the MIME part tree.
func extractAttachments(part *gmailMessagePart) []gmailAttachmentInfo {
	var attachments []gmailAttachmentInfo
	collectAttachments(part, &attachments)
	if len(attachments) == 0 {
		return nil
	}
	return attachments
}

func collectAttachments(part *gmailMessagePart, out *[]gmailAttachmentInfo) {
	if part.Body.AttachmentID != "" {
		info := gmailAttachmentInfo{
			MimeType: part.MimeType,
			Size:     part.Body.Size,
		}
		// Extract filename from Content-Disposition or part headers.
		for _, h := range part.Headers {
			if strings.EqualFold(h.Name, "Content-Disposition") {
				if fn := parseFilename(h.Value); fn != "" {
					info.Filename = fn
				}
			}
		}
		*out = append(*out, info)
	}
	for i := range part.Parts {
		collectAttachments(&part.Parts[i], out)
	}
}

// parseFilename extracts a filename from a Content-Disposition header value.
func parseFilename(disposition string) string {
	for _, param := range strings.Split(disposition, ";") {
		param = strings.TrimSpace(param)
		if strings.HasPrefix(strings.ToLower(param), "filename=") {
			name := param[len("filename="):]
			return strings.Trim(name, `"`)
		}
	}
	return ""
}

// decodeBase64URL decodes a base64url-encoded string (Gmail API format).
func decodeBase64URL(s string) string {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		// Gmail uses raw (no padding) base64url.
		b, err = base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			return s // return as-is if decoding fails
		}
	}
	return string(b)
}

// truncateEmailBody limits body content to maxEmailBodySize.
func truncateEmailBody(s string) string {
	if len(s) > maxEmailBodySize {
		return s[:maxEmailBodySize] + "\n[truncated]"
	}
	return s
}
