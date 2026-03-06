package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listDriveFilesAction implements connectors.Action for microsoft.list_drive_files.
// It lists files and folders in OneDrive via the Microsoft Graph API.
type listDriveFilesAction struct {
	conn *MicrosoftConnector
}

type listDriveFilesParams struct {
	FolderPath string `json:"folder_path"`
	Top        int    `json:"top"`
}

func (p *listDriveFilesParams) defaults() {
	if p.Top <= 0 {
		p.Top = 10
	}
	if p.Top > 50 {
		p.Top = 50
	}
}

// driveFileSummary is the simplified response returned to the caller.
type driveFileSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Size        int64  `json:"size"`
	MimeType    string `json:"mime_type,omitempty"`
	ChildCount  int    `json:"child_count,omitempty"`
	WebURL      string `json:"web_url,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	ModifiedAt  string `json:"modified_at,omitempty"`
}

func (a *listDriveFilesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listDriveFilesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	params.defaults()

	if err := validateFolderPathStrict(params.FolderPath); err != nil {
		return nil, err
	}

	var path string
	if params.FolderPath == "" {
		path = fmt.Sprintf("/me/drive/root/children?$top=%d&$select=id,name,size,webUrl,createdDateTime,lastModifiedDateTime,folder,file", params.Top)
	} else {
		path = fmt.Sprintf("/me/drive/root:/%s:/children?$top=%d&$select=id,name,size,webUrl,createdDateTime,lastModifiedDateTime,folder,file", params.FolderPath, params.Top)
	}

	var resp graphDriveItemsResponse
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	folderDisplay := params.FolderPath
	if folderDisplay == "" {
		folderDisplay = "/"
	}

	summaries := make([]driveFileSummary, len(resp.Value))
	for i, item := range resp.Value {
		summary := driveFileSummary{
			ID:         item.ID,
			Name:       item.Name,
			Size:       item.Size,
			WebURL:     item.WebURL,
			CreatedAt:  item.CreatedDateTime,
			ModifiedAt: item.ModifiedDateTime,
		}
		if item.Folder != nil {
			summary.Type = "folder"
			summary.ChildCount = item.Folder.ChildCount
		} else {
			summary.Type = "file"
			if item.File != nil {
				summary.MimeType = item.File.MimeType
			}
		}
		summaries[i] = summary
	}

	return connectors.JSONResult(struct {
		FolderPath string             `json:"folder_path"`
		Items      []driveFileSummary `json:"items"`
	}{
		FolderPath: folderDisplay,
		Items:      summaries,
	})
}

// validateFolderPathStrict rejects paths with traversal sequences, absolute paths, or backslashes.
// Used by list_drive_files where folder_path must be relative (no leading slash).
func validateFolderPathStrict(p string) error {
	if p == "" {
		return nil
	}
	return validateRelativePath("folder_path", p)
}

// validateRelativePath validates that a path is relative and safe from traversal
// and URL injection attacks. Used by both folder_path and file_path validation.
func validateRelativePath(field, p string) error {
	if strings.Contains(p, "..") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must not contain path traversal sequences", field)}
	}
	if strings.Contains(p, "\\") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must not contain backslashes", field)}
	}
	if strings.HasPrefix(p, "/") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must be a relative path", field)}
	}
	if strings.ContainsAny(p, "?#%") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must not contain URL-special characters", field)}
	}
	return nil
}
