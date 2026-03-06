package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	// docxContentType is the MIME type for Word documents (.docx).
	docxContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

	// maxSimpleUploadSize is the OneDrive simple upload limit (4 MB).
	maxSimpleUploadSize = 4 * 1024 * 1024
)

// createDocumentAction implements connectors.Action for microsoft.create_document.
// It creates a new Word document in OneDrive via PUT /me/drive/root:/{path}:/content.
type createDocumentAction struct {
	conn *MicrosoftConnector
}

// createDocumentParams is the user-facing parameter schema.
type createDocumentParams struct {
	Filename   string `json:"filename"`
	FolderPath string `json:"folder_path,omitempty"`
	Content    string `json:"content,omitempty"`
}

func (p *createDocumentParams) validate() error {
	p.Filename = strings.TrimSpace(p.Filename)
	if p.Filename == "" {
		return &connectors.ValidationError{Message: "missing required parameter: filename"}
	}
	if strings.Contains(p.Filename, "/") || strings.Contains(p.Filename, "\\") || strings.Contains(p.Filename, "..") {
		return &connectors.ValidationError{Message: "invalid filename: must not contain path separators or traversal sequences"}
	}
	if p.FolderPath != "" {
		if strings.Contains(p.FolderPath, "..") {
			return &connectors.ValidationError{Message: "invalid folder_path: must not contain path traversal sequences"}
		}
	}
	if len(p.Content) > maxSimpleUploadSize {
		return &connectors.ValidationError{Message: fmt.Sprintf("content exceeds maximum size of %d MB for simple upload", maxSimpleUploadSize/(1024*1024))}
	}
	return nil
}

func (p *createDocumentParams) defaults() {
	if !strings.HasSuffix(strings.ToLower(p.Filename), ".docx") {
		p.Filename += ".docx"
	}
}

// graphDriveItem represents a OneDrive item returned by the Graph API.
type graphDriveItem struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	WebURL               string `json:"webUrl"`
	Size                 int64  `json:"size,omitempty"`
	CreatedDateTime      string `json:"createdDateTime,omitempty"`
	LastModifiedDateTime string `json:"lastModifiedDateTime,omitempty"`
	DownloadURL          string `json:"@microsoft.graph.downloadUrl,omitempty"`
}

// documentResult is the simplified response returned to the caller for create/update.
type documentResult struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	WebURL          string `json:"web_url"`
	CreatedDateTime string `json:"created_date_time,omitempty"`
	LastModifiedDateTime string `json:"last_modified_date_time,omitempty"`
}

// Execute creates a new Word document in OneDrive.
func (a *createDocumentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createDocumentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.defaults()

	// Build the upload path.
	var path string
	if params.FolderPath != "" {
		folderPath := strings.TrimPrefix(params.FolderPath, "/")
		folderPath = strings.TrimSuffix(folderPath, "/")
		path = fmt.Sprintf("/me/drive/root:/%s/%s:/content", folderPath, params.Filename)
	} else {
		path = fmt.Sprintf("/me/drive/root:/%s:/content", params.Filename)
	}

	var content []byte
	if params.Content != "" {
		content = []byte(params.Content)
	}

	var item graphDriveItem
	if err := a.conn.doUpload(ctx, http.MethodPut, path, req.Credentials, content, docxContentType, &item); err != nil {
		return nil, err
	}

	return connectors.JSONResult(documentResult{
		ID:              item.ID,
		Name:            item.Name,
		WebURL:          item.WebURL,
		CreatedDateTime: item.CreatedDateTime,
	})
}
