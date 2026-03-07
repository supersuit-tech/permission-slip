package protonmail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-message/mail"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const maxBodySize = 1024 * 1024 // 1 MB body limit

type readEmailAction struct {
	conn *ProtonMailConnector
}

type readEmailParams struct {
	MessageID uint32 `json:"message_id"`
	Folder    string `json:"folder"`
}

func (p *readEmailParams) validate() error {
	if p.MessageID == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: message_id"}
	}
	if p.Folder == "" {
		p.Folder = "INBOX"
	}
	return nil
}

type emailDetail struct {
	SeqNum      uint32           `json:"seq_num"`
	Subject     string           `json:"subject"`
	From        []string         `json:"from"`
	To          []string         `json:"to"`
	Cc          []string         `json:"cc,omitempty"`
	ReplyTo     []string         `json:"reply_to,omitempty"`
	Date        string           `json:"date"`
	MessageID   string           `json:"message_id_header,omitempty"`
	Flags       []string         `json:"flags"`
	ContentType string           `json:"content_type"`
	Body        string           `json:"body"`
	Attachments []attachmentInfo `json:"attachments,omitempty"`
}

type attachmentInfo struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        uint32 `json:"size"`
}

func (a *readEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params readEmailParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	session, err := connectIMAP(req.Credentials, a.conn.timeout)
	if err != nil {
		return nil, err
	}
	defer session.close()

	mboxData, err := session.selectMailbox(params.Folder)
	if err != nil {
		return nil, err
	}

	if params.MessageID > mboxData.NumMessages {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("message_id %d not found (mailbox has %d messages)", params.MessageID, mboxData.NumMessages),
		}
	}

	// Fetch the full message body.
	var seqSet imap.SeqSet
	seqSet.AddNum(params.MessageID)

	bodySection := &imap.FetchItemBodySection{
		Peek: true, // don't mark as read
	}

	fetchCmd := session.client.Fetch(seqSet, &imap.FetchOptions{
		Envelope: true,
		Flags:    true,
		BodySection: []*imap.FetchItemBodySection{
			bodySection,
		},
		BodyStructure: &imap.FetchItemBodyStructure{Extended: true},
	})
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("message %d not found", params.MessageID)}
	}

	buf, err := msg.Collect()
	if err != nil {
		return nil, mapIMAPError(err)
	}

	detail := emailDetail{
		SeqNum: buf.SeqNum,
	}

	if buf.Envelope != nil {
		detail.Subject = buf.Envelope.Subject
		detail.Date = buf.Envelope.Date.Format(time.RFC3339)
		detail.MessageID = buf.Envelope.MessageID
		detail.From = formatAddresses(buf.Envelope.From)
		detail.To = formatAddresses(buf.Envelope.To)
		detail.Cc = formatAddresses(buf.Envelope.Cc)
		detail.ReplyTo = formatAddresses(buf.Envelope.ReplyTo)
	}

	for _, f := range buf.Flags {
		detail.Flags = append(detail.Flags, string(f))
	}

	// Collect attachment info from body structure.
	if buf.BodyStructure != nil {
		buf.BodyStructure.Walk(func(path []int, part imap.BodyStructure) bool {
			if sp, ok := part.(*imap.BodyStructureSinglePart); ok {
				if fn := sp.Filename(); fn != "" {
					detail.Attachments = append(detail.Attachments, attachmentInfo{
						Filename:    fn,
						ContentType: sp.MediaType(),
						Size:        sp.Size,
					})
				}
			}
			return true
		})
	}

	// Parse the body content.
	rawBody := buf.FindBodySection(bodySection)
	if rawBody != nil {
		body, contentType := parseBody(rawBody)
		detail.Body = body
		detail.ContentType = contentType
	}

	return connectors.JSONResult(detail)
}

// parseBody extracts the text body from a raw RFC 5322 message.
func parseBody(rawBody []byte) (body, contentType string) {
	mr, err := mail.CreateReader(bytes.NewReader(rawBody))
	if err != nil {
		// Fall back to raw body if we can't parse it.
		return truncateBody(string(rawBody)), "text/plain"
	}
	defer mr.Close()

	contentType = "text/plain"
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		switch part.Header.(type) {
		case *mail.InlineHeader:
			h := part.Header.(*mail.InlineHeader)
			ct, _, _ := h.ContentType()
			if strings.HasPrefix(ct, "text/plain") || strings.HasPrefix(ct, "text/html") {
				// Read maxBodySize+1 to detect if content was truncated.
				b, err := io.ReadAll(io.LimitReader(part.Body, maxBodySize+1))
				if err == nil {
					if len(b) > maxBodySize {
						body = string(b[:maxBodySize]) + "\n[truncated]"
					} else {
						body = string(b)
					}
					contentType = ct
				}
				if strings.HasPrefix(ct, "text/plain") {
					// Prefer text/plain; if found, stop looking.
					return body, contentType
				}
			}
		}
	}

	return body, contentType
}

func truncateBody(s string) string {
	if len(s) > maxBodySize {
		return s[:maxBodySize] + "\n[truncated]"
	}
	return s
}
