package protonmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	netmail "net/mail"
	"net/smtp"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// resolveSourceForReply is the IMAP-backed lookup for reply threading. Tests may
// replace it to avoid a live proxy.
var resolveSourceForReply = resolveSourceForReplyIMAP

type replyEmailAction struct {
	conn     *ProtonMailConnector
	sendFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

type replyEmailParams struct {
	InReplyToMessageID uint32   `json:"in_reply_to_message_id"`
	Folder             string   `json:"folder"`
	To                 []string `json:"to"`
	Cc                 []string `json:"cc"`
	Bcc                []string `json:"bcc"`
	Subject            string   `json:"subject"`
	Body               string   `json:"body"`
	ContentType        string   `json:"content_type"`
}

func (p *replyEmailParams) validate() error {
	if p.InReplyToMessageID == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: in_reply_to_message_id"}
	}
	if p.Folder == "" {
		p.Folder = "INBOX"
	}
	if len(p.To) > 0 {
		if err := validateAddresses(p.To, "to"); err != nil {
			return err
		}
	}
	if err := validateAddresses(p.Cc, "cc"); err != nil {
		return err
	}
	if err := validateAddresses(p.Bcc, "bcc"); err != nil {
		return err
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	if p.ContentType == "" {
		p.ContentType = "text/plain"
	}
	if p.ContentType != "text/plain" && p.ContentType != "text/html" {
		return &connectors.ValidationError{Message: "content_type must be text/plain or text/html"}
	}
	return nil
}

type replySource struct {
	From            []string
	Subject         string
	MessageIDHeader string
}

func (a *replyEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params replyEmailParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	source, err := resolveSourceForReply(ctx, a.conn, req.Credentials, params.Folder, params.InReplyToMessageID, req.MailboxUIDValidity)
	if err != nil {
		return nil, err
	}
	if source.MessageIDHeader == "" {
		return nil, &connectors.ExternalError{Message: "could not determine Message-ID from source email for reply threading"}
	}

	to := params.To
	if len(to) == 0 {
		if len(source.From) == 0 {
			return nil, &connectors.ExternalError{Message: "could not determine reply recipient from source email; provide to explicitly"}
		}
		to = source.From
	}

	subject := params.Subject
	if subject == "" {
		subject = source.Subject
		if !strings.HasPrefix(strings.ToLower(subject), "re:") {
			subject = "Re: " + subject
		}
	}

	host, port, username, password := smtpConfig(req.Credentials)
	addr := net.JoinHostPort(host, port)

	allRecipients := make([]string, 0, len(to)+len(params.Cc)+len(params.Bcc))
	for _, raw := range append(append(to, params.Cc...), params.Bcc...) {
		parsed, _ := netmail.ParseAddress(raw)
		allRecipients = append(allRecipients, parsed.Address)
	}

	msg := buildReplyMessage(username, to, params.Cc, params.Bcc, subject, params.Body, params.ContentType, source.MessageIDHeader)
	auth := smtp.PlainAuth("", username, password, host)

	if a.sendFunc != nil {
		if err := a.sendFunc(addr, auth, username, allRecipients, msg); err != nil {
			return nil, mapSMTPError(err)
		}
	} else {
		if err := sendMailTLS(ctx, addr, host, auth, username, allRecipients, msg); err != nil {
			return nil, mapSMTPError(err)
		}
	}

	return connectors.JSONResult(map[string]any{
		"status":     "sent",
		"from":       username,
		"recipients": allRecipients,
		"subject":    subject,
	})
}

func buildReplyMessage(from string, to, cc, bcc []string, subject, body, contentType, messageIDHeader string) []byte {
	var b strings.Builder

	b.WriteString("From: " + sanitizeHeaderValue(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeaderValue(strings.Join(to, ", ")) + "\r\n")
	if len(cc) > 0 {
		b.WriteString("Cc: " + sanitizeHeaderValue(strings.Join(cc, ", ")) + "\r\n")
	}
	b.WriteString("Subject: " + sanitizeHeaderValue(subject) + "\r\n")
	b.WriteString("In-Reply-To: " + sanitizeHeaderValue(messageIDHeader) + "\r\n")
	b.WriteString("References: " + sanitizeHeaderValue(messageIDHeader) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: " + contentType + "; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)

	_ = bcc // BCC recipients are envelope-only; not included in message headers.
	return []byte(b.String())
}

func resolveSourceForReplyIMAP(ctx context.Context, conn *ProtonMailConnector, creds connectors.Credentials, folder string, uid uint32, store connectors.MailboxUIDValidityStore) (*replySource, error) {
	metaByUID, err := resolveMessageEnvelopesWithMessageID(ctx, conn, creds, folder, []uint32{uid}, store)
	if err != nil {
		return nil, err
	}
	meta, ok := metaByUID[uid]
	if !ok {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("message uid %d not found in folder %q", uid, folder),
		}
	}
	return &replySource{
		From:            meta.From,
		Subject:         meta.Subject,
		MessageIDHeader: meta.MessageIDHeader,
	}, nil
}
