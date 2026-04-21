package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ResolveResourceDetails fetches human-readable metadata for resources
// referenced by opaque IDs in Microsoft Graph action parameters. Errors are
// non-fatal — the caller stores the approval without details on failure.
func (c *MicrosoftConnector) ResolveResourceDetails(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	switch actionType {
	// OneDrive item (file name)
	case "microsoft.get_drive_file", "microsoft.delete_drive_file":
		return c.resolveDriveItemName(ctx, creds, params)

	// Word document title (drive item name)
	case "microsoft.get_document", "microsoft.update_document":
		return c.resolveDocumentTitle(ctx, creds, params)

	// PowerPoint presentation title
	case "microsoft.get_presentation":
		return c.resolvePresentationTitle(ctx, creds, params)

	// Excel workbook title
	case "microsoft.excel_list_worksheets", "microsoft.excel_read_range",
		"microsoft.excel_write_range", "microsoft.excel_append_rows":
		return c.resolveWorkbookTitle(ctx, creds, params)

	// Calendar display name
	case "microsoft.list_calendar_events":
		return c.resolveCalendarName(ctx, creds, params)

	// Team display name
	case "microsoft.list_channels":
		return c.resolveTeamName(ctx, creds, params)

	// Team + channel display names
	case "microsoft.send_channel_message", "microsoft.list_channel_messages":
		return c.resolveTeamAndChannel(ctx, creds, params)

	default:
		return nil, nil
	}
}

func (c *MicrosoftConnector) resolveDriveItemName(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		ItemID string `json:"item_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.ItemID == "" {
		return nil, fmt.Errorf("missing item_id")
	}
	name, err := c.fetchDriveItemName(ctx, creds, p.ItemID)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, nil
	}
	return map[string]any{"file_name": name}, nil
}

func (c *MicrosoftConnector) resolveDocumentTitle(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		ItemID string `json:"item_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.ItemID == "" {
		return nil, fmt.Errorf("missing item_id")
	}
	name, err := c.fetchDriveItemName(ctx, creds, p.ItemID)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, nil
	}
	return map[string]any{"document_title": name}, nil
}

func (c *MicrosoftConnector) resolvePresentationTitle(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		ItemID string `json:"item_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.ItemID == "" {
		return nil, fmt.Errorf("missing item_id")
	}
	name, err := c.fetchDriveItemName(ctx, creds, p.ItemID)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, nil
	}
	return map[string]any{"presentation_title": name}, nil
}

func (c *MicrosoftConnector) resolveWorkbookTitle(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		ItemID string `json:"item_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.ItemID == "" {
		return nil, fmt.Errorf("missing item_id")
	}
	name, err := c.fetchDriveItemName(ctx, creds, p.ItemID)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, nil
	}
	return map[string]any{"workbook_title": name}, nil
}

func (c *MicrosoftConnector) fetchDriveItemName(ctx context.Context, creds connectors.Credentials, itemID string) (string, error) {
	path := "/me/drive/items/" + url.PathEscape(itemID) + "?$select=name"
	var resp struct {
		Name string `json:"name"`
	}
	if err := c.doRequest(ctx, http.MethodGet, path, creds, nil, &resp); err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (c *MicrosoftConnector) resolveCalendarName(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		CalendarID string `json:"calendar_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters")
	}
	var path string
	if p.CalendarID == "" {
		path = "/me/calendar?$select=name"
	} else {
		path = "/me/calendars/" + url.PathEscape(p.CalendarID) + "?$select=name"
	}
	var resp struct {
		Name string `json:"name"`
	}
	if err := c.doRequest(ctx, http.MethodGet, path, creds, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Name == "" {
		return nil, nil
	}
	return map[string]any{"calendar_name": resp.Name}, nil
}

func (c *MicrosoftConnector) resolveTeamName(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		TeamID string `json:"team_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.TeamID == "" {
		return nil, fmt.Errorf("missing team_id")
	}
	name, err := c.fetchTeamDisplayName(ctx, creds, p.TeamID)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, nil
	}
	return map[string]any{"team_name": name}, nil
}

func (c *MicrosoftConnector) fetchTeamDisplayName(ctx context.Context, creds connectors.Credentials, teamID string) (string, error) {
	path := "/teams/" + url.PathEscape(teamID) + "?$select=displayName"
	var resp struct {
		DisplayName string `json:"displayName"`
	}
	if err := c.doRequest(ctx, http.MethodGet, path, creds, nil, &resp); err != nil {
		return "", err
	}
	return resp.DisplayName, nil
}

func (c *MicrosoftConnector) resolveTeamAndChannel(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		TeamID    string `json:"team_id"`
		ChannelID string `json:"channel_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.TeamID == "" || p.ChannelID == "" {
		return nil, fmt.Errorf("missing team_id or channel_id")
	}
	teamName, err := c.fetchTeamDisplayName(ctx, creds, p.TeamID)
	if err != nil {
		return nil, err
	}
	chPath := "/teams/" + url.PathEscape(p.TeamID) + "/channels/" + url.PathEscape(p.ChannelID) + "?$select=displayName"
	var chResp struct {
		DisplayName string `json:"displayName"`
	}
	if err := c.doRequest(ctx, http.MethodGet, chPath, creds, nil, &chResp); err != nil {
		return nil, err
	}
	details := map[string]any{}
	if teamName != "" {
		details["team_name"] = teamName
	}
	if chResp.DisplayName != "" {
		details["channel_name"] = chResp.DisplayName
	}
	if len(details) == 0 {
		return nil, nil
	}
	return details, nil
}

// Compile-time check that MicrosoftConnector implements ResourceDetailResolver.
var _ connectors.ResourceDetailResolver = (*MicrosoftConnector)(nil)
