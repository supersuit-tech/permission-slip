package protonmail

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	netmail "net/mail"
	"net/smtp"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type sendEmailAction struct {
	conn     *ProtonMailConnector
	sendFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

type sendEmailParams struct {
	To          []string `json:"to"`
	Cc          []string `json:"cc"`
	Bcc         []string `json:"bcc"`
	Subject     string   `json:"subject"`
	Body        string   `json:"body"`
	ContentType string   `json:"content_type"`
	ReplyTo     string   `json:"reply_to"`
}

func (p *sendEmailParams) validate() error {
	if len(p.To) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: to"}
	}
	if err := validateAddresses(p.To, "to"); err != nil {
		return err
	}
	if err := validateAddresses(p.Cc, "cc"); err != nil {
		return err
	}
	if err := validateAddresses(p.Bcc, "bcc"); err != nil {
		return err
	}
	if p.ReplyTo != "" {
		if err := validateAddresses([]string{p.ReplyTo}, "reply_to"); err != nil {
			return err
		}
	}
	if p.Subject == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subject"}
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

// validateAddresses checks that all addresses are valid email addresses.
func validateAddresses(addrs []string, field string) error {
	for _, addr := range addrs {
		if addr == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("%s addresses must not be empty", field)}
		}
		if _, err := netmail.ParseAddress(addr); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s email address %q: %v", field, addr, err)}
		}
	}
	return nil
}

// sanitizeHeaderValue strips CR and LF characters to prevent header injection.
func sanitizeHeaderValue(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

func (a *sendEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendEmailParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	host, port, username, password := smtpConfig(req.Credentials)
	addr := net.JoinHostPort(host, port)

	// Build all recipients for the SMTP envelope.
	allRecipients := make([]string, 0, len(params.To)+len(params.Cc)+len(params.Bcc))
	allRecipients = append(allRecipients, params.To...)
	allRecipients = append(allRecipients, params.Cc...)
	allRecipients = append(allRecipients, params.Bcc...)

	// Build the email message.
	msg := buildMessage(username, params)

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
		"subject":    params.Subject,
	})
}

func buildMessage(from string, params sendEmailParams) []byte {
	var b strings.Builder

	b.WriteString("From: " + sanitizeHeaderValue(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeaderValue(strings.Join(params.To, ", ")) + "\r\n")
	if len(params.Cc) > 0 {
		b.WriteString("Cc: " + sanitizeHeaderValue(strings.Join(params.Cc, ", ")) + "\r\n")
	}
	if params.ReplyTo != "" {
		b.WriteString("Reply-To: " + sanitizeHeaderValue(params.ReplyTo) + "\r\n")
	}
	b.WriteString("Subject: " + sanitizeHeaderValue(params.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: " + params.ContentType + "; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(params.Body)

	return []byte(b.String())
}

// sendMailTLS connects via TLS (Proton Mail Bridge uses STARTTLS on port 1025).
func sendMailTLS(ctx context.Context, addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Use a dialer with context deadline if set.
	dialer := &net.Dialer{}
	if deadline, ok := ctx.Deadline(); ok {
		dialer.Deadline = deadline
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("SMTP connection timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("SMTP connection failed: %v", err)}
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return &connectors.ExternalError{Message: fmt.Sprintf("SMTP client creation failed: %v", err)}
	}
	defer client.Close()

	// Try STARTTLS — Proton Mail Bridge supports it.
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: host}
		if err := client.StartTLS(tlsConfig); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("STARTTLS failed: %v", err)}
		}
	}

	if err := client.Auth(auth); err != nil {
		return &connectors.AuthError{Message: fmt.Sprintf("SMTP auth failed: %v", err)}
	}
	if err := client.Mail(from); err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("SMTP MAIL FROM failed: %v", err)}
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("SMTP RCPT TO failed for %s: %v", rcpt, err)}
		}
	}
	wc, err := client.Data()
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("SMTP DATA failed: %v", err)}
	}
	if _, err := wc.Write(msg); err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("SMTP write failed: %v", err)}
	}
	if err := wc.Close(); err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("SMTP data close failed: %v", err)}
	}
	return client.Quit()
}

func mapSMTPError(err error) error {
	if err == nil {
		return nil
	}
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("SMTP operation timed out: %v", err)}
	}
	if connectors.IsAuthError(err) || connectors.IsExternalError(err) || connectors.IsTimeoutError(err) {
		return err
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "auth") || strings.Contains(errMsg, "Auth") ||
		strings.Contains(errMsg, "credential") || strings.Contains(errMsg, "535") {
		return &connectors.AuthError{Message: fmt.Sprintf("SMTP auth error: %v", err)}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("SMTP error: %v", err)}
}
