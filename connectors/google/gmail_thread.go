package google

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// gmailThreadAPIResponse is the Gmail threads.get API response (format=full).
type gmailThreadAPIResponse struct {
	ID       string             `json:"id"`
	Messages []gmailFullMessage `json:"messages"`
}

// buildGmailEmailThread fetches the full thread and returns a normalized email_thread payload.
func (c *GoogleConnector) buildGmailEmailThread(ctx context.Context, creds connectors.Credentials, threadID string) (*connectors.EmailThreadPayload, error) {
	if threadID == "" {
		return nil, fmt.Errorf("missing thread_id")
	}
	threadURL := c.gmailBaseURL + "/gmail/v1/users/me/threads/" + url.PathEscape(threadID) + "?format=full"
	var thr gmailThreadAPIResponse
	if err := c.doJSON(ctx, creds, http.MethodGet, threadURL, nil, &thr); err != nil {
		return nil, err
	}
	if len(thr.Messages) == 0 {
		return &connectors.EmailThreadPayload{Subject: "", Messages: nil}, nil
	}

	// Oldest → newest by internalDate (fallback: preserve API order).
	msgs := append([]gmailFullMessage(nil), thr.Messages...)
	sort.Slice(msgs, func(i, j int) bool {
		ai, _ := strconv.ParseInt(msgs[i].InternalDate, 10, 64)
		aj, _ := strconv.ParseInt(msgs[j].InternalDate, 10, 64)
		return ai < aj
	})

	subject := ""
	for _, m := range msgs {
		if s := headerValue(&m.Payload, "Subject"); s != "" {
			subject = stripHeaderNewlines(s)
			break
		}
	}

	out := make([]connectors.EmailThreadMessage, 0, len(msgs))
	for _, m := range msgs {
		em := connectors.EmailThreadMessage{
			From:      stripHeaderNewlines(headerValue(&m.Payload, "From")),
			To:        splitAddressList(headerValue(&m.Payload, "To")),
			Cc:        splitAddressList(headerValue(&m.Payload, "Cc")),
			Date:      gmailMessageDateRFC3339(&m),
			Snippet:   m.Snippet,
			MessageID: m.ID,
		}
		plain, html := extractHTMLAndPlain(&m.Payload, 0)
		em.BodyText = plain
		em.BodyHTML = html
		if em.BodyText == "" && em.BodyHTML != "" {
			em.BodyText = htmlToPlainText(em.BodyHTML)
		}
		em.Attachments = gmailAttachmentsToThread(extractAttachments(&m.Payload))
		connectors.TruncateEmailThreadBodies(&em)
		out = append(out, em)
	}

	return &connectors.EmailThreadPayload{
		Subject:  subject,
		Messages: out,
	}, nil
}

func headerValue(part *gmailMessagePart, name string) string {
	if part == nil {
		return ""
	}
	lower := strings.ToLower(name)
	for _, h := range part.Headers {
		if strings.ToLower(h.Name) == lower {
			return h.Value
		}
	}
	return ""
}

func splitAddressList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, stripHeaderNewlines(p))
		}
	}
	return out
}

func gmailMessageDateRFC3339(m *gmailFullMessage) string {
	if m.InternalDate != "" {
		if ms, err := strconv.ParseInt(m.InternalDate, 10, 64); err == nil {
			return time.UnixMilli(ms).UTC().Format(time.RFC3339)
		}
	}
	if d := headerValue(&m.Payload, "Date"); d != "" {
		return stripHeaderNewlines(d)
	}
	return ""
}

// extractHTMLAndPlain walks the MIME tree and returns best-effort text/plain and text/html bodies.
func extractHTMLAndPlain(part *gmailMessagePart, depth int) (plain, html string) {
	if part == nil || depth > maxMIMEDepth {
		return "", ""
	}

	if part.Body.Data != "" && part.MimeType == "text/plain" {
		return decodeBase64URL(part.Body.Data), ""
	}
	if part.Body.Data != "" && part.MimeType == "text/html" {
		return "", decodeBase64URL(part.Body.Data)
	}

	if strings.HasPrefix(part.MimeType, "multipart/") {
		var plainOut, htmlOut string
		for i := range part.Parts {
			p, h := extractHTMLAndPlain(&part.Parts[i], depth+1)
			if p != "" {
				plainOut = p
			}
			if h != "" {
				htmlOut = h
			}
		}
		return plainOut, htmlOut
	}

	return "", ""
}

func gmailAttachmentsToThread(in []gmailAttachmentInfo) []connectors.EmailThreadAttachment {
	if len(in) == 0 {
		return nil
	}
	out := make([]connectors.EmailThreadAttachment, 0, len(in))
	for _, a := range in {
		fn := a.Filename
		if fn == "" {
			fn = "(no filename)"
		}
		out = append(out, connectors.EmailThreadAttachment{
			Filename:  fn,
			SizeBytes: int64(a.Size),
		})
	}
	return out
}
