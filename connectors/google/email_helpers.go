package google

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"html"
	"regexp"
	"strings"
)

var tagPattern = regexp.MustCompile(`<[^>]*>`)

// emailHTMLDefault returns true when html is nil (default HTML) or *html is true.
func emailHTMLDefault(html *bool) bool {
	if html == nil {
		return true
	}
	return *html
}

// htmlToPlainText produces a minimal plain-text fallback from HTML for the
// text/plain MIME part (tags stripped, entities decoded).
func htmlToPlainText(s string) string {
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = strings.ReplaceAll(s, "</p>", "\n")
	s = strings.ReplaceAll(s, "</div>", "\n")
	s = tagPattern.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

func randomMimeBoundary() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Extremely unlikely; use a fixed fallback (still unique enough for one message).
		return "ps_mime_fallback_boundary"
	}
	return "ps_mime_" + hex.EncodeToString(b[:])
}

// buildGmailRaw constructs an RFC 2822 email message and returns it as a
// base64url-encoded string suitable for the Gmail API's raw field.
//
// When html is true, the message is multipart/alternative with text/plain and
// text/html parts. When html is false, a single text/plain part is used.
//
// Callers are responsible for sanitizing header values before passing them in.
// extraHeaders are written between Subject and Content-Type, in the order given;
// entries with an empty value are skipped.
func buildGmailRaw(to, subject, body string, html bool, extraHeaders [][2]string) string {
	var msg strings.Builder
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	for _, h := range extraHeaders {
		if h[1] != "" {
			msg.WriteString(h[0] + ": " + h[1] + "\r\n")
		}
	}
	if html {
		boundary := randomMimeBoundary()
		plain := htmlToPlainText(body)
		if plain == "" {
			plain = " "
		}
		msg.WriteString("MIME-Version: 1.0\r\n")
		msg.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n")
		msg.WriteString("\r\n")
		msg.WriteString("--" + boundary + "\r\n")
		msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(plain)
		msg.WriteString("\r\n")
		msg.WriteString("--" + boundary + "\r\n")
		msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(body)
		msg.WriteString("\r\n")
		msg.WriteString("--" + boundary + "--\r\n")
	} else {
		msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(body)
	}
	return base64.RawURLEncoding.EncodeToString([]byte(msg.String()))
}
