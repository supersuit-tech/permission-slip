package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getDriveFileAction implements connectors.Action for google.get_drive_file.
// It retrieves file metadata and optionally exports content for Google Workspace files.
type getDriveFileAction struct {
	conn *GoogleConnector
}

// getDriveFileParams is the user-facing parameter schema.
type getDriveFileParams struct {
	FileID         string `json:"file_id"`
	IncludeContent bool   `json:"include_content"`
}

func (p *getDriveFileParams) validate() error {
	if p.FileID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: file_id"}
	}
	if !isValidDriveID(p.FileID) {
		return &connectors.ValidationError{Message: "file_id contains invalid characters; expected alphanumeric ID"}
	}
	return nil
}

// driveFileResult is the shape returned to the agent.
type driveFileResult struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	MimeType             string `json:"mime_type"`
	ModifiedTime         string `json:"modified_time,omitempty"`
	Size                 string `json:"size,omitempty"`
	WebViewLink          string `json:"web_view_link,omitempty"`
	Content              string `json:"content,omitempty"`
	ContentSkippedReason string `json:"content_skipped_reason,omitempty"`
}

// googleWorkspaceMimeTypes maps Google Workspace MIME types to their
// text export MIME types. Only these types support content export.
var googleWorkspaceMimeTypes = map[string]string{
	"application/vnd.google-apps.document":     "text/plain",
	"application/vnd.google-apps.spreadsheet":  "text/csv",
	"application/vnd.google-apps.presentation": "text/plain",
}

// Execute retrieves file metadata and optionally exports content.
func (a *getDriveFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getDriveFileParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Step 1: fetch file metadata.
	fields := "id,name,mimeType,modifiedTime,size,webViewLink"
	metaURL := a.conn.driveBaseURL + "/drive/v3/files/" + url.PathEscape(params.FileID) + "?fields=" + url.QueryEscape(fields)

	var meta driveFileEntry
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, metaURL, nil, &meta); err != nil {
		return nil, err
	}

	result := driveFileResult{
		ID:           meta.ID,
		Name:         meta.Name,
		MimeType:     meta.MimeType,
		ModifiedTime: meta.ModifiedTime,
		Size:         meta.Size,
		WebViewLink:  meta.WebViewLink,
	}

	// Step 2: optionally export content for Google Workspace files.
	if params.IncludeContent {
		exportMime, isWorkspace := googleWorkspaceMimeTypes[meta.MimeType]
		if isWorkspace {
			content, err := a.exportContent(ctx, req.Credentials, params.FileID, exportMime)
			if err != nil {
				return nil, err
			}
			result.Content = content
		} else if isTextMimeType(meta.MimeType) {
			// For regular text files, download content directly.
			content, err := a.downloadContent(ctx, req.Credentials, params.FileID)
			if err != nil {
				return nil, err
			}
			result.Content = content
		} else {
			// Binary files can't be exported as text.
			result.ContentSkippedReason = "binary file type — content export is only supported for text files and Google Workspace documents"
		}
	}

	return connectors.JSONResult(result)
}

// exportContent exports a Google Workspace file as text.
func (a *getDriveFileAction) exportContent(ctx context.Context, creds connectors.Credentials, fileID, exportMime string) (string, error) {
	exportURL := a.conn.driveBaseURL + "/drive/v3/files/" + url.PathEscape(fileID) + "/export?mimeType=" + url.QueryEscape(exportMime)
	return a.conn.doRawGet(ctx, creds, exportURL)
}

// downloadContent downloads a non-Google-Workspace file's content.
func (a *getDriveFileAction) downloadContent(ctx context.Context, creds connectors.Credentials, fileID string) (string, error) {
	downloadURL := a.conn.driveBaseURL + "/drive/v3/files/" + url.PathEscape(fileID) + "?alt=media"
	return a.conn.doRawGet(ctx, creds, downloadURL)
}

// isTextMimeType returns true if the MIME type represents a text-based file
// that can be safely downloaded and returned as a string.
func isTextMimeType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "text/") ||
		mimeType == "application/json" ||
		mimeType == "application/xml" ||
		mimeType == "application/javascript"
}

