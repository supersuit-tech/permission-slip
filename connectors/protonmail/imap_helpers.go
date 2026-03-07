package protonmail

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// imapDial is a package-level variable so tests can replace it.
// It dials with a timeout and uses TLS for non-localhost hosts.
var imapDial = func(addr string, timeout time.Duration) (*imapclient.Client, error) {
	host, _, _ := net.SplitHostPort(addr)
	dialer := &net.Dialer{Timeout: timeout}

	if isLocalhost(host) {
		// Proton Mail Bridge on localhost uses plain IMAP (no TLS).
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		return imapclient.New(conn, nil), nil
	}

	// For remote hosts, use TLS to protect credentials in transit.
	tlsConn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName: host,
	})
	if err != nil {
		return nil, err
	}
	return imapclient.New(tlsConn, nil), nil
}

// isLocalhost returns true for loopback addresses.
func isLocalhost(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// imapSession holds an authenticated IMAP session with a selected mailbox.
type imapSession struct {
	client *imapclient.Client
}

// connectIMAP creates an authenticated IMAP connection with a timeout.
func connectIMAP(creds connectors.Credentials, timeout time.Duration) (*imapSession, error) {
	host, port, username, password := imapConfig(creds)
	addr := net.JoinHostPort(host, port)

	client, err := imapDial(addr, timeout)
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

// defaultLimit is the default number of emails to fetch.
const defaultLimit = 10

// maxLimit is the maximum number of emails that can be fetched.
const maxLimit = 50

// validateLimit applies defaults and validates the limit parameter.
func validateLimit(limit *int) error {
	if *limit <= 0 {
		*limit = defaultLimit
	}
	if *limit > maxLimit {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be at most %d", maxLimit)}
	}
	return nil
}

// emptyEmailResult returns a JSON result with an empty email list.
func emptyEmailResult() (*connectors.ActionResult, error) {
	return connectors.JSONResult(map[string]any{
		"emails": []emailSummary{},
		"total":  0,
	})
}

// emailListResult returns a JSON result wrapping the given email summaries.
func emailListResult(emails []emailSummary) (*connectors.ActionResult, error) {
	return connectors.JSONResult(map[string]any{
		"emails": emails,
		"total":  len(emails),
	})
}

// fetchEnvelopes fetches message envelopes for the given sequence set and
// returns them as email summaries.
func fetchEnvelopes(session *imapSession, seqSet imap.SeqSet) ([]emailSummary, error) {
	fetchCmd := session.client.Fetch(seqSet, &imap.FetchOptions{
		Envelope: true,
		Flags:    true,
	})
	defer fetchCmd.Close()

	var emails []emailSummary
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		buf, err := msg.Collect()
		if err != nil {
			return nil, mapIMAPError(err)
		}
		if buf.Envelope != nil {
			emails = append(emails, envelopeToSummary(buf.SeqNum, buf.Envelope, buf.Flags))
		}
	}
	return emails, nil
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

// formatAddresses formats IMAP addresses as human-readable strings.
// Addresses with a name are formatted as "Name <email>", others as bare email.
func formatAddresses(addrs []imap.Address) []string {
	var result []string
	for _, addr := range addrs {
		a := addr.Addr()
		if a == "" {
			continue
		}
		if addr.Name != "" {
			result = append(result, fmt.Sprintf("%s <%s>", addr.Name, a))
		} else {
			result = append(result, a)
		}
	}
	return result
}

// envelopeToSummary converts an IMAP envelope to our JSON summary format.
func envelopeToSummary(seqNum uint32, env *imap.Envelope, flags []imap.Flag) emailSummary {
	summary := emailSummary{
		SeqNum:  seqNum,
		Subject: env.Subject,
		Date:    env.Date.Format(time.RFC3339),
		From:    formatAddresses(env.From),
		To:      formatAddresses(env.To),
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
