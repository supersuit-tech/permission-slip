package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchDriveAction implements connectors.Action for google.search_drive.
// It searches Google Drive files by name, MIME type, or full-text content
// via the Google Drive API GET /drive/v3/files.
type searchDriveAction struct {
	conn *GoogleConnector
}

// searchDriveParams is the user-facing parameter schema.
type searchDriveParams struct {
	Query      string `json:"query"`
	FileType   string `json:"file_type"`
	FolderID   string `json:"folder_id"`
	MaxResults int    `json:"max_results"`
}

// driveFileTypeMap maps friendly type names to Drive MIME types.
var driveFileTypeMap = map[string]string{
	"audio":        "audio/",
	"document":     "application/vnd.google-apps.document",
	"folder":       "application/vnd.google-apps.folder",
	"image":        "image/",
	"pdf":          "application/pdf",
	"presentation": "application/vnd.google-apps.presentation",
	"spreadsheet":  "application/vnd.google-apps.spreadsheet",
	"video":        "video/",
}

// driveFileTypeNames is a sorted list of valid file_type values, used for
// deterministic error messages regardless of map iteration order.
var driveFileTypeNames = func() []string {
	names := make([]string, 0, len(driveFileTypeMap))
	for k := range driveFileTypeMap {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}()

func (p *searchDriveParams) validate() error {
	if p.Query == "" && p.FileType == "" && p.FolderID == "" {
		return &connectors.ValidationError{Message: "at least one of query, file_type, or folder_id must be provided"}
	}
	if p.FileType != "" {
		if _, ok := driveFileTypeMap[p.FileType]; !ok {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid file_type; valid options: %s", strings.Join(driveFileTypeNames, ", "))}
		}
	}
	if p.FolderID != "" && !isValidDriveID(p.FolderID) {
		return &connectors.ValidationError{Message: "folder_id contains invalid characters; expected alphanumeric ID"}
	}
	return nil
}

func (p *searchDriveParams) normalize() {
	if p.MaxResults <= 0 {
		p.MaxResults = 10
	}
	if p.MaxResults > 100 {
		p.MaxResults = 100
	}
}

// Execute searches Google Drive and returns matching file metadata.
func (a *searchDriveAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchDriveParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	var queryParts []string

	// Full-text / name search: use the fullText contains operator for broad search,
	// or combine name contains if the query looks like a simple name search.
	if params.Query != "" {
		escaped := strings.ReplaceAll(params.Query, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `'`, `\'`)
		queryParts = append(queryParts, fmt.Sprintf("(name contains '%s' or fullText contains '%s')", escaped, escaped))
	}

	if params.FileType != "" {
		mimeType := driveFileTypeMap[params.FileType]
		// For prefix-based types (image/, video/, audio/), use contains; otherwise exact match.
		if strings.HasSuffix(mimeType, "/") {
			escaped := strings.ReplaceAll(mimeType, `'`, `\'`)
			queryParts = append(queryParts, fmt.Sprintf("mimeType contains '%s'", escaped))
		} else {
			escaped := strings.ReplaceAll(mimeType, `'`, `\'`)
			queryParts = append(queryParts, fmt.Sprintf("mimeType = '%s'", escaped))
		}
	}

	if params.FolderID != "" {
		queryParts = append(queryParts, fmt.Sprintf("'%s' in parents", params.FolderID))
	}

	// Exclude trashed files.
	queryParts = append(queryParts, "trashed = false")

	q := url.Values{}
	q.Set("q", strings.Join(queryParts, " and "))
	q.Set("pageSize", strconv.Itoa(params.MaxResults))
	q.Set("fields", "files(id,name,mimeType,modifiedTime,size,webViewLink,parents)")
	q.Set("orderBy", "modifiedTime desc")

	var listResp driveListResponse
	searchURL := a.conn.driveBaseURL + "/drive/v3/files?" + q.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, searchURL, nil, &listResp); err != nil {
		return nil, err
	}

	summaries := make([]driveFileSummary, len(listResp.Files))
	for i, f := range listResp.Files {
		summaries[i] = driveFileSummary{
			ID:           f.ID,
			Name:         f.Name,
			MimeType:     f.MimeType,
			ModifiedTime: f.ModifiedTime,
			Size:         f.Size,
			WebViewLink:  f.WebViewLink,
		}
	}

	return connectors.JSONResult(map[string]any{
		"files": summaries,
		"count": len(summaries),
	})
}
