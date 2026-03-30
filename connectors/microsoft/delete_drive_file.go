package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteDriveFileAction implements connectors.Action for microsoft.delete_drive_file.
// It moves a file to the OneDrive recycle bin (recoverable) via DELETE /me/drive/items/{id}.
type deleteDriveFileAction struct {
	conn *MicrosoftConnector
}

type deleteDriveFileParams struct {
	ItemID string `json:"item_id"`
}

func (p *deleteDriveFileParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id"}
	}
	if err := validateItemID(p.ItemID); err != nil {
		return err
	}
	return nil
}

func (a *deleteDriveFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteDriveFileParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/me/drive/items/%s", params.ItemID)

	// DELETE returns 204 No Content on success.
	if err := a.conn.doRequest(ctx, http.MethodDelete, path, req.Credentials, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":  "deleted",
		"item_id": params.ItemID,
		"message": "File moved to recycle bin and can be recovered",
	})
}
