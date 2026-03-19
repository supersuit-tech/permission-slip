package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// maxUploadContentSize limits the upload content to 4 MB.
const maxUploadContentSize = 4 * 1024 * 1024

// uploadDriveFileAction implements connectors.Action for microsoft.upload_drive_file.
// It uploads or creates a file in OneDrive via PUT /me/drive/root:/{path}:/content.
type uploadDriveFileAction struct {
	conn *MicrosoftConnector
}

type uploadDriveFileParams struct {
	FilePath         string `json:"file_path"`
	Content          string `json:"content"`
	ConflictBehavior string `json:"conflict_behavior"`
}

func (p *uploadDriveFileParams) defaults() {
	if p.ConflictBehavior == "" {
		p.ConflictBehavior = "rename"
	}
}

func (p *uploadDriveFileParams) validate() error {
	if p.FilePath == "" {
		return &connectors.ValidationError{Message: "missing required parameter: file_path"}
	}
	if p.Content != "" && len(p.Content) > maxUploadContentSize {
		return &connectors.ValidationError{
			Message: "content exceeds maximum upload size of 4 MB",
		}
	}
	if err := validateFilePath(p.FilePath); err != nil {
		return err
	}
	switch p.ConflictBehavior {
	case "rename", "replace", "fail":
		// valid
	default:
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid conflict_behavior %q: must be rename, replace, or fail", p.ConflictBehavior),
		}
	}
	return nil
}

// uploadDriveFileResult is the simplified response for uploaded files.
type uploadDriveFileResult struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	WebURL     string `json:"web_url,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	ModifiedAt string `json:"modified_at,omitempty"`
}

func (a *uploadDriveFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params uploadDriveFileParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	params.defaults()
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/me/drive/root:/%s:/content?@microsoft.graph.conflictBehavior=%s",
		params.FilePath, params.ConflictBehavior)

	// When creating an .xlsx file with no content, use the minimal XLSX template
	// so the file is a valid workbook. An empty file (0 bytes) with an .xlsx
	// extension is not valid Office Open XML, and the Graph API will return
	// 501/FileCorruptTryRepair when any Excel operation is attempted on it.
	uploadContent := []byte(params.Content)
	if params.Content == "" && strings.HasSuffix(strings.ToLower(params.FilePath), ".xlsx") {
		uploadContent = bytes.Clone(minimalXLSX)
	}

	content, err := a.conn.doPutRaw(ctx, path, req.Credentials, uploadContent)
	if err != nil {
		return nil, err
	}

	var item graphDriveItem
	if err := json.Unmarshal(content, &item); err != nil {
		return nil, &connectors.ExternalError{Message: "failed to decode upload response"}
	}

	return connectors.JSONResult(uploadDriveFileResult{
		ID:         item.ID,
		Name:       item.Name,
		Size:       item.Size,
		WebURL:     item.WebURL,
		CreatedAt:  item.CreatedDateTime,
		ModifiedAt: item.LastModifiedDateTime,
	})
}

// validateFilePath rejects paths with traversal sequences, backslashes, or absolute paths.
func validateFilePath(p string) error {
	return validateRelativePath("file_path", p)
}
