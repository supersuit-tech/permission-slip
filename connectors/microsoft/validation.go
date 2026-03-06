package microsoft

import (
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validateEmail performs basic email address validation for a recipient list entry.
// Used by send_email (to/cc) and create_calendar_event (attendees).
func validateEmail(field string, idx int, addr string) error {
	if addr == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("%s[%d] is empty", field, idx)}
	}
	if !strings.Contains(addr, "@") || strings.HasPrefix(addr, "@") || strings.HasSuffix(addr, "@") {
		return &connectors.ValidationError{Message: fmt.Sprintf("%s[%d] is not a valid email address: %q", field, idx, addr)}
	}
	return nil
}

// validateGraphID rejects Graph API resource identifiers (team IDs, channel IDs,
// message IDs, etc.) that contain path traversal sequences or path separators.
// These values are interpolated into URL paths, so we must ensure they cannot
// manipulate the request target. Microsoft Graph IDs are opaque strings (typically
// UUIDs or base64-encoded values) that never contain slashes or dot-dot sequences.
func validateGraphID(field, value string) error {
	if value == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", field)}
	}
	if strings.Contains(value, "..") || strings.Contains(value, "/") || strings.Contains(value, "\\") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must not contain path separators or traversal sequences", field)}
	}
	return nil
}

// detectContentType returns "HTML" if the body contains HTML tags, otherwise "Text".
func detectContentType(body string) string {
	if strings.Contains(body, "<") && strings.Contains(body, ">") {
		return "HTML"
	}
	return "Text"
}
