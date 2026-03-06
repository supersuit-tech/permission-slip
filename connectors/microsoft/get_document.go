package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getDocumentAction implements connectors.Action for microsoft.get_document.
// It retrieves metadata for a OneDrive item via GET /me/drive/items/{itemId}.
type getDocumentAction struct {
	conn *MicrosoftConnector
}

// getDocumentParams is the user-facing parameter schema.
type getDocumentParams struct {
	ItemID string `json:"item_id"`
}

func (p *getDocumentParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id"}
	}
	return nil
}

// documentMetadata is the simplified response returned to the caller.
type documentMetadata struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	WebURL               string `json:"web_url"`
	Size                 int64  `json:"size"`
	CreatedDateTime      string `json:"created_date_time"`
	LastModifiedDateTime string `json:"last_modified_date_time"`
}

// Execute gets metadata for a document in OneDrive.
func (a *getDocumentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getDocumentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/me/drive/items/%s?$select=id,name,webUrl,size,createdDateTime,lastModifiedDateTime", params.ItemID)

	var item graphDriveItem
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &item); err != nil {
		return nil, err
	}

	return connectors.JSONResult(documentMetadata{
		ID:                   item.ID,
		Name:                 item.Name,
		WebURL:               item.WebURL,
		Size:                 item.Size,
		CreatedDateTime:      item.CreatedDateTime,
		LastModifiedDateTime: item.LastModifiedDateTime,
	})
}
