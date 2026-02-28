package notify

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
)

// SMTPSender sends email notifications via a generic SMTP server.
// Supports STARTTLS on port 587 and plain SMTP on port 25.
type SMTPSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

// NewSMTPSender creates an SMTPSender with the given configuration.
func NewSMTPSender(host, port, username, password, from string) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPSender) Name() string { return "email" }

func (s *SMTPSender) Send(ctx context.Context, approval Approval, recipient Recipient) error {
	if recipient.Email == nil || *recipient.Email == "" {
		return fmt.Errorf("recipient %s has no email address", recipient.UserID)
	}

	to := *recipient.Email
	if strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("recipient %s email contains invalid characters", recipient.UserID)
	}
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("recipient %s has invalid email address: %w", recipient.UserID, err)
	}

	subject := buildEmailSubject(approval)
	plainBody := buildEmailPlainBody(approval)
	htmlBody := buildEmailHTMLBody(approval)

	msg := buildMIMEMessage(s.from, to, subject, plainBody, htmlBody)

	return s.sendSMTP(ctx, to, []byte(msg))
}

// sendSMTP performs the SMTP transaction with context-aware dialing and
// deadlines. This ensures context cancellation actually stops the in-flight
// connection rather than leaving a leaked goroutine.
func (s *SMTPSender) sendSMTP(ctx context.Context, to string, msg []byte) error {
	addr := net.JoinHostPort(s.host, s.port)

	// Context-aware dial — cancellation immediately aborts the connection.
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	// Propagate context deadline to the connection so the SMTP conversation
	// is bounded even after the dial succeeds.
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("smtp set deadline: %w", err)
		}
	}

	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer c.Close()

	// STARTTLS when available (port 587 is the standard submission port).
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if s.username != "" {
		if err := c.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := c.Mail(s.from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return c.Quit()
}

// buildMIMEMessage constructs a multipart/alternative MIME message with both
// plain-text and HTML parts.
func buildMIMEMessage(from, to, subject, plainBody, htmlBody string) string {
	boundary := generateMIMEBoundary()

	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s\r\n", from))
	b.WriteString(fmt.Sprintf("To: %s\r\n", to))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject)))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	b.WriteString("\r\n")

	// Plain-text part
	b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")
	b.WriteString(quotedPrintableEncode(plainBody))
	b.WriteString("\r\n")

	// HTML part
	b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")
	b.WriteString(quotedPrintableEncode(htmlBody))
	b.WriteString("\r\n")

	// Closing boundary
	b.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return b.String()
}

// quotedPrintableEncode encodes s using MIME quoted-printable encoding.
func quotedPrintableEncode(s string) string {
	var buf strings.Builder
	w := quotedprintable.NewWriter(&buf)
	// quotedprintable.Writer.Write and Close only return errors if the
	// underlying writer does; strings.Builder never fails.
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	return buf.String()
}

// generateMIMEBoundary returns a random MIME boundary string. Using a random
// boundary avoids the (unlikely) possibility of email body content colliding
// with a static boundary, which would break the MIME structure.
func generateMIMEBoundary() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// Fallback: not random, but still functional. This should never
		// happen — crypto/rand uses the OS CSPRNG.
		return "----=_PermissionSlipBoundary"
	}
	return "----=_PS_" + hex.EncodeToString(buf[:])
}
