package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createDriveFolderAction implements connectors.Action for google.create_drive_folder.
// It creates a folder via the Google Drive API POST /drive/v3/files with
// mimeType application/vnd.google-apps.folder.
type createDriveFolderAction struct {
	conn *GoogleConnector
}

// createDriveFolderParams is the user-facing parameter schema.
type createDriveFolderParams struct {
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}

func (p *createDriveFolderParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.ParentID != "" && !isValidDriveID(p.ParentID) {
		return &connectors.ValidationError{Message: "parent_id contains invalid characters; expected alphanumeric ID"}
	}
	return nil
}

// driveFolderCreateRequest is the Google Drive API request body for folder creation.
type driveFolderCreateRequest struct {
	Name     string   `json:"name"`
	MimeType string   `json:"mimeType"`
	Parents  []string `json:"parents,omitempty"`
}

// driveFolderResponse is the Google Drive API response after creating a folder.
type driveFolderResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MimeType    string `json:"mimeType"`
	WebViewLink string `json:"webViewLink"`
}

// Execute creates a folder in Google Drive and returns its metadata.
func (a *createDriveFolderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createDriveFolderParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := driveFolderCreateRequest{
		Name:     params.Name,
		MimeType: "application/vnd.google-apps.folder",
	}
	if params.ParentID != "" {
		body.Parents = []string{params.ParentID}
	}

	var resp driveFolderResponse
	createURL := a.conn.driveBaseURL + "/drive/v3/files?fields=id,name,mimeType,webViewLink"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, createURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":            resp.ID,
		"name":          resp.Name,
		"web_view_link": resp.WebViewLink,
	})
}
