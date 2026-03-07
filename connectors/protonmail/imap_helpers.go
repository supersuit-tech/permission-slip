package protonmail

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// imapDial is a package-level variable so tests can replace it.
var imapDial = func(addr string, _ *imapclient.Options) (*imapclient.Client, error) {
	return imapclient.DialInsecure(addr, nil)
}

// imapSession holds an authenticated IMAP session with a selected mailbox.
type imapSession struct {
	client *imapclient.Client
}

// connectIMAP creates an authenticated IMAP connection.
func connectIMAP(ctx context.Context, creds connectors.Credentials, timeout time.Duration) (*imapSession, error) {
	host, port, username, password := imapConfig(creds)
	addr := net.JoinHostPort(host, port)

	_ = timeout // dial timeout is handled by imapclient defaults
	_ = ctx     // context not supported by imapclient.Dial*

	client, err := imapDial(addr, nil)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("IMAP connection timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("IMAP connection failed: %v", err)}
	}

	if err := client.Login(username, password).Wait(); err != nil {
		client.Close()
		return nil, &connectors.AuthError{Message: fmt.Sprintf("IMAP login failed: %v", err)}
	}

	return &imapSession{client: client}, nil
}

// selectMailbox selects a mailbox (e.g., "INBOX") in read-only mode.
func (s *imapSession) selectMailbox(folder string) (*imap.SelectData, error) {
	if folder == "" {
		folder = "INBOX"
	}
	data, err := s.client.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("IMAP SELECT %q failed: %v", folder, err)}
	}
	return data, nil
}

// close logs out and closes the IMAP connection.
func (s *imapSession) close() {
	if s.client != nil {
		s.client.Logout().Wait()
		s.client.Close()
	}
}

// emailSummary is the JSON representation of an email summary.
type emailSummary struct {
	SeqNum  uint32   `json:"seq_num"`
	Subject string   `json:"subject"`
	From    []string `json:"from"`
	To      []string `json:"to"`
	Date    string   `json:"date"`
	Flags   []string `json:"flags"`
}

// envelopeToSummary converts an IMAP envelope to our JSON summary format.
func envelopeToSummary(seqNum uint32, env *imap.Envelope, flags []imap.Flag) emailSummary {
	summary := emailSummary{
		SeqNum:  seqNum,
		Subject: env.Subject,
		Date:    env.Date.Format(time.RFC3339),
	}
	for _, addr := range env.From {
		a := addr.Addr()
		if a != "" {
			if addr.Name != "" {
				summary.From = append(summary.From, fmt.Sprintf("%s <%s>", addr.Name, a))
			} else {
				summary.From = append(summary.From, a)
			}
		}
	}
	for _, addr := range env.To {
		a := addr.Addr()
		if a != "" {
			summary.To = append(summary.To, a)
		}
	}
	for _, f := range flags {
		summary.Flags = append(summary.Flags, string(f))
	}
	return summary
}

// mapIMAPError converts an IMAP error to the appropriate connector error type.
func mapIMAPError(err error) error {
	if err == nil {
		return nil
	}
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("IMAP operation timed out: %v", err)}
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "auth") || strings.Contains(errMsg, "Auth") ||
		strings.Contains(errMsg, "LOGIN") || strings.Contains(errMsg, "credentials") {
		return &connectors.AuthError{Message: fmt.Sprintf("IMAP auth error: %v", err)}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("IMAP error: %v", err)}
}
