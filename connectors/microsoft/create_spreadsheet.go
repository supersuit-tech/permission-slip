package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createSpreadsheetAction implements connectors.Action for microsoft.create_spreadsheet.
// It creates a new empty .xlsx file in the user's OneDrive via the Microsoft Graph API.
type createSpreadsheetAction struct {
	conn *MicrosoftConnector
}

// createSpreadsheetParams is the user-facing parameter schema.
type createSpreadsheetParams struct {
	Filename   string `json:"filename"`
	FolderPath string `json:"folder_path,omitempty"`
}

func (p *createSpreadsheetParams) validate() error {
	p.Filename = strings.TrimSpace(p.Filename)
	if p.Filename == "" {
		return &connectors.ValidationError{Message: "missing required parameter: filename (e.g. 'Budget 2026')"}
	}
	if strings.ContainsAny(p.Filename, "/\\") || strings.Contains(p.Filename, "..") {
		return &connectors.ValidationError{Message: "invalid filename: must be a simple name without path separators (e.g. 'My Spreadsheet' not 'folder/My Spreadsheet')"}
	}
	if strings.ContainsRune(p.Filename, 0) {
		return &connectors.ValidationError{Message: "invalid filename: must not contain null bytes"}
	}
	return validateFolderPath(p.FolderPath)
}

func (p *createSpreadsheetParams) defaults() {
	if !strings.HasSuffix(strings.ToLower(p.Filename), ".xlsx") {
		p.Filename += ".xlsx"
	}
}

// spreadsheetResult is the simplified response returned to the caller.
type spreadsheetResult struct {
	ItemID     string `json:"item_id"`
	Name       string `json:"name"`
	WebURL     string `json:"web_url"`
	FolderPath string `json:"folder_path"`
}

// Execute creates a new .xlsx file in OneDrive.
// It uses PUT /me/drive/root:/{path}/{filename}.xlsx:/content with the minimal
// XLSX template bytes as the request body.
func (a *createSpreadsheetAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createSpreadsheetParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.defaults()

	// Build the upload path. URL-encode user-supplied segments to prevent
	// path manipulation via special characters (?, #, etc.).
	escapedFilename := url.PathEscape(params.Filename)
	var path string
	folderDisplay := "/"
	if params.FolderPath == "" || params.FolderPath == "/" {
		path = fmt.Sprintf("/me/drive/root:/%s:/content", escapedFilename)
	} else {
		folder := normalizeFolderPath(params.FolderPath)
		folderDisplay = "/" + folder
		path = fmt.Sprintf("/me/drive/root:/%s/%s:/content", escapePathSegments(folder), escapedFilename)
	}

	var resp graphDriveItemResponse
	const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if err := a.conn.doUpload(ctx, http.MethodPut, path, req.Credentials, minimalXLSX, xlsxContentType, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(spreadsheetResult{
		ItemID:     resp.ID,
		Name:       resp.Name,
		WebURL:     resp.WebURL,
		FolderPath: folderDisplay,
	})
}
