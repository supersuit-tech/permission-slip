// Package google implements the Google connector for the Permission Slip
// connector execution layer. It uses Google REST APIs (Gmail, Calendar, Slides, Sheets, Docs, Chat, Drive)
// with plain net/http and OAuth 2.0 access tokens provided by the platform.
//
// The connector exposes 29 actions covering email (send, reply, read, list, archive), calendar (create,
// list, update, delete, meetings), Slides, Sheets, Docs, Chat, and Drive (list, get,
// upload, delete, search, create folder).
package google

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultGmailBaseURL    = "https://gmail.googleapis.com"
	defaultCalendarBaseURL = "https://www.googleapis.com/calendar/v3"
	defaultSlidesBaseURL   = "https://slides.googleapis.com"
	defaultSheetsBaseURL   = "https://sheets.googleapis.com/v4"
	defaultDocsBaseURL     = "https://docs.googleapis.com"
	defaultDriveBaseURL    = "https://www.googleapis.com"
	defaultChatBaseURL     = "https://chat.googleapis.com"
	defaultTimeout         = 30 * time.Second
	credKeyAccessToken     = "access_token"

	// defaultRetryAfter is used when the Google API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes is the maximum response body size we'll read from
	// Google APIs. This prevents OOM from unexpectedly large responses.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// GoogleConnector owns the shared HTTP client and base URLs used by all
// Google actions. Actions hold a pointer back to the connector to access
// these shared resources.
type GoogleConnector struct {
	client          *http.Client
	gmailBaseURL    string
	calendarBaseURL string
	slidesBaseURL   string
	sheetsBaseURL   string
	docsBaseURL     string
	driveBaseURL    string
	chatBaseURL     string
}

// New creates a GoogleConnector with sensible defaults.
func New() *GoogleConnector {
	return &GoogleConnector{
		client:          &http.Client{Timeout: defaultTimeout},
		gmailBaseURL:    defaultGmailBaseURL,
		calendarBaseURL: defaultCalendarBaseURL,
		slidesBaseURL:   defaultSlidesBaseURL,
		sheetsBaseURL:   defaultSheetsBaseURL,
		docsBaseURL:     defaultDocsBaseURL,
		driveBaseURL:    defaultDriveBaseURL,
		chatBaseURL:     defaultChatBaseURL,
	}
}

// newForTest creates a GoogleConnector that points at a test server.
func newForTest(client *http.Client, gmailBaseURL, calendarBaseURL, sheetsBaseURL string) *GoogleConnector {
	return &GoogleConnector{
		client:          client,
		gmailBaseURL:    gmailBaseURL,
		calendarBaseURL: calendarBaseURL,
		sheetsBaseURL:   sheetsBaseURL,
	}
}

func newForTestWithChat(client *http.Client, gmailBaseURL, calendarBaseURL, chatBaseURL string) *GoogleConnector {
	return &GoogleConnector{
		client:          client,
		gmailBaseURL:    gmailBaseURL,
		calendarBaseURL: calendarBaseURL,
		chatBaseURL:     chatBaseURL,
	}
}

func newForTestWithSlides(client *http.Client, gmailBaseURL, calendarBaseURL, slidesBaseURL string) *GoogleConnector {
	return &GoogleConnector{
		client:          client,
		gmailBaseURL:    gmailBaseURL,
		calendarBaseURL: calendarBaseURL,
		slidesBaseURL:   slidesBaseURL,
	}
}

// newDriveForTest creates a GoogleConnector with only driveBaseURL set, for Drive action tests.
func newDriveForTest(client *http.Client, driveBaseURL string) *GoogleConnector {
	return &GoogleConnector{
		client:       client,
		driveBaseURL: driveBaseURL,
	}
}

// newForTestDocs creates a GoogleConnector that points at a test server for
// Google Docs and Drive API calls.
func newForTestDocs(client *http.Client, docsBaseURL, driveBaseURL string) *GoogleConnector {
	return &GoogleConnector{
		client:       client,
		docsBaseURL:  docsBaseURL,
		driveBaseURL: driveBaseURL,
	}
}

// newCalendarForTest creates a GoogleConnector with only calendarBaseURL set,
// for Calendar action tests.
func newCalendarForTest(client *http.Client, calendarBaseURL string) *GoogleConnector {
	return &GoogleConnector{
		client:          client,
		calendarBaseURL: calendarBaseURL,
	}
}

// newGmailForTest creates a GoogleConnector with only gmailBaseURL set,
// for Gmail action tests.
func newGmailForTest(client *http.Client, gmailBaseURL string) *GoogleConnector {
	return &GoogleConnector{
		client:       client,
		gmailBaseURL: gmailBaseURL,
	}
}

// ID returns "google", matching the connectors.id in the database.
func (c *GoogleConnector) ID() string { return "google" }

// Actions returns the registered action handlers keyed by action_type.
func (c *GoogleConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"google.send_email":            &sendEmailAction{conn: c},
		"google.list_emails":           &listEmailsAction{conn: c},
		"google.create_calendar_event": &createCalendarEventAction{conn: c},
		"google.list_calendar_events":  &listCalendarEventsAction{conn: c},
		"google.create_presentation":   &createPresentationAction{conn: c},
		"google.get_presentation":      &getPresentationAction{conn: c},
		"google.add_slide":             &addSlideAction{conn: c},
		"google.sheets_read_range":     &sheetsReadRangeAction{conn: c},
		"google.sheets_write_range":    &sheetsWriteRangeAction{conn: c},
		"google.sheets_append_rows":    &sheetsAppendRowsAction{conn: c},
		"google.sheets_list_sheets":    &sheetsListSheetsAction{conn: c},
		"google.create_document":       &createDocumentAction{conn: c},
		"google.get_document":          &getDocumentAction{conn: c},
		"google.update_document":       &updateDocumentAction{conn: c},
		"google.list_documents":        &listDocumentsAction{conn: c},
		"google.list_drive_files":      &listDriveFilesAction{conn: c},
		"google.get_drive_file":        &getDriveFileAction{conn: c},
		"google.upload_drive_file":     &uploadDriveFileAction{conn: c},
		"google.delete_drive_file":     &deleteDriveFileAction{conn: c},
		"google.send_chat_message":     &sendChatMessageAction{conn: c},
		"google.list_chat_spaces":      &listChatSpacesAction{conn: c},
		"google.create_meeting":        &createMeetingAction{conn: c},
		"google.update_calendar_event": &updateCalendarEventAction{conn: c},
		"google.delete_calendar_event": &deleteCalendarEventAction{conn: c},
		"google.search_drive":          &searchDriveAction{conn: c},
		"google.create_drive_folder":   &createDriveFolderAction{conn: c},
		"google.send_email_reply":      &sendEmailReplyAction{conn: c},
		"google.read_email":            &readEmailAction{conn: c},
		"google.archive_email":         &archiveEmailAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token. Since tokens are provided by the platform's
// OAuth infrastructure, format validation is minimal.
func (c *GoogleConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	return nil
}

// doJSON is the shared request lifecycle for Google API calls that send and
// receive JSON. It marshals reqBody as JSON, sends the request with OAuth
// bearer auth, handles rate limiting and timeouts, and unmarshals the
// response into respBody.
func (c *GoogleConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, url string, reqBody, respBody any) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return wrapHTTPError(err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Google API response",
			}
		}
	}

	return nil
}

// doRawGet performs a GET request and returns the response body as a string.
// Used for Drive file export/download endpoints that return non-JSON content.
func (c *GoogleConnector) doRawGet(ctx context.Context, creds connectors.Credentials, rawURL string) (string, error) {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return "", &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", wrapHTTPError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, body); err != nil {
		return "", err
	}

	return string(body), nil
}

// wrapHTTPError converts HTTP client errors into typed connector errors.
// This centralizes the timeout/cancel/external error mapping so it doesn't
// need to be duplicated across doJSON, doRawGet, and multipart upload.
func wrapHTTPError(err error) error {
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("Google API request timed out: %v", err)}
	}
	if errors.Is(err, context.Canceled) {
		return &connectors.CanceledError{Message: "Google API request canceled"}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("Google API request failed: %v", err)}
}

// checkResponse maps HTTP status codes to typed connector errors.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract a Google API error message.
	var apiErr struct {
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	msg := "Google API error"
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		msg = apiErr.Error.Message
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Google auth error: %s", msg)}
	case http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Google permission denied: %s", msg)}
	case http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Google API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Google API error (HTTP %d): %s", statusCode, msg),
		}
	}
}
