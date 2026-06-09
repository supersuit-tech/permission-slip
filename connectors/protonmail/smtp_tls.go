package protonmail

import "crypto/tls"

// smtpTLSConfig returns the TLS configuration used for SMTP STARTTLS upgrades.
//
// For loopback hosts (Proton Mail Bridge or hydroxide), certificate
// verification is skipped: Bridge serves a self-signed certificate by design,
// so strict verification can never succeed on a headless host. The session is
// still TLS-encrypted and the traffic never leaves the machine — this mirrors
// imapDial, which dials loopback without TLS for the same reason. For remote
// hosts, full verification is enforced to protect credentials in transit.
func smtpTLSConfig(host string) *tls.Config {
	return &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: isLocalhost(host), //nolint:gosec // loopback only; Bridge's cert is self-signed by design
	}
}
