package confluence

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	// maxAttachmentBytes is the maximum decoded attachment size we will accept
	// before uploading. Confluence's own limit is 250 MB, but we cap at 10 MB
	// to avoid OOM from large base64 payloads decoding entirely into memory.
	// Base64 encoding inflates size by ~33%, so a 10 MB file → ~13.5 MB base64.
	maxAttachmentBytes = 10 << 20 // 10 MB
)

// addAttachmentAction implements connectors.Action for confluence.add_attachment.
// It uploads an attachment to a page via POST /wiki/api/v2/pages/{page_id}/attachments
// using multipart/form-data. The file content is provided as base64-encoded data.
type addAttachmentAction struct {
	conn *ConfluenceConnector
}

type addAttachmentParams struct {
	PageID      string `json:"page_id"`
	Filename    string `json:"filename"`
	ContentB64  string `json:"content_base64"`
	MediaType   string `json:"media_type,omitempty"`
	Comment     string `json:"comment,omitempty"`
}

func (p *addAttachmentParams) validate() error {
	p.PageID = strings.TrimSpace(p.PageID)
	if p.PageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: page_id"}
	}
	p.Filename = strings.TrimSpace(p.Filename)
	if p.Filename == "" {
		return &connectors.ValidationError{Message: "missing required parameter: filename"}
	}
	// Prevent path traversal in filename.
	if strings.Contains(p.Filename, "/") || strings.Contains(p.Filename, "\\") || strings.Contains(p.Filename, "..") {
		return &connectors.ValidationError{Message: "filename must not contain path separators or traversal sequences"}
	}
	if p.ContentB64 == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content_base64"}
	}
	// Enforce a hard cap on the base64 payload length (~13.5 MB → 10 MB decoded).
	// base64 expands data by ~33%, so maxAttachmentBytes * 4/3 covers the encoded size.
	maxB64Len := maxAttachmentBytes * 4 / 3
	if len(p.ContentB64) > maxB64Len {
		return &connectors.ValidationError{
			Message: "content_base64 exceeds maximum allowed size (decoded limit: 10 MB)",
		}
	}
	// Validate media_type if provided: must be a parseable MIME type with no
	// CR/LF characters (which could inject headers into the multipart body).
	if p.MediaType != "" {
		if strings.ContainsAny(p.MediaType, "\r\n") {
			return &connectors.ValidationError{Message: "media_type must not contain newline characters"}
		}
		if _, _, err := mime.ParseMediaType(p.MediaType); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("media_type is not a valid MIME type: %v", err)}
		}
	}
	return nil
}

type addAttachmentResponse struct {
	Results []attachmentItem `json:"results"`
}

func (a *addAttachmentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addAttachmentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Decode base64 content.
	fileContent, err := base64.StdEncoding.DecodeString(params.ContentB64)
	if err != nil {
		// Try URL-safe base64 as fallback.
		fileContent, err = base64.URLEncoding.DecodeString(params.ContentB64)
		if err != nil {
			return nil, &connectors.ValidationError{Message: "content_base64 is not valid base64-encoded data"}
		}
	}

	// Build multipart body.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	mediaType := params.MediaType
	if mediaType == "" {
		mediaType = mimeTypeFromFilename(params.Filename)
	}

	// Add file field with correct content-type header.
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, params.Filename))
	h.Set("Content-Type", mediaType)
	fw, err := mw.CreatePart(h)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("creating multipart field: %v", err)}
	}
	if _, err := io.Copy(fw, bytes.NewReader(fileContent)); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("writing file content: %v", err)}
	}

	if params.Comment != "" {
		if err := mw.WriteField("comment", params.Comment); err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("writing comment field: %v", err)}
		}
	}

	if err := mw.Close(); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("closing multipart writer: %v", err)}
	}

	base, err := a.conn.apiBase(req.Credentials)
	if err != nil {
		return nil, err
	}

	email, ok := req.Credentials.Get("email")
	if !ok || email == "" {
		return nil, &connectors.ValidationError{Message: "email credential is missing or empty"}
	}
	token, ok := req.Credentials.Get("api_token")
	if !ok || token == "" {
		return nil, &connectors.ValidationError{Message: "api_token credential is missing or empty"}
	}

	path := "/pages/" + url.PathEscape(params.PageID) + "/attachments"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, &buf)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("creating request: %v", err)}
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(email+":"+token)))
	// Confluence requires this header to update existing attachments without error.
	httpReq.Header.Set("X-Atlassian-Token", "no-check")

	resp, err := a.conn.client.Do(httpReq)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("Confluence API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return nil, err
	}

	var attachResp addAttachmentResponse
	if err := json.Unmarshal(respBytes, &attachResp); err != nil {
		return nil, &connectors.ExternalError{Message: "failed to parse attachment response"}
	}

	if len(attachResp.Results) == 0 {
		return connectors.JSONResult(map[string]string{
			"page_id":  params.PageID,
			"filename": params.Filename,
			"status":   "uploaded",
		})
	}

	att := attachResp.Results[0]
	return connectors.JSONResult(map[string]interface{}{
		"id":         att.ID,
		"title":      att.Title,
		"media_type": att.MediaType,
		"file_size":  att.FileSize,
	})
}

// mimeTypeFromFilename returns a MIME type based on file extension.
func mimeTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".pdf":  "application/pdf",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".txt":  "text/plain",
		".csv":  "text/csv",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".zip":  "application/zip",
		".json": "application/json",
		".xml":  "application/xml",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
	}
	if mt, ok := mimeTypes[ext]; ok {
		return mt
	}
	return "application/octet-stream"
}
