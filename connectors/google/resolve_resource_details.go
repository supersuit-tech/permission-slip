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
	case "google.list_calendar_events", "google.create_calendar_event", "google.create_meeting":
		return c.resolveCalendar(ctx, creds, params)

	// Chat
	case "google.send_chat_message":
		return c.resolveChatSpace(ctx, creds, params)

	// Drive
	case "google.delete_drive_file", "google.get_drive_file":
		return c.resolveDriveFile(ctx, creds, params)
	case "google.upload_drive_file", "google.create_drive_folder":
		return c.resolveDriveFolder(ctx, creds, params, "My Drive")
	case "google.list_drive_files", "google.search_drive":
		return c.resolveDriveFolder(ctx, creds, params, "")

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
	case "google.read_email", "google.archive_email", "google.download_attachment":
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
		Summary  string `json:"summary"`
		HTMLLink string `json:"htmlLink"`
		Start    struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"start"`
		End struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"end"`
	}
	getURL := c.calendarBaseURL + "/calendars/" + url.PathEscape(p.CalendarID) + "/events/" + url.PathEscape(p.EventID) + "?fields=summary,start,end,htmlLink"
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return nil, err
	}

	details := map[string]any{"title": resp.Summary}
	connectors.AttachResources(details, connectors.ResourceRef{
		Param: "event_id",
		ID:    p.EventID,
		Name:  resp.Summary,
		URL:   resp.HTMLLink,
	})
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
	// Best-effort canonical calendar id/name for standing-approval derivation.
	// Failures must not drop the event title/times already resolved.
	if cal, err := c.lookupCalendar(ctx, creds, p.CalendarID); err == nil {
		details["calendar_id"] = cal.ID
		name := cal.Summary
		if name == "" {
			name = cal.ID
		}
		if cal.Summary != "" {
			details["calendar_name"] = cal.Summary
		}
		requestedID := p.CalendarID
		if requestedID == "" {
			requestedID = "primary"
		}
		connectors.AttachResources(details, connectors.ResourceRef{
			Param: "calendar_id",
			ID:    requestedID,
			Name:  name,
		})
		if cal.ID != requestedID {
			connectors.AttachResources(details, connectors.ResourceRef{
				Param: "calendar_id",
				ID:    cal.ID,
				Name:  name,
			})
		}
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

	file, err := c.lookupDriveFile(ctx, creds, p.FileID)
	if err != nil {
		return nil, err
	}

	details := map[string]any{
		"file_name": file.Name,
		"mime_type": file.MimeType,
	}
	c.attachSharedDriveDetails(ctx, creds, details, file.DriveID)
	return connectors.AttachResources(details, connectors.ResourceRef{
		Param: "file_id",
		ID:    p.FileID,
		Name:  file.Name,
		URL:   file.WebViewLink,
	}), nil
}

// resolveDriveFolder fetches the human-readable name for a Drive folder_id or
// parent_id. When the ID is omitted and defaultRootName is set (upload / create
// folder), the destination is My Drive root. List/search omit the ID to mean
// "all files", so defaultRootName is left empty and no details are returned.
//
// folder_id may be a My Drive folder, a Shared Drive folder, or a Shared Drive
// ID (the drive root). files.get often succeeds for Shared Drive IDs but
// returns the generic name "Drive"; drives.get supplies the real title.
// Nested Shared Drive folders include that title so reviewers can see which
// drive the folder lives in (e.g. "2026-documents in Chiedo's assistant drive").
// A 404 from files.get still falls back to drives.get.
func (c *GoogleConnector) resolveDriveFolder(ctx context.Context, creds connectors.Credentials, params json.RawMessage, defaultRootName string) (map[string]any, error) {
	var p struct {
		FolderID string `json:"folder_id"`
		ParentID string `json:"parent_id"`
		DriveID  string `json:"drive_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid drive folder params: %w", err)
	}

	folderID := driveFolderTargetID(p.FolderID, p.ParentID)

	details := map[string]any{}
	folderParam := "folder_id"
	if p.FolderID == "" && p.ParentID != "" {
		folderParam = "parent_id"
	}

	switch {
	case folderID != "":
		if !isValidDriveID(folderID) {
			return nil, fmt.Errorf("invalid folder_id")
		}
		folder, err := c.lookupDriveFolder(ctx, creds, folderID)
		if err != nil {
			return nil, err
		}
		for k, v := range driveFolderDetails(folder.Name, folder.DriveID, folder.DriveName) {
			details[k] = v
		}
		connectors.AttachResources(details, connectors.ResourceRef{
			Param: folderParam,
			ID:    folderID,
			Name:  folder.Name,
			URL:   driveFolderURL(folderID),
		})
	case defaultRootName != "":
		for k, v := range driveFolderDetails(defaultRootName, "", "") {
			details[k] = v
		}
	}

	if p.DriveID != "" && p.DriveID != folderID {
		if !isValidDriveID(p.DriveID) {
			return nil, fmt.Errorf("invalid drive_id")
		}
		driveName, err := c.lookupSharedDriveName(ctx, creds, p.DriveID)
		if err != nil {
			return nil, err
		}
		if _, ok := details["drive_id"]; !ok {
			details["drive_id"] = p.DriveID
		}
		if driveName != "" {
			if _, ok := details["drive_name"]; !ok {
				details["drive_name"] = driveName
			}
		}
		connectors.AttachResources(details, connectors.ResourceRef{
			Param: "drive_id",
			ID:    p.DriveID,
			Name:  driveName,
			URL:   driveFolderURL(p.DriveID),
		})
	}

	if len(details) == 0 {
		return nil, nil
	}
	return details, nil
}

func driveFolderTargetID(folderID, parentID string) string {
	if folderID != "" {
		return folderID
	}
	return parentID
}

// genericSharedDriveRootName is what files.get returns for a Shared Drive's
// root folder. The real drive title only comes from drives.get.
const genericSharedDriveRootName = "Drive"

func driveFolderDetails(name, driveID, driveName string) map[string]any {
	details := map[string]any{
		"folder_name": name,
		"parent_name": name,
	}
	attachDriveFields(details, driveID, driveName)
	return details
}

func attachDriveFields(details map[string]any, driveID, driveName string) {
	if driveID != "" {
		details["drive_id"] = driveID
	}
	if driveName != "" {
		details["drive_name"] = driveName
	}
}

func (c *GoogleConnector) attachSharedDriveDetails(ctx context.Context, creds connectors.Credentials, details map[string]any, driveID string) {
	if driveID == "" {
		return
	}
	details["drive_id"] = driveID
	if driveName, err := c.lookupSharedDriveName(ctx, creds, driveID); err == nil && driveName != "" {
		details["drive_name"] = driveName
	}
}

// driveFolderLookup is the Drive API result for a folder_id / parent_id.
// DriveID is set when the folder lives on a Shared Drive (including when the
// ID itself is the Shared Drive root). DriveName is the Shared Drive title
// when the Drive API returned it.
type driveFolderLookup struct {
	Name      string
	DriveID   string
	DriveName string
}

// driveFileLookup is the Drive API result for a file_id or spreadsheet_id.
// DriveID is set when the file lives on a Shared Drive.
type driveFileLookup struct {
	Name        string
	MimeType    string
	DriveID     string
	WebViewLink string
}

func driveRootDisplayName(name string) string {
	return name + " in the / directory"
}

func driveFolderInSharedDriveDisplayName(folderName, driveName string) string {
	return folderName + " in " + driveName
}

func (c *GoogleConnector) lookupDriveFolder(ctx context.Context, creds connectors.Credentials, id string) (driveFolderLookup, error) {
	var fileResp struct {
		Name    string   `json:"name"`
		DriveID string   `json:"driveId"`
		Parents []string `json:"parents"`
	}
	q := url.Values{}
	q.Set("fields", "name,driveId,parents")
	applySupportsAllDrives(q)
	fileURL := c.driveBaseURL + "/drive/v3/files/" + url.PathEscape(id) + "?" + q.Encode()
	fileErr := c.doJSON(ctx, creds, http.MethodGet, fileURL, nil, &fileResp)
	if fileErr != nil {
		if !isGoogleNotFound(fileErr) {
			return driveFolderLookup{}, fileErr
		}
		driveName, driveErr := c.lookupSharedDriveName(ctx, creds, id)
		if driveErr != nil {
			return driveFolderLookup{}, driveErr
		}
		return driveFolderLookup{
			Name:      driveRootDisplayName(driveName),
			DriveID:   id,
			DriveName: driveName,
		}, nil
	}

	name := fileResp.Name
	isDriveRoot := (fileResp.DriveID != "" && fileResp.DriveID == id) ||
		(name == genericSharedDriveRootName && len(fileResp.Parents) == 0)
	// files.get succeeds for Shared Drive IDs (with supportsAllDrives) but
	// names the root folder "Drive". Fall through to drives.get for the title.
	if isDriveRoot {
		driveID := fileResp.DriveID
		if driveID == "" {
			driveID = id
		}
		driveName := ""
		if lookedUp, err := c.lookupSharedDriveName(ctx, creds, driveID); err == nil && lookedUp != "" {
			name = lookedUp
			driveName = lookedUp
		}
		if name == "" {
			return driveFolderLookup{}, fmt.Errorf("folder %q has no name", id)
		}
		return driveFolderLookup{Name: driveRootDisplayName(name), DriveID: driveID, DriveName: driveName}, nil
	}
	if name == "" {
		return driveFolderLookup{}, fmt.Errorf("folder %q has no name", id)
	}
	if fileResp.DriveID != "" {
		if driveName, err := c.lookupSharedDriveName(ctx, creds, fileResp.DriveID); err == nil && driveName != "" {
			return driveFolderLookup{
				Name:      driveFolderInSharedDriveDisplayName(name, driveName),
				DriveID:   fileResp.DriveID,
				DriveName: driveName,
			}, nil
		}
		return driveFolderLookup{Name: name, DriveID: fileResp.DriveID}, nil
	}
	return driveFolderLookup{Name: name}, nil
}

func (c *GoogleConnector) lookupDriveFile(ctx context.Context, creds connectors.Credentials, id string) (driveFileLookup, error) {
	var resp struct {
		Name        string `json:"name"`
		MimeType    string `json:"mimeType"`
		DriveID     string `json:"driveId"`
		WebViewLink string `json:"webViewLink"`
	}
	q := url.Values{}
	q.Set("fields", "name,mimeType,driveId,webViewLink")
	applySupportsAllDrives(q)
	getURL := c.driveBaseURL + "/drive/v3/files/" + url.PathEscape(id) + "?" + q.Encode()
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return driveFileLookup{}, err
	}
	return driveFileLookup{
		Name:        resp.Name,
		MimeType:    resp.MimeType,
		DriveID:     resp.DriveID,
		WebViewLink: resp.WebViewLink,
	}, nil
}

func (c *GoogleConnector) lookupSharedDriveName(ctx context.Context, creds connectors.Credentials, id string) (string, error) {
	var driveResp struct {
		Name string `json:"name"`
	}
	driveURL := c.driveBaseURL + "/drive/v3/drives/" + url.PathEscape(id) + "?fields=name"
	if err := c.doJSON(ctx, creds, http.MethodGet, driveURL, nil, &driveResp); err != nil {
		return "", err
	}
	if driveResp.Name == "" {
		return "", fmt.Errorf("shared drive %q has no name", id)
	}
	return driveResp.Name, nil
}

func isGoogleNotFound(err error) bool {
	var ext *connectors.ExternalError
	return errors.As(err, &ext) && ext.StatusCode == http.StatusNotFound
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

	details := map[string]any{"title": resp.Title}
	return connectors.AttachResources(details, connectors.ResourceRef{
		Param: "document_id",
		ID:    p.DocumentID,
		Name:  resp.Title,
		URL:   documentEditURL(p.DocumentID),
	}), nil
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
	// Spreadsheets are Drive files; membership is optional for the preview.
	if file, err := c.lookupDriveFile(ctx, creds, p.SpreadsheetID); err == nil {
		c.attachSharedDriveDetails(ctx, creds, details, file.DriveID)
	}
	return connectors.AttachResources(details, connectors.ResourceRef{
		Param: "spreadsheet_id",
		ID:    p.SpreadsheetID,
		Name:  resp.Properties.Title,
		URL:   spreadsheetURL(p.SpreadsheetID),
	}), nil
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
	details := map[string]any{
		"title":              resp.Title,
		"presentation_title": resp.Title,
	}
	return connectors.AttachResources(details, connectors.ResourceRef{
		Param: "presentation_id",
		ID:    p.PresentationID,
		Name:  resp.Title,
		URL:   presentationURL(p.PresentationID),
	}), nil
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
	details := map[string]any{"space_display_name": resp.DisplayName}
	return connectors.AttachResources(details, connectors.ResourceRef{
		Param: "space_name",
		ID:    p.SpaceName,
		Name:  resp.DisplayName,
		URL:   chatSpaceURL(p.SpaceName),
	}), nil
}

// resolveCalendar fetches the calendar summary (human-readable name) and
// canonical id for a calendar ID. API: GET /calendars/{calendarId}?fields=id,summary.
// When calendar_id is empty, defaults to "primary", matching listCalendarEventsParams.normalize().
func (c *GoogleConnector) resolveCalendar(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		CalendarID string `json:"calendar_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid calendar params: %w", err)
	}

	cal, err := c.fetchCalendar(ctx, creds, p.CalendarID)
	if err != nil {
		return nil, err
	}
	details := map[string]any{}
	if cal.Summary != "" {
		details["calendar_name"] = cal.Summary
	}
	if cal.ID != "" {
		details["calendar_id"] = cal.ID
	}
	if len(details) == 0 {
		return nil, fmt.Errorf("calendar %q has no name", normalizeCalendarID(p.CalendarID))
	}
	name := cal.Summary
	if name == "" {
		name = cal.ID
	}
	requestedID := normalizeCalendarID(p.CalendarID)
	refs := []connectors.ResourceRef{{
		Param: "calendar_id",
		ID:    requestedID,
		Name:  name,
	}}
	if cal.ID != "" && cal.ID != requestedID {
		refs = append(refs, connectors.ResourceRef{
			Param: "calendar_id",
			ID:    cal.ID,
			Name:  name,
		})
	}
	return connectors.AttachResources(details, refs...), nil
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

	meta, err := c.fetchEmailMetadata(ctx, creds, messageID)
	if err != nil {
		return nil, err
	}
	if p.ThreadID != "" && meta != nil {
		name, _ := meta["subject"].(string)
		if name == "" {
			name, _ = meta["from"].(string)
		}
		if name != "" {
			connectors.AttachResources(meta, connectors.ResourceRef{
				Param: "thread_id",
				ID:    p.ThreadID,
				Name:  name,
				URL:   gmailMessageURL(p.ThreadID),
			})
		}
	}
	return meta, nil
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
	if meta != nil {
		name, _ := meta["subject"].(string)
		if name == "" {
			name, _ = meta["from"].(string)
		}
		if name != "" {
			connectors.AttachResources(meta, connectors.ResourceRef{
				Param: "thread_id",
				ID:    p.ThreadID,
				Name:  name,
				URL:   gmailMessageURL(p.ThreadID),
			})
		}
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
	name, _ := details["subject"].(string)
	if name == "" {
		name, _ = details["from"].(string)
	}
	if name != "" {
		connectors.AttachResources(details, connectors.ResourceRef{
			Param: "message_id",
			ID:    messageID,
			Name:  name,
			URL:   gmailMessageURL(messageID),
		})
	}
	return details, nil
}
