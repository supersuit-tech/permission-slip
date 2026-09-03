package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// maxUploadBytes is the maximum decoded content size for file uploads (10 MB).
// This is a connector-side ceiling for receipts, PDFs, and images — well below
// Drive's object limit, and matching confluence.add_attachment.
const maxUploadBytes = 10 * 1024 * 1024

// maxUploadBase64Len is the maximum accepted content_base64 string length
// (~10 MB decoded, with base64's 4/3 expansion plus padding).
const maxUploadBase64Len = maxUploadBytes*4/3 + 4

// uploadDriveFileAction implements connectors.Action for google.upload_drive_file.
// It creates a new file in Google Drive using multipart upload.
type uploadDriveFileAction struct {
	conn *GoogleConnector
}

// uploadDriveFileParams is the user-facing parameter schema.
type uploadDriveFileParams struct {
	Name          string `json:"name"`
	Content       string `json:"content"`
	ContentBase64 string `json:"content_base64"`
	MimeType      string `json:"mime_type"`
	FolderID      string `json:"folder_id"`
}

func (p *uploadDriveFileParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	hasText := p.Content != ""
	hasBinary := strings.TrimSpace(p.ContentBase64) != ""
	if hasText && hasBinary {
		return &connectors.ValidationError{Message: "content and content_base64 are mutually exclusive; provide exactly one"}
	}
	if !hasText && !hasBinary {
		return &connectors.ValidationError{Message: "missing required parameter: content or content_base64"}
	}
	if hasText && len(p.Content) > maxUploadBytes {
		return &connectors.ValidationError{
			Message: "content exceeds maximum upload size of 10 MB",
		}
	}
	if hasBinary && len(p.ContentBase64) > maxUploadBase64Len {
		return &connectors.ValidationError{
			Message: "content_base64 exceeds maximum upload size of 10 MB (decoded)",
		}
	}
	if p.FolderID != "" && !isValidDriveID(p.FolderID) {
		return &connectors.ValidationError{Message: "folder_id contains invalid characters; expected alphanumeric ID"}
	}
	if err := validateMimeType(p.MimeType); err != nil {
		return err
	}
	return nil
}

func (p *uploadDriveFileParams) normalize() {
	if p.MimeType != "" {
		return
	}
	if strings.TrimSpace(p.ContentBase64) != "" {
		p.MimeType = mimeTypeFromFilename(p.Name)
		return
	}
	p.MimeType = "text/plain"
}

func (p *uploadDriveFileParams) payload() ([]byte, error) {
	if strings.TrimSpace(p.ContentBase64) != "" {
		decoded, err := decodeBase64Bytes(p.ContentBase64)
		if err != nil {
			return nil, &connectors.ValidationError{Message: "content_base64 is not valid base64-encoded data"}
		}
		if len(decoded) > maxUploadBytes {
			return nil, &connectors.ValidationError{
				Message: "content_base64 exceeds maximum upload size of 10 MB (decoded)",
			}
		}
		return decoded, nil
	}
	return []byte(p.Content), nil
}

// driveUploadMetadata is the metadata part of the multipart upload.
type driveUploadMetadata struct {
	Name    string   `json:"name"`
	Parents []string `json:"parents,omitempty"`
}

// driveUploadResponse is the Google Drive API response from files.create.
type driveUploadResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	WebViewLink string `json:"webViewLink"`
}

// Execute uploads a file to Google Drive and returns the created file metadata.
func (a *uploadDriveFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params uploadDriveFileParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	content, err := params.payload()
	if err != nil {
		return nil, err
	}

	token, ok := req.Credentials.Get(credKeyAccessToken)
	if !ok || token == "" {
		return nil, &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	// Build multipart request body.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Part 1: JSON metadata.
	meta := driveUploadMetadata{Name: params.Name}
	if params.FolderID != "" {
		meta.Parents = []string{params.FolderID}
	}
	metaHeader := make(textproto.MIMEHeader)
	metaHeader.Set("Content-Type", "application/json")
	metaPart, err := writer.CreatePart(metaHeader)
	if err != nil {
		return nil, fmt.Errorf("creating metadata part: %w", err)
	}
	if err := json.NewEncoder(metaPart).Encode(meta); err != nil {
		return nil, fmt.Errorf("encoding metadata: %w", err)
	}

	// Part 2: File content.
	contentHeader := make(textproto.MIMEHeader)
	contentHeader.Set("Content-Type", params.MimeType)
	contentPart, err := writer.CreatePart(contentHeader)
	if err != nil {
		return nil, fmt.Errorf("creating content part: %w", err)
	}
	if _, err := contentPart.Write(content); err != nil {
		return nil, fmt.Errorf("writing content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	// Send the multipart upload request.
	uploadURL := a.conn.driveBaseURL + "/upload/drive/v3/files?uploadType=multipart&fields=id,name,webViewLink"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := a.conn.client.Do(httpReq)
	if err != nil {
		return nil, wrapHTTPError(err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return nil, err
	}

	var uploadResp driveUploadResponse
	if err := json.Unmarshal(respBytes, &uploadResp); err != nil {
		return nil, &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Google API response",
		}
	}

	result := map[string]string{
		"id":   uploadResp.ID,
		"name": uploadResp.Name,
	}
	if uploadResp.WebViewLink != "" {
		result["web_view_link"] = uploadResp.WebViewLink
	}
	return connectors.JSONResult(result)
}

// validateMimeType rejects newline injection and unparseable MIME types.
func validateMimeType(mt string) error {
	if mt == "" {
		return nil
	}
	if strings.ContainsAny(mt, "\r\n") {
		return &connectors.ValidationError{Message: "mime_type must not contain newline characters"}
	}
	if _, _, err := mime.ParseMediaType(mt); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("mime_type is not a valid MIME type: %v", err)}
	}
	return nil
}

// mimeTypeFromFilename infers a MIME type from a file name for binary uploads.
func mimeTypeFromFilename(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return "application/octet-stream"
	}
	if mt := mime.TypeByExtension(ext); mt != "" {
		parsed, _, err := mime.ParseMediaType(mt)
		if err == nil && parsed != "" {
			return parsed
		}
		return mt
	}
	switch ext {
	case ".heic":
		return "image/heic"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
