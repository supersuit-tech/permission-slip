package google

import (
	"encoding/base64"
	"strings"
)

// buildGmailRaw constructs a plain-text RFC 2822 email message and returns
// it as a base64url-encoded string suitable for the Gmail API's raw field.
//
// Callers are responsible for sanitizing header values before passing them in.
// extraHeaders are written between Subject and Content-Type, in the order given;
// entries with an empty value are skipped.
func buildGmailRaw(to, subject, body string, extraHeaders [][2]string) string {
	var msg strings.Builder
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	for _, h := range extraHeaders {
		if h[1] != "" {
			msg.WriteString(h[0] + ": " + h[1] + "\r\n")
		}
	}
	msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)
	return base64.RawURLEncoding.EncodeToString([]byte(msg.String()))
}
