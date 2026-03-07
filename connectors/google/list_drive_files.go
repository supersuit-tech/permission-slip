package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listDriveFilesAction implements connectors.Action for google.list_drive_files.
// It lists files via the Google Drive API GET /drive/v3/files.
type listDriveFilesAction struct {
	conn *GoogleConnector
}

// listDriveFilesParams is the user-facing parameter schema.
type listDriveFilesParams struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
	FolderID   string `json:"folder_id"`
	OrderBy    string `json:"order_by"`
}

// validOrderByValues lists the allowed order_by values for Drive files.list.
var validOrderByValues = map[string]bool{
	"":                     true,
	"modifiedTime":         true,
	"modifiedTime desc":    true,
	"name":                 true,
	"name desc":            true,
	"createdTime":          true,
	"createdTime desc":     true,
	"folder":               true,
	"recency":              true,
	"viewedByMeTime":       true,
	"viewedByMeTime desc":  true,
	"sharedWithMeTime":     true,
	"sharedWithMeTime desc": true,
}

func (p *listDriveFilesParams) validate() error {
	if p.FolderID != "" && !isValidDriveID(p.FolderID) {
		return &connectors.ValidationError{Message: "folder_id contains invalid characters; expected alphanumeric ID"}
	}
	if !validOrderByValues[p.OrderBy] {
		return &connectors.ValidationError{Message: "invalid order_by value; valid options: modifiedTime, modifiedTime desc, name, name desc, createdTime, createdTime desc, folder, recency, viewedByMeTime, viewedByMeTime desc, sharedWithMeTime, sharedWithMeTime desc"}
	}
	return nil
}

func (p *listDriveFilesParams) normalize() {
	if p.MaxResults <= 0 {
		p.MaxResults = 10
	}
	if p.MaxResults > 100 {
		p.MaxResults = 100
	}
}

// driveIDPattern matches valid Google Drive file/folder IDs.
// Drive IDs are alphanumeric strings with hyphens and underscores, typically
// 20-60 characters. This rejects injection attempts (quotes, slashes, spaces).
var driveIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// isValidDriveID returns true if s looks like a valid Google Drive file or folder ID.
func isValidDriveID(s string) bool {
	return driveIDPattern.MatchString(s)
}

// driveListResponse is the Google Drive API response from files.list.
type driveListResponse struct {
	Files []driveFileEntry `json:"files"`
}

// driveFileEntry is a single file entry from the Google Drive API (camelCase fields).
type driveFileEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MimeType     string `json:"mimeType"`
	ModifiedTime string `json:"modifiedTime"`
	Size         string `json:"size"`
	WebViewLink  string `json:"webViewLink"`
}

// driveFileSummary is the shape returned to the agent for file listings (snake_case).
type driveFileSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MimeType     string `json:"mime_type"`
	ModifiedTime string `json:"modified_time,omitempty"`
	Size         string `json:"size,omitempty"`
	WebViewLink  string `json:"web_view_link,omitempty"`
}

// Execute lists files from Google Drive and returns their metadata.
func (a *listDriveFilesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listDriveFilesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	q := url.Values{}
	q.Set("pageSize", strconv.Itoa(params.MaxResults))
	q.Set("fields", "files(id,name,mimeType,modifiedTime,size,webViewLink)")

	if params.OrderBy != "" {
		q.Set("orderBy", params.OrderBy)
	}

	// Build the search query.
	var queryParts []string
	if params.Query != "" {
		queryParts = append(queryParts, params.Query)
	}
	if params.FolderID != "" {
		queryParts = append(queryParts, fmt.Sprintf("'%s' in parents", params.FolderID))
	}
	// Exclude trashed files by default.
	queryParts = append(queryParts, "trashed = false")
	q.Set("q", strings.Join(queryParts, " and "))

	var listResp driveListResponse
	listURL := a.conn.driveBaseURL + "/drive/v3/files?" + q.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, listURL, nil, &listResp); err != nil {
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
