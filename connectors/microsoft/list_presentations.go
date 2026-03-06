package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listPresentationsAction implements connectors.Action for microsoft.list_presentations.
// It searches for .pptx files in the user's OneDrive via the Microsoft Graph API.
type listPresentationsAction struct {
	conn *MicrosoftConnector
}

// listPresentationsParams is the user-facing parameter schema.
type listPresentationsParams struct {
	FolderPath string `json:"folder_path,omitempty"`
	Top        int    `json:"top"`
}

func (p *listPresentationsParams) defaults() {
	if p.Top <= 0 {
		p.Top = 10
	}
	if p.Top > 50 {
		p.Top = 50
	}
}

func (p *listPresentationsParams) validate() error {
	return validateFolderPath(p.FolderPath)
}

// graphDriveSearchResponse is the Microsoft Graph API response for drive searches.
type graphDriveSearchResponse struct {
	Value []graphDriveItemResponse `json:"value"`
}

// presentationSummary is the simplified response returned to the caller.
type presentationSummary struct {
	ItemID       string `json:"item_id"`
	Name         string `json:"name"`
	WebURL       string `json:"web_url"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified"`
}

// Execute lists PowerPoint files from the user's OneDrive.
func (a *listPresentationsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listPresentationsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.defaults()

	var path string
	if params.FolderPath == "" || params.FolderPath == "/" {
		path = fmt.Sprintf("/me/drive/root/search(q='.pptx')?$top=%d&$select=id,name,webUrl,size,lastModifiedDateTime", params.Top)
	} else {
		folder := normalizeFolderPath(params.FolderPath)
		path = fmt.Sprintf("/me/drive/root:/%s:/search(q='.pptx')?$top=%d&$select=id,name,webUrl,size,lastModifiedDateTime", escapePathSegments(folder), params.Top)
	}

	var resp graphDriveSearchResponse
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	summaries := make([]presentationSummary, 0, len(resp.Value))
	for _, item := range resp.Value {
		// Only include actual .pptx files (search may return partial matches).
		if !strings.HasSuffix(strings.ToLower(item.Name), ".pptx") {
			continue
		}
		summaries = append(summaries, presentationSummary{
			ItemID:       item.ID,
			Name:         item.Name,
			WebURL:       item.WebURL,
			Size:         item.Size,
			LastModified: item.LastModified,
		})
	}

	return connectors.JSONResult(summaries)
}
