package connectors

import (
	"encoding/json"
	"unicode/utf8"
)

// MaxEmailThreadBodyRunes is the per-message body cap (html and text separately)
// for approval UI payloads. When exceeded, bodies are truncated and Truncated is set.
const MaxEmailThreadBodyRunes = 20 * 1024

// EmailThreadAttachment is attachment metadata without content (filename + size only).
type EmailThreadAttachment struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
}

// EmailThreadMessage is one message in a normalized email thread for approval UI.
type EmailThreadMessage struct {
	From        string                  `json:"from"`
	To          []string                `json:"to"`
	Cc          []string                `json:"cc"`
	Date        string                  `json:"date"`
	BodyHTML    string                  `json:"body_html"`
	BodyText    string                  `json:"body_text"`
	Snippet     string                  `json:"snippet"`
	MessageID   string                  `json:"message_id"`
	Truncated   bool                    `json:"truncated"`
	Attachments []EmailThreadAttachment `json:"attachments,omitempty"`
}

// EmailThreadPayload is the normalized thread shape stored at context.details.email_thread.
type EmailThreadPayload struct {
	Subject  string               `json:"subject"`
	Messages []EmailThreadMessage `json:"messages"`
}

// TruncateEmailThreadBodies applies MaxEmailThreadBodyRunes to body_html and body_text,
// setting truncated when either field is cut. Empty strings are left unchanged.
func TruncateEmailThreadBodies(m *EmailThreadMessage) {
	if m == nil {
		return
	}
	if s, cut := truncateAtMaxRunes(m.BodyHTML); cut {
		m.BodyHTML = s
		m.Truncated = true
	}
	if s, cut := truncateAtMaxRunes(m.BodyText); cut {
		m.BodyText = s
		m.Truncated = true
	}
}

func truncateAtMaxRunes(s string) (string, bool) {
	if s == "" {
		return "", false
	}
	if utf8.RuneCountInString(s) <= MaxEmailThreadBodyRunes {
		return s, false
	}
	runes := []rune(s)
	if len(runes) <= MaxEmailThreadBodyRunes {
		return s, false
	}
	return string(runes[:MaxEmailThreadBodyRunes]), true
}

// EmailThreadDetailsMap returns resource_details map entries for an email thread.
// Keys match context.details.email_thread in the OpenAPI contract.
func EmailThreadDetailsMap(thread *EmailThreadPayload) map[string]any {
	if thread == nil {
		return nil
	}
	b, err := json.Marshal(thread)
	if err != nil {
		return nil
	}
	var asMap map[string]any
	if err := json.Unmarshal(b, &asMap); err != nil {
		return nil
	}
	return map[string]any{"email_thread": asMap}
}
