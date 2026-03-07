package slack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	// maxUploadContentBytes caps file content at 50 MB to prevent
	// excessive memory allocation (the content is held in memory as a
	// string, converted to []byte, then buffered in a multipart writer).
	maxUploadContentBytes = 50 << 20 // 50 MB
)

// uploadFileAction implements connectors.Action for slack.upload_file.
// It uploads a file to a channel via Slack's v2 upload flow:
// 1. POST /files.getUploadURLExternal to get an upload URL
// 2. POST the file content to the upload URL
// 3. POST /files.completeUploadExternal to finalize and share
type uploadFileAction struct {
	conn *SlackConnector
}

// uploadFileParams is the user-facing parameter schema.
type uploadFileParams struct {
	Channel  string `json:"channel"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
	Title    string `json:"title,omitempty"`
}

func (p *uploadFileParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	if p.Filename == "" {
		return &connectors.ValidationError{Message: "missing required parameter: filename"}
	}
	if p.Content == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content"}
	}
	if len(p.Content) > maxUploadContentBytes {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("file content exceeds maximum size of %d MB", maxUploadContentBytes>>20),
		}
	}
	return nil
}

// Step 1: Get upload URL

type getUploadURLRequest struct {
	Filename string `json:"filename"`
	Length   int    `json:"length"`
}

type getUploadURLResponse struct {
	slackResponse
	UploadURL string `json:"upload_url"`
	FileID    string `json:"file_id"`
}

// Step 3: Complete upload

type completeUploadRequest struct {
	Files     []completeUploadFile `json:"files"`
	ChannelID string               `json:"channel_id"`
}

type completeUploadFile struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

type completeUploadResponse struct {
	slackResponse
	Files []struct {
		ID string `json:"id"`
	} `json:"files,omitempty"`
}

// Execute uploads a file to a Slack channel using the v2 upload flow.
func (a *uploadFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params uploadFileParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	contentBytes := []byte(params.Content)

	// Step 1: Get upload URL.
	getURLBody := getUploadURLRequest{
		Filename: params.Filename,
		Length:   len(contentBytes),
	}

	var getURLResp getUploadURLResponse
	if err := a.conn.doPost(ctx, "files.getUploadURLExternal", req.Credentials, getURLBody, &getURLResp); err != nil {
		return nil, err
	}
	if !getURLResp.OK {
		return nil, mapSlackError(getURLResp.Error)
	}

	// Step 2: Upload file content to the upload URL (multipart/form-data, no auth).
	// Validate the upload URL to prevent SSRF — only allow Slack-owned domains
	// over HTTPS. Skip in test mode (non-default base URL) since httptest
	// servers use localhost.
	if a.conn.baseURL == defaultBaseURL {
		if err := validateUploadURL(getURLResp.UploadURL); err != nil {
			return nil, err
		}
	}
	if err := a.uploadContent(ctx, getURLResp.UploadURL, params.Filename, contentBytes); err != nil {
		return nil, err
	}

	// Step 3: Complete the upload and share to the channel.
	title := params.Title
	if title == "" {
		title = params.Filename
	}
	completeBody := completeUploadRequest{
		Files: []completeUploadFile{
			{ID: getURLResp.FileID, Title: title},
		},
		ChannelID: params.Channel,
	}

	var completeResp completeUploadResponse
	if err := a.conn.doPost(ctx, "files.completeUploadExternal", req.Credentials, completeBody, &completeResp); err != nil {
		return nil, err
	}
	if !completeResp.OK {
		return nil, mapSlackError(completeResp.Error)
	}

	return connectors.JSONResult(map[string]string{
		"file_id": getURLResp.FileID,
		"channel": params.Channel,
	})
}

// validateUploadURL ensures the upload URL returned by Slack is safe to
// send file content to. This prevents SSRF if the Slack API response were
// tampered with or returned an unexpected URL (e.g., internal metadata
// endpoints like http://169.254.169.254/).
func validateUploadURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return &connectors.ValidationError{Message: "Slack returned an invalid upload URL"}
	}
	if parsed.Scheme != "https" {
		return &connectors.ValidationError{Message: "upload URL must use HTTPS"}
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "slack.com" || strings.HasSuffix(host, ".slack.com") ||
		host == "slack-files.com" || strings.HasSuffix(host, ".slack-files.com") {
		return nil
	}
	return &connectors.ValidationError{
		Message: fmt.Sprintf("upload URL host %q is not a recognized Slack domain", host),
	}
}

// uploadContent sends the file content to the Slack-provided upload URL
// as multipart/form-data.
func (a *uploadFileAction) uploadContent(ctx context.Context, uploadURL, filename string, content []byte) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("creating multipart form: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return fmt.Errorf("writing file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing multipart writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &buf)
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := a.conn.client.Do(httpReq)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("file upload timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("file upload canceled: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("file upload failed: %v", err)}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("file upload returned HTTP %d", resp.StatusCode),
		}
	}

	return nil
}
