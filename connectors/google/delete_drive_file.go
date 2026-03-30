package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteDriveFileAction implements connectors.Action for google.delete_drive_file.
// It moves a file to trash (soft delete) via PATCH /drive/v3/files/{id}.
type deleteDriveFileAction struct {
	conn *GoogleConnector
}

// deleteDriveFileParams is the user-facing parameter schema.
type deleteDriveFileParams struct {
	FileID string `json:"file_id"`
}

func (p *deleteDriveFileParams) validate() error {
	if p.FileID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: file_id"}
	}
	if !isValidDriveID(p.FileID) {
		return &connectors.ValidationError{Message: "file_id contains invalid characters; expected alphanumeric ID"}
	}
	return nil
}

// driveTrashRequest is the request body to move a file to trash.
type driveTrashRequest struct {
	Trashed bool `json:"trashed"`
}

// driveTrashResponse is the Google Drive API response after trashing.
type driveTrashResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Trashed bool   `json:"trashed"`
}

// Execute moves a file to trash in Google Drive.
func (a *deleteDriveFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteDriveFileParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := driveTrashRequest{Trashed: true}
	var resp driveTrashResponse

	patchURL := a.conn.driveBaseURL + "/drive/v3/files/" + url.PathEscape(params.FileID) + "?fields=id,name,trashed"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPatch, patchURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"id":      resp.ID,
		"name":    resp.Name,
		"trashed": resp.Trashed,
	})
}
