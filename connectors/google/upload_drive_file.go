package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// maxUploadBytes is the maximum content size for file uploads (4 MB).
const maxUploadBytes = 4 * 1024 * 1024

// uploadDriveFileAction implements connectors.Action for google.upload_drive_file.
// It creates a new file in Google Drive using multipart upload.
type uploadDriveFileAction struct {
	conn *GoogleConnector
}

// uploadDriveFileParams is the user-facing parameter schema.
type uploadDriveFileParams struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	MimeType string `json:"mime_type"`
	FolderID string `json:"folder_id"`
}

func (p *uploadDriveFileParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.Content == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content"}
	}
	if len(p.Content) > maxUploadBytes {
		return &connectors.ValidationError{
			Message: "content exceeds maximum upload size of 4 MB",
		}
	}
	if p.FolderID != "" && !isValidDriveID(p.FolderID) {
		return &connectors.ValidationError{Message: "folder_id contains invalid characters; expected alphanumeric ID"}
	}
	return nil
}

func (p *uploadDriveFileParams) normalize() {
	if p.MimeType == "" {
		p.MimeType = "text/plain"
	}
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

// Execute uploads a text file to Google Drive and returns the created file metadata.
func (a *uploadDriveFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params uploadDriveFileParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

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
	if _, err := contentPart.Write([]byte(params.Content)); err != nil {
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
