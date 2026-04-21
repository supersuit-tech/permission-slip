package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
	slackctx "github.com/supersuit-tech/permission-slip/connectors/slack/context"
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
		"slack.delete_message",
		"slack.remove_from_channel", "slack.remove_reaction", "slack.pin_message",
		"slack.unpin_message", "slack.archive_channel", "slack.rename_channel":
		base, err := c.resolveChannel(ctx, creds, params)
		if err != nil {
			return nil, err
		}
		switch actionType {
		case "slack.add_reaction", "slack.remove_reaction", "slack.pin_message", "slack.unpin_message",
			"slack.archive_channel", "slack.invite_to_channel", "slack.remove_from_channel":
			extra, xerr := c.resolveSlackApprovalContext(ctx, actionType, params, creds)
			if xerr != nil {
				return base, nil
			}
			return mergeResourceDetailMaps(base, extra), nil
		default:
			return base, nil
		}

	case "slack.search_messages":
		return c.resolveSearchMessagesChannel(ctx, creds, params)

	// User-based actions
	case "slack.send_dm":
		return c.resolveUser(ctx, creds, params)

	default:
		return nil, nil
	}
}

func mergeResourceDetailMaps(a, b map[string]any) map[string]any {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	out := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func (c *SlackConnector) resolveSlackApprovalContext(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var cache slackctx.MentionCache
	switch actionType {
	case "slack.add_reaction", "slack.remove_reaction":
		var p struct {
			Channel   string `json:"channel"`
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		sc, err := slackctx.BuildReactionContext(ctx, c, p.Channel, p.Timestamp, creds, nil, &cache)
		if err != nil {
			return nil, err
		}
		return slackctx.DetailsResourceMap(sc), nil
	case "slack.pin_message", "slack.unpin_message":
		var p struct {
			Channel string `json:"channel"`
			TS      string `json:"ts"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		sc, err := slackctx.BuildPinUnpinContext(ctx, c, p.Channel, p.TS, creds, nil, &cache)
		if err != nil {
			return nil, err
		}
		return slackctx.DetailsResourceMap(sc), nil
	case "slack.archive_channel":
		var p struct {
			Channel string `json:"channel"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		sc, err := slackctx.BuildArchiveContext(ctx, c, p.Channel, creds, nil, &cache)
		if err != nil {
			return nil, err
		}
		return slackctx.DetailsResourceMap(sc), nil
	case "slack.invite_to_channel":
		var p struct {
			Channel string `json:"channel"`
			Users   string `json:"users"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		sc, err := slackctx.BuildInviteContext(ctx, c, p.Channel, p.Users, creds, nil)
		if err != nil {
			return nil, err
		}
		return slackctx.DetailsResourceMap(sc), nil
	case "slack.remove_from_channel":
		var p struct {
			Channel string `json:"channel"`
			User    string `json:"user"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		sc, err := slackctx.BuildRemoveFromChannelContext(ctx, c, p.Channel, p.User, creds, nil)
		if err != nil {
			return nil, err
		}
		return slackctx.DetailsResourceMap(sc), nil
	default:
		return nil, nil
	}
}

// resolveSearchMessagesChannel resolves an optional channel ID for
// slack.search_messages. When channel is omitted, returns nil details (no error).
func (c *SlackConnector) resolveSearchMessagesChannel(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		Channel string `json:"channel"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil
	}
	if p.Channel == "" {
		return map[string]any{"channel_name": "Slack"}, nil
	}
	channelOnly, err := json.Marshal(map[string]string{"channel": p.Channel})
	if err != nil {
		return nil, err
	}
	return c.resolveChannel(ctx, creds, channelOnly)
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
