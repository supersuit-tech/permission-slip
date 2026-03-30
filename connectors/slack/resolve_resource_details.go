package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ResolveResourceDetails fetches human-readable metadata for resources
// referenced by opaque IDs in Slack action parameters. For channel-based
// actions it resolves channel IDs to names; for user-based actions it
// resolves user IDs to display names. Errors are non-fatal — the caller
// stores the approval without details on failure.
func (c *SlackConnector) ResolveResourceDetails(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	switch actionType {
	// Channel-based actions
	case "slack.send_message", "slack.read_channel_messages", "slack.read_thread",
		"slack.schedule_message", "slack.set_topic", "slack.invite_to_channel",
		"slack.upload_file", "slack.add_reaction", "slack.update_message",
		"slack.delete_message":
		return c.resolveChannel(ctx, creds, params)

	// User-based actions
	case "slack.send_dm":
		return c.resolveUser(ctx, creds, params)

	default:
		return nil, nil
	}
}

// resolveChannel calls conversations.info to fetch the channel name for a
// channel ID parameter.
func (c *SlackConnector) resolveChannel(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		Channel string `json:"channel"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.Channel == "" {
		return nil, fmt.Errorf("missing channel")
	}

	// Only resolve IDs (starting with C, G, or D) — not already-resolved names.
	if len(p.Channel) < 2 || (p.Channel[0] != 'C' && p.Channel[0] != 'G' && p.Channel[0] != 'D') {
		return nil, nil
	}

	type channelInfo struct {
		Name      string `json:"name"`
		IsPrivate bool   `json:"is_private"`
	}
	var resp struct {
		slackResponse
		Channel channelInfo `json:"channel"`
	}

	// conversations.info only accepts GET / form-encoded POST, not JSON body.
	if err := c.doGet(ctx, "conversations.info", creds, map[string]string{"channel": p.Channel}, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("conversations.info: %s", resp.Error)
	}

	name := resp.Channel.Name
	if name == "" {
		return nil, nil
	}

	// Prefix with # for public channels, leave private as-is.
	if !resp.Channel.IsPrivate {
		name = "#" + name
	}

	return map[string]any{"channel_name": name}, nil
}

// resolveUser calls users.info to fetch a display name for a user ID parameter.
func (c *SlackConnector) resolveUser(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.UserID == "" {
		return nil, fmt.Errorf("missing user_id")
	}

	// Only resolve IDs (starting with U or W).
	if len(p.UserID) < 2 || (p.UserID[0] != 'U' && p.UserID[0] != 'W') {
		return nil, nil
	}

	type userProfile struct {
		DisplayName string `json:"display_name"`
		RealName    string `json:"real_name"`
	}
	type userInfo struct {
		RealName string      `json:"real_name"`
		Name     string      `json:"name"`
		Profile  userProfile `json:"profile"`
	}
	var resp struct {
		slackResponse
		User userInfo `json:"user"`
	}

	// users.info only accepts GET / form-encoded POST, not JSON body.
	if err := c.doGet(ctx, "users.info", creds, map[string]string{"user": p.UserID}, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("users.info: %s", resp.Error)
	}

	// Prefer display_name > real_name (profile) > real_name (top-level) > username.
	displayName := strings.TrimSpace(resp.User.Profile.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(resp.User.Profile.RealName)
	}
	if displayName == "" {
		displayName = strings.TrimSpace(resp.User.RealName)
	}
	if displayName == "" {
		displayName = resp.User.Name
	}
	if displayName == "" {
		return nil, nil
	}

	return map[string]any{"user_name": displayName}, nil
}
