// Package protonmail implements the built-in Proton Mail connector for Permission
// Slip. It speaks IMAP/SMTP to a locally running Proton proxy — either Proton
// Mail Bridge (official, x86_64) or hydroxide (open-source, ARM-friendly). Both
// expose the same loopback IMAP/SMTP surface that this connector targets.
//
// Actions: send_email, reply_email (SMTP); read/search/inbox, archive/move/delete, flags (IMAP).
package protonmail

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultTimeout = 30 * time.Second

	credKeyUsername = "username"
	credKeyPassword = "password"
	credKeySMTPHost = "smtp_host"
	credKeySMTPPort = "smtp_port"
	credKeyIMAPHost = "imap_host"
	credKeyIMAPPort = "imap_port"

	defaultSMTPHost = "127.0.0.1"
	defaultSMTPPort = "1025"
	defaultIMAPHost = "127.0.0.1"
	defaultIMAPPort = "1143"
)

// ProtonMailConnector owns the shared configuration for all Proton Mail actions.
type ProtonMailConnector struct {
	timeout time.Duration
}

// New creates a ProtonMailConnector with sensible defaults.
func New() *ProtonMailConnector {
	return &ProtonMailConnector{
		timeout: defaultTimeout,
	}
}

// ID returns "protonmail".
func (c *ProtonMailConnector) ID() string { return "protonmail" }

// Actions returns the registered action handlers keyed by action_type.
func (c *ProtonMailConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"protonmail.send_email":     &sendEmailAction{conn: c},
		"protonmail.reply_email":    &replyEmailAction{conn: c},
		"protonmail.read_inbox":     &readInboxAction{conn: c},
		"protonmail.search_emails":  &searchEmailsAction{conn: c},
		"protonmail.read_email":     &readEmailAction{conn: c},
		"protonmail.archive_email":  &archiveEmailAction{conn: c},
		"protonmail.list_folders":   &listFoldersAction{conn: c},
		"protonmail.mark_read":      newMarkReadAction(c),
		"protonmail.mark_unread":    newMarkUnreadAction(c),
		"protonmail.flag":           newFlagAction(c),
		"protonmail.unflag":         newUnflagAction(c),
		"protonmail.move_to_folder": &moveToFolderAction{conn: c},
		"protonmail.delete":         &deleteEmailAction{conn: c},
	}
}

// ValidateCredentials checks credential shape and verifies the local Proton
// proxy is reachable with a real IMAP LOGIN. The proxy (Bridge or hydroxide)
// must be running when credentials are saved.
func (c *ProtonMailConnector) ValidateCredentials(ctx context.Context, creds connectors.Credentials) error {
	if err := validateCredentialShape(creds); err != nil {
		return err
	}

	timeout := c.timeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	return TestBridgeConnection(creds, timeout)
}

func validateCredentialShape(creds connectors.Credentials) error {
	username, ok := creds.Get(credKeyUsername)
	if !ok || username == "" {
		return &connectors.ValidationError{Message: "missing required credential: username"}
	}
	password, ok := creds.Get(credKeyPassword)
	if !ok || password == "" {
		return &connectors.ValidationError{Message: "missing required credential: password"}
	}

	for _, key := range []string{credKeySMTPPort, credKeyIMAPPort} {
		if v, exists := creds.Get(key); exists && v != "" {
			if _, err := strconv.Atoi(v); err != nil {
				return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must be a numeric port value", key)}
			}
		}
	}
	return nil
}

// smtpConfig extracts SMTP connection settings from credentials, using the
// loopback defaults shared by Proton Mail Bridge and hydroxide.
func smtpConfig(creds connectors.Credentials) (host, port, username, password string) {
	host, _ = creds.Get(credKeySMTPHost)
	if host == "" {
		host = defaultSMTPHost
	}
	port, _ = creds.Get(credKeySMTPPort)
	if port == "" {
		port = defaultSMTPPort
	}
	username, _ = creds.Get(credKeyUsername)
	password, _ = creds.Get(credKeyPassword)
	return host, port, username, password
}

// imapConfig extracts IMAP connection settings from credentials, using the
// loopback defaults shared by Proton Mail Bridge and hydroxide.
func imapConfig(creds connectors.Credentials) (host, port, username, password string) {
	host, _ = creds.Get(credKeyIMAPHost)
	if host == "" {
		host = defaultIMAPHost
	}
	port, _ = creds.Get(credKeyIMAPPort)
	if port == "" {
		port = defaultIMAPPort
	}
	username, _ = creds.Get(credKeyUsername)
	password, _ = creds.Get(credKeyPassword)
	return host, port, username, password
}
