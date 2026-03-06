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

// graphDriveItemsResponse is the Microsoft Graph API response for listing drive items.
type graphDriveItemsResponse struct {
	Value []graphDriveItem `json:"value"`
}

type graphDriveItem struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Size             int64           `json:"size"`
	WebURL           string          `json:"webUrl,omitempty"`
	CreatedDateTime  string          `json:"createdDateTime,omitempty"`
	ModifiedDateTime string          `json:"lastModifiedDateTime,omitempty"`
	Folder           *graphFolder    `json:"folder,omitempty"`
	File             *graphFileFacet `json:"file,omitempty"`
}

type graphFolder struct {
	ChildCount int `json:"childCount"`
}

type graphFileFacet struct {
	MimeType string `json:"mimeType"`
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

	if err := validateFolderPath(params.FolderPath); err != nil {
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

	return connectors.JSONResult(summaries)
}

// validateFolderPath rejects paths with traversal sequences, absolute paths, or backslashes.
func validateFolderPath(p string) error {
	if p == "" {
		return nil
	}
	if strings.Contains(p, "..") {
		return &connectors.ValidationError{Message: "invalid folder_path: must not contain path traversal sequences"}
	}
	if strings.Contains(p, "\\") {
		return &connectors.ValidationError{Message: "invalid folder_path: must not contain backslashes"}
	}
	if strings.HasPrefix(p, "/") {
		return &connectors.ValidationError{Message: "invalid folder_path: must be a relative path"}
	}
	return nil
}
