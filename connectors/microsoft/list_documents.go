package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listDocumentsAction implements connectors.Action for microsoft.list_documents.
// It lists Word documents from a OneDrive folder via GET /me/drive/root:/{path}:/children.
type listDocumentsAction struct {
	conn *MicrosoftConnector
}

// listDocumentsParams is the user-facing parameter schema.
type listDocumentsParams struct {
	FolderPath string `json:"folder_path,omitempty"`
	Top        int    `json:"top"`
}

func (p *listDocumentsParams) defaults() {
	if p.Top <= 0 {
		p.Top = 10
	}
	if p.Top > 50 {
		p.Top = 50
	}
}

func (p *listDocumentsParams) validate() error {
	return validateFolderPath(p.FolderPath)
}

// graphDriveChildren is the Graph API response for listing children of a folder.
type graphDriveChildren struct {
	Value []graphDriveItem `json:"value"`
}

// documentListItem is the simplified response for each document in the list.
type documentListItem struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	WebURL               string `json:"web_url"`
	Size                 int64  `json:"size"`
	LastModifiedDateTime string `json:"last_modified_date_time"`
}

// Execute lists Word documents from a OneDrive folder.
func (a *listDocumentsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listDocumentsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.defaults()

	// Build the list path. Filter for .docx files using $filter on the name.
	var basePath string
	if params.FolderPath != "" {
		basePath = fmt.Sprintf("/me/drive/root:/%s:/children", escapeFolderPath(params.FolderPath))
	} else {
		basePath = "/me/drive/root/children"
	}

	path := fmt.Sprintf("%s?$top=%d&$select=id,name,webUrl,size,lastModifiedDateTime&$filter=endswith(name,'.docx')", basePath, params.Top)

	var resp graphDriveChildren
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	documents := make([]documentListItem, len(resp.Value))
	for i, item := range resp.Value {
		documents[i] = documentListItem{
			ID:                   item.ID,
			Name:                 item.Name,
			WebURL:               item.WebURL,
			Size:                 item.Size,
			LastModifiedDateTime: item.LastModifiedDateTime,
		}
	}

	return connectors.JSONResult(map[string]any{
		"documents": documents,
	})
}
