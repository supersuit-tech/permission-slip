package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ResolveResourceDetails fetches human-readable metadata for resources
// referenced by opaque IDs in Google action parameters. Each action type
// maps to a specific Google API GET call. Errors are non-fatal — the caller
// stores the approval without details on failure.
func (c *GoogleConnector) ResolveResourceDetails(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	switch actionType {
	// Calendar
	case "google.delete_calendar_event", "google.update_calendar_event":
		return c.resolveCalendarEvent(ctx, creds, params)
	case "google.list_calendar_events":
		return c.resolveCalendar(ctx, creds, params)

	// Chat
	case "google.send_chat_message":
		return c.resolveChatSpace(ctx, creds, params)

	// Drive
	case "google.delete_drive_file", "google.get_drive_file":
		return c.resolveDriveFile(ctx, creds, params)

	// Docs
	case "google.get_document", "google.update_document":
		return c.resolveDocument(ctx, creds, params)

	// Sheets
	case "google.sheets_read_range", "google.sheets_write_range",
		"google.sheets_append_rows", "google.sheets_list_sheets":
		return c.resolveSpreadsheet(ctx, creds, params)

	// Slides
	case "google.get_presentation", "google.add_slide":
		return c.resolvePresentation(ctx, creds, params)

	// Gmail
	case "google.read_email", "google.archive_email":
		return c.resolveEmail(ctx, creds, params)
	case "google.send_email_reply":
		return c.resolveEmailReply(ctx, creds, params)

	default:
		return nil, nil
	}
}

// ── Calendar ────────────────────────────────────────────────────────────────

func (c *GoogleConnector) resolveCalendarEvent(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		EventID    string `json:"event_id"`
		CalendarID string `json:"calendar_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.EventID == "" {
		return nil, fmt.Errorf("missing event_id")
	}
	if p.CalendarID == "" {
		p.CalendarID = "primary"
	}

	var resp struct {
		Summary string `json:"summary"`
		Start   struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"start"`
		End struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"end"`
	}
	getURL := c.calendarBaseURL + "/calendars/" + url.PathEscape(p.CalendarID) + "/events/" + url.PathEscape(p.EventID) + "?fields=summary,start,end"
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return nil, err
	}

	details := map[string]any{"title": resp.Summary}
	startTime := resp.Start.DateTime
	if startTime == "" {
		startTime = resp.Start.Date
	}
	endTime := resp.End.DateTime
	if endTime == "" {
		endTime = resp.End.Date
	}
	if startTime != "" {
		details["start_time"] = startTime
	}
	if endTime != "" {
		details["end_time"] = endTime
	}
	return details, nil
}

// ── Drive ───────────────────────────────────────────────────────────────────

func (c *GoogleConnector) resolveDriveFile(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		FileID string `json:"file_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.FileID == "" {
		return nil, fmt.Errorf("missing file_id")
	}

	var resp struct {
		Name     string `json:"name"`
		MimeType string `json:"mimeType"`
	}
	getURL := c.driveBaseURL + "/drive/v3/files/" + url.PathEscape(p.FileID) + "?fields=" + url.QueryEscape("name,mimeType")
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return nil, err
	}

	return map[string]any{
		"file_name": resp.Name,
		"mime_type": resp.MimeType,
	}, nil
}

// ── Docs ────────────────────────────────────────────────────────────────────

func (c *GoogleConnector) resolveDocument(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		DocumentID string `json:"document_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.DocumentID == "" {
		return nil, fmt.Errorf("missing document_id")
	}

	var resp struct {
		Title string `json:"title"`
	}
	getURL := c.docsBaseURL + "/v1/documents/" + url.PathEscape(p.DocumentID) + "?fields=title"
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return nil, err
	}

	return map[string]any{"title": resp.Title}, nil
}

// ── Sheets ──────────────────────────────────────────────────────────────────

func (c *GoogleConnector) resolveSpreadsheet(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		SpreadsheetID string `json:"spreadsheet_id"`
		Range         string `json:"range"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.SpreadsheetID == "" {
		return nil, fmt.Errorf("missing spreadsheet_id")
	}

	var resp struct {
		Properties struct {
			Title string `json:"title"`
		} `json:"properties"`
	}
	getURL := c.sheetsBaseURL + "/spreadsheets/" + url.PathEscape(p.SpreadsheetID) + "?fields=" + url.QueryEscape("properties.title")
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return nil, err
	}

	details := map[string]any{"title": resp.Properties.Title}
	if p.Range != "" {
		details["range"] = p.Range
	}
	return details, nil
}

// ── Slides ──────────────────────────────────────────────────────────────────

func (c *GoogleConnector) resolvePresentation(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		PresentationID string `json:"presentation_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.PresentationID == "" {
		return nil, fmt.Errorf("missing presentation_id")
	}

	var resp struct {
		Title string `json:"title"`
	}
	getURL := c.slidesBaseURL + "/v1/presentations/" + url.PathEscape(p.PresentationID) + "?fields=title"
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return nil, err
	}

	// Include presentation_title so templates can disambiguate from other "title"
	// params (e.g. optional slide title on google.add_slide).
	return map[string]any{
		"title":              resp.Title,
		"presentation_title": resp.Title,
	}, nil
}

// resolveChatSpace fetches the Chat space display name for approval summaries.
// API: GET https://chat.googleapis.com/v1/{name}?fields=displayName
func (c *GoogleConnector) resolveChatSpace(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		SpaceName string `json:"space_name"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.SpaceName == "" {
		return nil, fmt.Errorf("missing space_name")
	}
	if _, err := validateChatSpaceName(p.SpaceName); err != nil {
		switch {
		case errors.Is(err, errChatSpaceNotPrefixed):
			return nil, fmt.Errorf("space_name must start with 'spaces/'")
		case errors.Is(err, errChatSpaceEmptyID), errors.Is(err, errChatSpaceInvalidChars):
			return nil, fmt.Errorf("invalid space_name")
		default:
			return nil, err
		}
	}

	var resp struct {
		DisplayName string `json:"displayName"`
	}
	getURL := c.chatBaseURL + "/v1/" + p.SpaceName + "?fields=displayName"
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return nil, err
	}
	return map[string]any{"space_display_name": resp.DisplayName}, nil
}

// resolveCalendar fetches the calendar summary (human-readable name) for a calendar ID.
// API: GET {calendarBaseURL}/calendars/{calendarId}?fields=summary (Calendar API v3).
// When calendar_id is empty, defaults to "primary", matching listCalendarEventsParams.normalize().
func (c *GoogleConnector) resolveCalendar(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		CalendarID string `json:"calendar_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid calendar params: %w", err)
	}
	if p.CalendarID == "" {
		p.CalendarID = "primary"
	}

	var resp struct {
		Summary string `json:"summary"`
	}
	getURL := c.calendarBaseURL + "/calendars/" + url.PathEscape(p.CalendarID) + "?fields=summary"
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return nil, err
	}
	return map[string]any{"calendar_name": resp.Summary}, nil
}

// ── Gmail ───────────────────────────────────────────────────────────────────

func (c *GoogleConnector) resolveEmail(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	// read_email uses message_id; archive_email uses thread_id and fetches first message
	var p struct {
		MessageID string `json:"message_id"`
		ThreadID  string `json:"thread_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	messageID := p.MessageID
	if messageID == "" && p.ThreadID != "" {
		// Fetch the thread to get the first message's ID.
		// Thread IDs and message IDs are separate namespaces in Gmail.
		var thread struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
		}
		threadURL := c.gmailBaseURL + "/gmail/v1/users/me/threads/" + url.PathEscape(p.ThreadID) + "?fields=" + url.QueryEscape("messages(id)")
		if err := c.doJSON(ctx, creds, http.MethodGet, threadURL, nil, &thread); err != nil {
			return nil, err
		}
		if len(thread.Messages) > 0 {
			messageID = thread.Messages[0].ID
		}
	}
	if messageID == "" {
		if p.ThreadID != "" {
			return nil, fmt.Errorf("thread %q has no messages", p.ThreadID)
		}
		return nil, fmt.Errorf("missing message_id or thread_id")
	}

	return c.fetchEmailMetadata(ctx, creds, messageID)
}

func (c *GoogleConnector) resolveEmailReply(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		ThreadID  string `json:"thread_id"`
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.MessageID == "" {
		return nil, fmt.Errorf("missing message_id")
	}
	meta, err := c.fetchEmailMetadata(ctx, creds, p.MessageID)
	if err != nil {
		return nil, err
	}
	if p.ThreadID == "" {
		return meta, nil
	}
	thread, err := c.buildGmailEmailThread(ctx, creds, p.ThreadID)
	if err != nil {
		return meta, nil // non-fatal: keep subject/from for display template
	}
	extra := connectors.EmailThreadDetailsMap(thread)
	if extra == nil {
		return meta, nil
	}
	if meta == nil {
		return extra, nil
	}
	for k, v := range extra {
		meta[k] = v
	}
	return meta, nil
}

func (c *GoogleConnector) fetchEmailMetadata(ctx context.Context, creds connectors.Credentials, messageID string) (map[string]any, error) {
	var resp struct {
		Payload struct {
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
		} `json:"payload"`
	}
	getURL := c.gmailBaseURL + "/gmail/v1/users/me/messages/" + url.PathEscape(messageID) + "?format=metadata&metadataHeaders=Subject&metadataHeaders=From"
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return nil, err
	}

	details := map[string]any{}
	for _, h := range resp.Payload.Headers {
		switch h.Name {
		case "Subject":
			details["subject"] = h.Value
		case "From":
			details["from"] = h.Value
		}
	}
	if len(details) == 0 {
		return nil, nil
	}
	return details, nil
}
