package microsoft

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
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
// message IDs, item IDs, etc.) that contain characters which could manipulate
// the request URL when interpolated into API paths. These IDs are opaque strings
// (typically UUIDs or base64-encoded values) from Microsoft Graph that never
// contain slashes, query parameters, or traversal sequences.
func validateGraphID(field, value string) error {
	if value == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", field)}
	}
	if strings.ContainsAny(value, "/\\?#%") || strings.Contains(value, "..") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid %s: must not contain path separators or URL-special characters (/, \\, ?, #, %%)", field),
		}
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

// sendMailGraphContentType maps the explicit html flag to Microsoft Graph
// sendMail body contentType. Nil defaults to HTML (issue #971).
func sendMailGraphContentType(html *bool) string {
	if html != nil && !*html {
		return "Text"
	}
	return "HTML"
}

// validateFolderPath checks a OneDrive folder path for traversal sequences and
// URL-significant characters that could manipulate the Graph API request.
// Slashes are allowed since folder paths are naturally slash-separated.
// Empty paths are valid (means root).
func validateFolderPath(folderPath string) error {
	if folderPath == "" {
		return nil
	}
	if strings.Contains(folderPath, "..") {
		return &connectors.ValidationError{
			Message: "invalid folder_path: must not contain '..' traversal sequences (e.g. use 'Documents/Presentations' instead)",
		}
	}
	if strings.ContainsAny(folderPath, "?#%\\") {
		return &connectors.ValidationError{
			Message: "invalid folder_path: must not contain special characters (?, #, %, \\)",
		}
	}
	if strings.ContainsRune(folderPath, 0) {
		return &connectors.ValidationError{Message: "invalid folder_path: must not contain null bytes"}
	}
	return nil
}

// normalizeFolderPath strips leading/trailing slashes from a folder path
// for consistent use in OneDrive API paths.
func normalizeFolderPath(folderPath string) string {
	folderPath = strings.TrimPrefix(folderPath, "/")
	folderPath = strings.TrimSuffix(folderPath, "/")
	return folderPath
}

// escapePathSegments URL-encodes each segment of a slash-separated path
// individually. This preserves the directory structure (slashes are kept)
// while encoding special characters within each segment name that could
// otherwise manipulate the Graph API URL.
func escapePathSegments(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
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
