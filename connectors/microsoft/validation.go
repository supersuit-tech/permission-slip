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

// validatePathSegment checks that a value intended for use in a URL path segment
// does not contain path traversal sequences or slash characters. This prevents
// attackers from escaping the intended API path (e.g., using "../../admin" as
// an item_id or table_name).
func validatePathSegment(field, value string) error {
	if strings.Contains(value, "..") || strings.Contains(value, "/") || strings.Contains(value, "\\") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid %s: must not contain path separators or traversal sequences", field),
		}
	}
	return nil
}

// validateValuesGrid checks that a 2D values array has consistent column counts
// across all rows. Inconsistent dimensions cause cryptic Graph API errors, so
// catching them early provides a better developer experience.
func validateValuesGrid(values [][]any) error {
	if len(values) == 0 {
		return nil
	}
	cols := len(values[0])
	for i, row := range values[1:] {
		if len(row) != cols {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("values row %d has %d columns, but row 0 has %d — all rows must have the same number of columns", i+1, len(row), cols),
			}
		}
	}
	return nil
}
