package protonmail

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"syscall"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// testIMAPConn performs IMAP login and a read-only INBOX SELECT without reading
// message content. Tests may replace it to avoid a running Bridge instance.
var testIMAPConn = func(creds connectors.Credentials, timeout time.Duration) error {
	session, err := connectIMAP(creds, timeout)
	if err != nil {
		return err
	}
	defer session.close()
	if _, err := session.selectMailbox("INBOX"); err != nil {
		return err
	}
	return nil
}

// testSMTPConn verifies SMTP reachability with EHLO + AUTH only — no mail is sent.
// Tests may replace it to avoid a running Bridge instance.
var testSMTPConn = func(creds connectors.Credentials, timeout time.Duration) error {
	host, port, username, password := smtpConfig(creds)
	addr := net.JoinHostPort(host, port)

	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return mapDialError("SMTP", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return &connectors.ExternalError{Message: fmt.Sprintf("SMTP handshake failed: %v", err)}
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(smtpTLSConfig(host)); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("SMTP STARTTLS failed: %v", err)}
		}
	}

	auth := smtp.PlainAuth("", username, password, host)
	if err := client.Auth(auth); err != nil {
		return &connectors.AuthError{Message: mapBridgeAuthMessage("SMTP", err)}
	}
	return client.Quit()
}

// TestBridgeConnection verifies IMAP and SMTP connectivity to the local Proton
// proxy. It performs login/handshake only — no mailbox reads or mail delivery.
func TestBridgeConnection(creds connectors.Credentials, timeout time.Duration) error {
	if err := validateCredentialShape(creds); err != nil {
		return err
	}
	if err := testIMAPConn(creds, timeout); err != nil {
		return mapBridgeError("IMAP", err)
	}
	if err := testSMTPConn(creds, timeout); err != nil {
		return mapBridgeError("SMTP", err)
	}
	return nil
}

// TestBridgeConnectionContext is like TestBridgeConnection but respects ctx deadlines.
func TestBridgeConnectionContext(ctx context.Context, creds connectors.Credentials, timeout time.Duration) error {
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}
	return TestBridgeConnection(creds, timeout)
}

func mapDialError(proto string, err error) error {
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{
			Message: fmt.Sprintf("%s host unreachable — check the %s host and that Bridge is running on this machine", proto, strings.ToLower(proto)),
		}
	}
	if isConnectionRefused(err) {
		return &connectors.ExternalError{
			Message: fmt.Sprintf("%s connection refused — Bridge not running or wrong port. Start Bridge (systemctl --user status protonmail-bridge) and confirm ports with protonmail-bridge info", proto),
		}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("%s connection failed: %v", proto, err)}
}

// mapBridgeAuthMessage reports which protocol the proxy rejected and the
// server's actual response, with the most common cause as a hint — auth can
// fail for reasons other than a wrong password (e.g. the Bridge instance
// serving the port has no account logged in), so don't assert one cause.
func mapBridgeAuthMessage(proto string, err error) string {
	return fmt.Sprintf("%s authentication rejected by Bridge: %v — check that the password is the bridge password from protonmail-bridge info (not your Proton account password) and that this Bridge instance has your account logged in", proto, err)
}

func mapBridgeError(proto string, err error) error {
	if err == nil {
		return nil
	}
	if connectors.IsValidationError(err) || connectors.IsAuthError(err) ||
		connectors.IsExternalError(err) || connectors.IsTimeoutError(err) {
		// Re-wrap with clearer messages where the low-level text is opaque.
		switch {
		case connectors.IsAuthError(err):
			var ae *connectors.AuthError
			if errors.As(err, &ae) {
				return &connectors.AuthError{Message: mapBridgeAuthMessage(proto, errors.New(ae.Message))}
			}
		case connectors.IsTimeoutError(err):
			return &connectors.TimeoutError{
				Message: fmt.Sprintf("%s host unreachable — check the %s host and that Bridge is running on this machine", proto, strings.ToLower(proto)),
			}
		case connectors.IsExternalError(err):
			var ee *connectors.ExternalError
			if errors.As(err, &ee) && isConnectionRefused(errors.New(ee.Message)) {
				return &connectors.ExternalError{
					Message: fmt.Sprintf("%s connection refused — Bridge not running or wrong port. Start Bridge and confirm ports with protonmail-bridge info", proto),
				}
			}
		}
		return err
	}
	return mapDialError(proto, err)
}

func isConnectionRefused(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && errors.Is(opErr.Err, syscall.ECONNREFUSED) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") || strings.Contains(msg, "connect: connection refused")
}
