package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	// maxEmailBodySize caps the decoded email body at 1 MB to prevent memory
	// issues with large messages. Bodies exceeding this are truncated.
	maxEmailBodySize = 1024 * 1024

	// maxMIMEDepth limits recursion depth when walking MIME part trees to
	// prevent stack overflow from crafted deeply-nested messages.
	maxMIMEDepth = 20
)

// readEmailAction implements connectors.Action for google.read_email.
// It fetches a single email by message ID and returns the full body,
// headers, and attachment metadata.
type readEmailAction struct {
	conn *GoogleConnector
}

// readEmailParams is the user-facing parameter schema.
type readEmailParams struct {
	MessageID string `json:"message_id"`
	Format    string `json:"format,omitempty"`
}

// validFormats are the allowed values for the format parameter.
var validFormats = map[string]bool{
	"full":     true,
	"metadata": true,
	"minimal":  true,
}

func (p *readEmailParams) validate() error {
	if p.MessageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message_id"}
	}
	if p.Format != "" && !validFormats[p.Format] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid format: %q (must be full, metadata, or minimal)", p.Format)}
	}
	return nil
}

func (p *readEmailParams) normalize() {
	if p.Format == "" {
		p.Format = "full"
	}
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
	PartID   string `json:"partId"`
	MimeType string `json:"mimeType"`
	Filename string `json:"filename"`
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
	ID          string                `json:"id"`
	ThreadID    string                `json:"thread_id"`
	From        string                `json:"from,omitempty"`
	To          string                `json:"to,omitempty"`
	Cc          string                `json:"cc,omitempty"`
	Subject     string                `json:"subject,omitempty"`
	Date        string                `json:"date,omitempty"`
	Snippet     string                `json:"snippet,omitempty"`
	Labels      []string              `json:"labels,omitempty"`
	ContentType string                `json:"content_type,omitempty"`
	Body        string                `json:"body,omitempty"`
	Attachments []gmailAttachmentInfo `json:"attachments,omitempty"`
}

// gmailAttachmentInfo describes an attachment without including its content.
type gmailAttachmentInfo struct {
	Filename     string `json:"filename,omitempty"`
	MimeType     string `json:"mime_type"`
	Size         int    `json:"size"`
	PartID       string `json:"part_id,omitempty"`
	AttachmentID string `json:"attachment_id,omitempty"`
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
	params.normalize()

	var msg gmailFullMessage
	msgURL := a.conn.gmailBaseURL + "/gmail/v1/users/me/messages/" + url.PathEscape(params.MessageID) + "?format=" + params.Format
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, msgURL, nil, &msg); err != nil {
		return nil, err
	}

	detail := emailFullDetail{
		ID:       msg.ID,
		ThreadID: msg.ThreadID,
		Snippet:  msg.Snippet,
		Labels:   msg.LabelIDs,
	}

	// Extract headers from the top-level payload (case-insensitive per RFC 5322).
	for _, h := range msg.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "from":
			detail.From = h.Value
		case "to":
			detail.To = h.Value
		case "cc":
			detail.Cc = h.Value
		case "subject":
			detail.Subject = h.Value
		case "date":
			detail.Date = h.Value
		}
	}

	// Only extract body and attachments for format=full. The metadata and
	// minimal formats return no body data or MIME parts from the Gmail API.
	if params.Format == "full" {
		body, contentType := extractBody(&msg.Payload, 0)
		if body != "" {
			detail.Body = body
			detail.ContentType = contentType
		}
		detail.Attachments = extractAttachments(&msg.Payload)
	}

	return connectors.JSONResult(detail)
}

// extractBody walks the MIME part tree and returns the best text body.
// It prefers text/plain over text/html. For multipart messages it recurses
// into alternative/mixed/related parts. Recursion is capped at maxMIMEDepth
// to prevent stack overflow from crafted deeply-nested messages.
func extractBody(part *gmailMessagePart, depth int) (body, contentType string) {
	if depth > maxMIMEDepth {
		return "", "text/plain"
	}

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
			b, ct := extractBody(p, depth+1)
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
	collectAttachments(part, &attachments, 0)
	if len(attachments) == 0 {
		return nil
	}
	return attachments
}

func collectAttachments(part *gmailMessagePart, out *[]gmailAttachmentInfo, depth int) {
	if depth > maxMIMEDepth {
		return
	}
	if part.Body.AttachmentID != "" {
		info := gmailAttachmentInfo{
			MimeType:     part.MimeType,
			Size:         part.Body.Size,
			PartID:       part.PartID,
			AttachmentID: part.Body.AttachmentID,
		}
		// Extract filename from Content-Disposition, falling back to
		// Content-Type name= (used by some older mailers).
		for _, h := range part.Headers {
			if strings.EqualFold(h.Name, "Content-Disposition") {
				if fn := parseFilename(h.Value); fn != "" {
					info.Filename = fn
				}
			}
			if info.Filename == "" && strings.EqualFold(h.Name, "Content-Type") {
				if fn := parseFilename(h.Value); fn != "" {
					info.Filename = fn
				}
			}
		}
		if info.Filename == "" && part.Filename != "" {
			info.Filename = part.Filename
		}
		*out = append(*out, info)
	}
	for i := range part.Parts {
		collectAttachments(&part.Parts[i], out, depth+1)
	}
}

// parseFilename extracts a filename from a Content-Disposition or Content-Type
// header value. Checks filename=, filename*= (RFC 5987), name=, and name*=.
func parseFilename(headerValue string) string {
	var plainName string
	for _, param := range strings.Split(headerValue, ";") {
		param = strings.TrimSpace(param)
		lower := strings.ToLower(param)

		// RFC 5987 extended filenames (filename*=UTF-8''encoded%20name)
		// take priority over plain filenames per RFC 6266.
		if strings.HasPrefix(lower, "filename*=") {
			if fn := decodeRFC5987(param[len("filename*="):]); fn != "" {
				return fn
			}
		}
		if strings.HasPrefix(lower, "name*=") {
			if fn := decodeRFC5987(param[len("name*="):]); fn != "" {
				return fn
			}
		}

		if strings.HasPrefix(lower, "filename=") {
			plainName = strings.Trim(param[len("filename="):], `"`)
		}
		if plainName == "" && strings.HasPrefix(lower, "name=") {
			plainName = strings.Trim(param[len("name="):], `"`)
		}
	}
	return plainName
}

// decodeRFC5987 decodes a RFC 5987 encoded value like "UTF-8”caf%C3%A9.pdf".
// Only UTF-8 charset is supported; other charsets (e.g. ISO-8859-1) return ""
// to avoid producing invalid UTF-8 strings.
func decodeRFC5987(value string) string {
	// Format: charset'language'encoded_value
	parts := strings.SplitN(value, "'", 3)
	if len(parts) != 3 {
		return ""
	}
	if !strings.EqualFold(parts[0], "UTF-8") {
		return ""
	}
	decoded, err := url.PathUnescape(parts[2])
	if err != nil {
		return ""
	}
	return decoded
}

// decodeBase64URL decodes a base64url-encoded string (Gmail API format).
// Gmail uses raw (no padding) base64url, so we try that first. Padded
// encoding is tried as a fallback for edge cases.
func decodeBase64URL(s string) string {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		b, err = base64.URLEncoding.DecodeString(s)
		if err != nil {
			b, err = base64.StdEncoding.DecodeString(s)
			if err != nil {
				return s // return as-is if decoding fails
			}
		}
	}
	return string(b)
}

// truncateEmailBody limits body content to maxEmailBodySize, cutting at a
// valid UTF-8 boundary to avoid splitting multi-byte runes.
func truncateEmailBody(s string) string {
	if len(s) <= maxEmailBodySize {
		return s
	}
	// Walk backwards from the limit to avoid splitting a multi-byte rune.
	i := maxEmailBodySize
	for i > 0 && !utf8.RuneStart(s[i]) {
		i--
	}
	return s[:i] + "\n[truncated]"
}
