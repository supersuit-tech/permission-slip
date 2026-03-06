package microsoft

import (
	"fmt"
	"net/url"
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
// item IDs, etc.) that contain path traversal sequences, path separators, or null
// bytes. These values are interpolated into URL paths, so we must ensure they
// cannot manipulate the request target. Microsoft Graph IDs are opaque strings
// (typically UUIDs or base64-encoded values) that never contain slashes or
// dot-dot sequences.
func validateGraphID(field, value string) error {
	if value == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", field)}
	}
	if strings.ContainsAny(value, "/\\") || strings.Contains(value, "..") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must not contain path separators or traversal sequences", field)}
	}
	if strings.ContainsRune(value, 0) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must not contain null bytes", field)}
	}
	return nil
}

// validateItemID checks that an item_id is a valid Graph API identifier.
func validateItemID(id string) error {
	return validateGraphID("item_id", id)
}

// detectContentType returns "HTML" if the body contains HTML tags, otherwise "Text".
func detectContentType(body string) string {
	if strings.Contains(body, "<") && strings.Contains(body, ">") {
		return "HTML"
	}
	return "Text"
}

// validateFolderPath checks that a folder path does not contain path traversal,
// backslashes, or null bytes. Empty paths are valid (means root).
func validateFolderPath(path string) error {
	if path == "" {
		return nil
	}
	if strings.Contains(path, "..") || strings.Contains(path, "\\") {
		return &connectors.ValidationError{Message: "invalid folder_path: must not contain path traversal sequences or backslashes"}
	}
	if strings.ContainsRune(path, 0) {
		return &connectors.ValidationError{Message: "invalid folder_path: must not contain null bytes"}
	}
	return nil
}

// escapeFolderPath URL-encodes each segment of a folder path, preserving
// forward slashes as path separators. Trims leading/trailing slashes.
func escapeFolderPath(path string) string {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	segments := strings.Split(path, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}
