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

// validatePathSegment rejects values that contain URL-significant characters
// which could alter the request path or query when interpolated into a Graph
// API URL. This prevents a crafted item_id like "foo/../me/messages" or
// "foo?$select=body" from hitting unintended endpoints.
func validatePathSegment(field, value string) error {
	for _, ch := range value {
		switch ch {
		case '/', '\\', '?', '#', '%':
			return &connectors.ValidationError{
				Message: fmt.Sprintf("invalid %s: must not contain URL-special characters (/, \\, ?, #, %%)", field),
			}
		}
	}
	if strings.Contains(value, "..") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid %s: must not contain traversal sequences", field),
		}
	}
	return nil
}

// validateFolderPath checks a OneDrive folder path for traversal sequences and
// URL-significant characters that could manipulate the Graph API request.
// Slashes are allowed since folder paths are naturally slash-separated.
func validateFolderPath(folderPath string) error {
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
