package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createPresentationAction implements connectors.Action for microsoft.create_presentation.
// It creates a new empty .pptx file in the user's OneDrive via the Microsoft Graph API.
type createPresentationAction struct {
	conn *MicrosoftConnector
}

// createPresentationParams is the user-facing parameter schema.
type createPresentationParams struct {
	Filename   string `json:"filename"`
	FolderPath string `json:"folder_path,omitempty"`
}

func (p *createPresentationParams) validate() error {
	if p.Filename == "" {
		return &connectors.ValidationError{Message: "missing required parameter: filename (e.g. 'Quarterly Report')"}
	}
	// Reject path traversal or directory separators in filename.
	if strings.Contains(p.Filename, "/") || strings.Contains(p.Filename, "\\") || strings.Contains(p.Filename, "..") {
		return &connectors.ValidationError{Message: "invalid filename: must be a simple name without path separators (e.g. 'My Deck' not 'folder/My Deck')"}
	}
	return validateFolderPath(p.FolderPath)
}

// graphDriveItemResponse is the Microsoft Graph API response for a DriveItem.
type graphDriveItemResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	WebURL         string `json:"webUrl"`
	Size           int64  `json:"size"`
	LastModifiedBy struct {
		User struct {
			DisplayName string `json:"displayName"`
		} `json:"user"`
	} `json:"lastModifiedBy"`
	LastModified string `json:"lastModifiedDateTime"`
}

// Execute creates a new .pptx file in OneDrive.
// It uses PUT /me/drive/root:/{path}/{filename}.pptx:/content with the minimal
// PPTX template bytes as the request body.
func (a *createPresentationAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPresentationParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Ensure filename ends with .pptx.
	filename := params.Filename
	if !strings.HasSuffix(strings.ToLower(filename), ".pptx") {
		filename += ".pptx"
	}

	// Build the upload path.
	var path string
	folderDisplay := "/"
	if params.FolderPath == "" || params.FolderPath == "/" {
		path = fmt.Sprintf("/me/drive/root:/%s:/content", filename)
	} else {
		folder := normalizeFolderPath(params.FolderPath)
		folderDisplay = "/" + folder
		path = fmt.Sprintf("/me/drive/root:/%s/%s:/content", folder, filename)
	}

	var resp graphDriveItemResponse
	if err := a.conn.doPutFileRequest(ctx, path, req.Credentials, minimalPPTX, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"item_id":     resp.ID,
		"name":        resp.Name,
		"web_url":     resp.WebURL,
		"folder_path": folderDisplay,
	})
}
