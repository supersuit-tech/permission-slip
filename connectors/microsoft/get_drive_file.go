package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getDriveFileAction implements connectors.Action for microsoft.get_drive_file.
// It retrieves file metadata and optionally downloads text content via the Graph API.
type getDriveFileAction struct {
	conn *MicrosoftConnector
}

type getDriveFileParams struct {
	ItemID         string `json:"item_id"`
	IncludeContent bool   `json:"include_content"`
}

func (p *getDriveFileParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id"}
	}
	if err := validateItemID(p.ItemID); err != nil {
		return err
	}
	return nil
}

// driveFileDetail is the detailed response for a single drive item.
type driveFileDetail struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Size           int64  `json:"size"`
	MimeType       string `json:"mime_type,omitempty"`
	WebURL         string `json:"web_url,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	ModifiedAt     string `json:"modified_at,omitempty"`
	Content        string `json:"content,omitempty"`
	ContentSkipped string `json:"content_skipped,omitempty"`
}

// textMimeTypes lists MIME type prefixes that are safe to download as text.
var textMimeTypes = []string{
	"text/",
	"application/json",
	"application/xml",
	"application/javascript",
	"application/x-yaml",
	"application/x-sh",
	"application/sql",
	"application/xhtml+xml",
}

func isTextMimeType(mimeType string) bool {
	lower := strings.ToLower(mimeType)
	for _, prefix := range textMimeTypes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func (a *getDriveFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getDriveFileParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Get file metadata.
	metadataPath := fmt.Sprintf("/me/drive/items/%s?$select=id,name,size,webUrl,createdDateTime,lastModifiedDateTime,folder,file", params.ItemID)

	var item graphDriveItem
	if err := a.conn.doRequest(ctx, http.MethodGet, metadataPath, req.Credentials, nil, &item); err != nil {
		return nil, err
	}

	detail := driveFileDetail{
		ID:         item.ID,
		Name:       item.Name,
		Size:       item.Size,
		WebURL:     item.WebURL,
		CreatedAt:  item.CreatedDateTime,
		ModifiedAt: item.ModifiedDateTime,
	}
	if item.Folder != nil {
		detail.Type = "folder"
	} else {
		detail.Type = "file"
		if item.File != nil {
			detail.MimeType = item.File.MimeType
		}
	}

	// Optionally download text content.
	if params.IncludeContent && detail.Type == "folder" {
		detail.ContentSkipped = "content download is not available for folders"
	}
	if params.IncludeContent && detail.Type == "file" {
		if detail.MimeType == "" {
			return nil, &connectors.ValidationError{
				Message: "cannot download content: file has no MIME type; only text files are supported",
			}
		}
		if !isTextMimeType(detail.MimeType) {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("cannot download content for binary file type %q; only text files are supported", detail.MimeType),
			}
		}
		contentPath := fmt.Sprintf("/me/drive/items/%s/content", params.ItemID)
		content, err := a.conn.doRequestRaw(ctx, http.MethodGet, contentPath, req.Credentials)
		if err != nil {
			return nil, err
		}
		detail.Content = content
	}

	return connectors.JSONResult(detail)
}

// validateItemID rejects item IDs containing path separators, traversal sequences,
// or URL-special characters that could be used for query/fragment injection.
func validateItemID(id string) error {
	if strings.ContainsAny(id, "/\\") {
		return &connectors.ValidationError{Message: "invalid item_id: must not contain path separators"}
	}
	if strings.Contains(id, "..") {
		return &connectors.ValidationError{Message: "invalid item_id: must not contain path traversal sequences"}
	}
	if strings.ContainsAny(id, "?#%") {
		return &connectors.ValidationError{Message: "invalid item_id: must not contain URL-special characters"}
	}
	return nil
}
