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

// detectContentType returns "HTML" if the body contains HTML tags, otherwise "Text".
func detectContentType(body string) string {
	if strings.Contains(body, "<") && strings.Contains(body, ">") {
		return "HTML"
	}
	return "Text"
}

// validateItemID checks that an item_id is non-empty and does not contain
// path traversal sequences, separators, or null bytes.
func validateItemID(id string) error {
	if id == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id"}
	}
	if strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
		return &connectors.ValidationError{Message: "invalid item_id: must not contain path separators or traversal sequences"}
	}
	if strings.ContainsRune(id, 0) {
		return &connectors.ValidationError{Message: "invalid item_id: must not contain null bytes"}
	}
	return nil
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
