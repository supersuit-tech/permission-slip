package google

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// googleSharedDriveMetadataActions are Drive and Sheets actions whose target
// can be resolved to a verified Shared Drive ID for $meta.drive_id matching.
var googleSharedDriveMetadataActions = map[string]struct{}{
	"google.upload_drive_file":   {},
	"google.create_drive_folder": {},
	"google.list_drive_files":    {},
	"google.search_drive":        {},
	"google.get_drive_file":      {},
	"google.sheets_read_range":   {},
	"google.sheets_write_range":  {},
	"google.sheets_append_rows":  {},
	"google.sheets_list_sheets":  {},
}

// googleCalendarWriteMetadataActions are Calendar writes whose target calendar
// can be resolved to a verified canonical ID for $meta.calendar_id matching.
var googleCalendarWriteMetadataActions = map[string]struct{}{
	"google.create_calendar_event": {},
	"google.update_calendar_event": {},
	"google.delete_calendar_event": {},
	"google.create_meeting":        {},
}

// googleDriveMetaConstraintFields are valid $meta keys for Shared Drive actions.
var googleDriveMetaConstraintFields = []string{"drive_id"}

// googleCalendarMetaConstraintFields are valid $meta keys for Calendar write actions.
var googleCalendarMetaConstraintFields = []string{"calendar_id"}

// ResolveConstraintMetadata returns verified Drive membership or Calendar
// identity for standing approval matching.
//
// drive_id is the Shared Drive the target lives on (folder, file, spreadsheet,
// or a verified drive_id list/search parameter). My Drive targets omit drive_id
// so a Shared Drive constraint does not match.
//
// calendar_id is the canonical Calendar API id (typically an email), so
// "primary" and the primary calendar's email resolve to the same value.
// Lookup failure is unavailable (fail-closed).
func (c *GoogleConnector) ResolveConstraintMetadata(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	if _, ok := googleCalendarWriteMetadataActions[actionType]; ok {
		return c.resolveCalendarConstraintMetadata(ctx, params, creds)
	}
	if _, ok := googleSharedDriveMetadataActions[actionType]; !ok {
		return nil, connectors.ErrConstraintMetadataUnavailable
	}

	switch actionType {
	case "google.list_drive_files", "google.search_drive":
		return c.resolveListScopeDriveMeta(ctx, params, creds)
	case "google.get_drive_file":
		return c.resolveFileDriveMeta(ctx, params, creds)
	case "google.sheets_read_range", "google.sheets_write_range",
		"google.sheets_append_rows", "google.sheets_list_sheets":
		return c.resolveSpreadsheetDriveMeta(ctx, params, creds)
	default:
		return c.resolveFolderDriveMeta(ctx, params, creds)
	}
}

func (c *GoogleConnector) resolveCalendarConstraintMetadata(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var p struct {
		CalendarID string `json:"calendar_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("%w: invalid calendar params: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}

	cal, err := c.lookupCalendar(ctx, creds, p.CalendarID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}

	return map[string]any{"calendar_id": cal.ID}, nil
}

func (c *GoogleConnector) resolveFolderDriveMeta(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var p struct {
		FolderID string `json:"folder_id"`
		ParentID string `json:"parent_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("%w: invalid drive folder params: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}

	id := driveFolderTargetID(p.FolderID, p.ParentID)
	if id == "" {
		// Omitted destination is My Drive root — not a Shared Drive member.
		return map[string]any{}, nil
	}
	return c.lookupFolderDriveMeta(ctx, creds, id)
}

func (c *GoogleConnector) resolveListScopeDriveMeta(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var p struct {
		FolderID string `json:"folder_id"`
		DriveID  string `json:"drive_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("%w: invalid drive list params: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}

	if p.FolderID != "" {
		return c.lookupFolderDriveMeta(ctx, creds, p.FolderID)
	}
	if p.DriveID != "" {
		return c.lookupVerifiedSharedDriveMeta(ctx, creds, p.DriveID)
	}
	// Neither folder nor drive: search/list across all files. Omit drive_id so
	// a Shared Drive standing approval does not match.
	return map[string]any{}, nil
}

func (c *GoogleConnector) resolveFileDriveMeta(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var p struct {
		FileID string `json:"file_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.FileID == "" {
		return nil, fmt.Errorf("%w: missing file_id", connectors.ErrConstraintMetadataUnavailable)
	}
	return c.lookupFileDriveMeta(ctx, creds, p.FileID)
}

func (c *GoogleConnector) resolveSpreadsheetDriveMeta(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var p struct {
		SpreadsheetID string `json:"spreadsheet_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.SpreadsheetID == "" {
		return nil, fmt.Errorf("%w: missing spreadsheet_id", connectors.ErrConstraintMetadataUnavailable)
	}
	return c.lookupFileDriveMeta(ctx, creds, p.SpreadsheetID)
}

func (c *GoogleConnector) lookupFolderDriveMeta(ctx context.Context, creds connectors.Credentials, id string) (map[string]any, error) {
	if !isValidDriveID(id) {
		return nil, fmt.Errorf("%w: invalid folder id", connectors.ErrConstraintMetadataUnavailable)
	}
	folder, err := c.lookupDriveFolder(ctx, creds, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}
	return constraintMetaFromDriveID(folder.DriveID), nil
}

func (c *GoogleConnector) lookupFileDriveMeta(ctx context.Context, creds connectors.Credentials, id string) (map[string]any, error) {
	if !isValidDriveID(id) {
		return nil, fmt.Errorf("%w: invalid drive file id", connectors.ErrConstraintMetadataUnavailable)
	}
	file, err := c.lookupDriveFile(ctx, creds, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}
	return constraintMetaFromDriveID(file.DriveID), nil
}

func (c *GoogleConnector) lookupVerifiedSharedDriveMeta(ctx context.Context, creds connectors.Credentials, id string) (map[string]any, error) {
	if !isValidDriveID(id) {
		return nil, fmt.Errorf("%w: invalid drive id", connectors.ErrConstraintMetadataUnavailable)
	}
	if _, err := c.lookupSharedDriveName(ctx, creds, id); err != nil {
		return nil, fmt.Errorf("%w: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}
	return constraintMetaFromDriveID(id), nil
}

func constraintMetaFromDriveID(driveID string) map[string]any {
	meta := map[string]any{}
	if driveID != "" {
		meta["drive_id"] = driveID
	}
	return meta
}

// ConstraintMetadataActionSupport reports which $meta fields are valid per action.
func (c *GoogleConnector) ConstraintMetadataActionSupport(actionType string) ([]string, bool) {
	if _, ok := googleCalendarWriteMetadataActions[actionType]; ok {
		return googleCalendarMetaConstraintFields, true
	}
	if _, ok := googleSharedDriveMetadataActions[actionType]; ok {
		return googleDriveMetaConstraintFields, true
	}
	return nil, false
}

var _ connectors.ConstraintMetadataResolver = (*GoogleConnector)(nil)
var _ connectors.ConstraintMetadataCapabilities = (*GoogleConnector)(nil)
