package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateDocumentAction implements connectors.Action for microsoft.update_document.
// It updates the content of a OneDrive file via PUT /me/drive/items/{itemId}/content.
type updateDocumentAction struct {
	conn *MicrosoftConnector
}

// updateDocumentParams is the user-facing parameter schema.
type updateDocumentParams struct {
	ItemID  string `json:"item_id"`
	Content string `json:"content"`
}

func (p *updateDocumentParams) validate() error {
	if err := validateItemID(p.ItemID); err != nil {
		return err
	}
	if p.Content == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content"}
	}
	if len(p.Content) > maxSimpleUploadSize {
		return &connectors.ValidationError{Message: fmt.Sprintf("content exceeds maximum size of %d MB for simple upload", maxSimpleUploadSize/(1024*1024))}
	}
	return nil
}

// Execute updates the content of a document in OneDrive.
func (a *updateDocumentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateDocumentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/me/drive/items/%s/content", url.PathEscape(params.ItemID))

	var item graphDriveItem
	if err := a.conn.doUpload(ctx, http.MethodPut, path, req.Credentials, []byte(params.Content), docxContentType, &item); err != nil {
		return nil, err
	}

	return connectors.JSONResult(documentResult{
		ID:                   item.ID,
		Name:                 item.Name,
		WebURL:               item.WebURL,
		LastModifiedDateTime: item.LastModifiedDateTime,
	})
}
