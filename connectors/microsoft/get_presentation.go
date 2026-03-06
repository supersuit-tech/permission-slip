package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getPresentationAction implements connectors.Action for microsoft.get_presentation.
// It retrieves metadata for a specific PowerPoint file by item ID via the Microsoft Graph API.
type getPresentationAction struct {
	conn *MicrosoftConnector
}

// getPresentationParams is the user-facing parameter schema.
type getPresentationParams struct {
	ItemID string `json:"item_id"`
}

func (p *getPresentationParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id"}
	}
	return nil
}

// Execute retrieves metadata for a PowerPoint file from OneDrive.
func (a *getPresentationAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getPresentationParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/me/drive/items/%s?$select=id,name,webUrl,size,lastModifiedBy,lastModifiedDateTime", params.ItemID)

	var resp graphDriveItemResponse
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"item_id":          resp.ID,
		"name":             resp.Name,
		"web_url":          resp.WebURL,
		"size":             resp.Size,
		"last_modified_by": resp.LastModifiedBy.User.DisplayName,
		"last_modified":    resp.LastModified,
	})
}
