package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listChannelsAction implements connectors.Action for slack.list_channels.
// It lists channels visible to the bot via POST /conversations.list.
type listChannelsAction struct {
	conn *SlackConnector
}

// listChannelsParams defines the user-facing parameter schema.
type listChannelsParams struct {
	// Types filters by channel type. Comma-separated list of:
	// public_channel, private_channel, mpim, im.
	// Defaults to all types: public_channel,private_channel,mpim,im.
	// Falls back to public_channel if UserEmail is not set (required for private types).
	Types string `json:"types,omitempty"`
	// Limit is the max number of channels to return (1-1000, default 100).
	Limit int `json:"limit,omitempty"`
	// Cursor is a pagination cursor from a previous response.
	Cursor string `json:"cursor,omitempty"`
	// ExcludeArchived filters out archived channels. Defaults to true.
	ExcludeArchived *bool `json:"exclude_archived,omitempty"`
}

func (p *listChannelsParams) validate() error {
	return validateLimit(p.Limit)
}

// listChannelsRequest is the Slack API request body for conversations.list.
type listChannelsRequest struct {
	Types           string `json:"types,omitempty"`
	Limit           int    `json:"limit,omitempty"`
	Cursor          string `json:"cursor,omitempty"`
	ExcludeArchived bool   `json:"exclude_archived"`
}

type listChannelsResponse struct {
	slackResponse
	Channels []listChannelEntry `json:"channels,omitempty"`
	Meta     *paginationMeta    `json:"response_metadata,omitempty"`
}

// listChannelEntry maps the Slack API response for a single channel from
// conversations.list. IM channels (DMs) omit Name and instead populate User
// with the other participant's Slack user ID.
type listChannelEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	User       string `json:"user,omitempty"`
	IsPrivate  bool   `json:"is_private"`
	IsArchived bool   `json:"is_archived"`
	NumMembers int    `json:"num_members"`
	Topic      struct {
		Value string `json:"value"`
	} `json:"topic"`
	Purpose struct {
		Value string `json:"value"`
	} `json:"purpose"`
}

// listChannelsResult is the action output.
type listChannelsResult struct {
	Channels   []listChannelSummary `json:"channels"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

// listChannelSummary is the user-facing output for a single channel. For IM
// channels, Name is empty and User contains the other participant's Slack user
// ID. Both fields use omitempty so the JSON output only includes relevant fields.
type listChannelSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	User       string `json:"user,omitempty"`
	IsPrivate  bool   `json:"is_private"`
	Topic      string `json:"topic,omitempty"`
	Purpose    string `json:"purpose,omitempty"`
	NumMembers int    `json:"num_members"`
}

// Execute lists Slack channels visible to the bot.
func (a *listChannelsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listChannelsParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	excludeArchived := true
	if params.ExcludeArchived != nil {
		excludeArchived = *params.ExcludeArchived
	}

	types := params.Types
	explicitTypes := types != ""
	if types == "" {
		types = "public_channel,private_channel,mpim,im"
	}

	// When listing private channels, group DMs, or DMs, resolve the user's
	// Slack identity and pre-fetch their channel memberships so we can filter
	// results efficiently. Uses users.conversations (one paginated call) instead
	// of per-channel isUserInChannel to avoid N+1 API calls.
	//
	// If the caller didn't explicitly request private types (using the default)
	// and has no email set, gracefully fall back to public_channel only instead
	// of returning an error. This preserves backward compatibility for callers
	// that previously relied on the old public_channel-only default.
	var userChannelIDs map[string]bool
	if channelTypesIncludePrivate(types) {
		if req.UserEmail == "" {
			if explicitTypes {
				return nil, &connectors.ValidationError{
					Message: "listing private channels, group DMs, or DMs requires your Permission Slip profile to have an email address matching your Slack account",
				}
			}
			// Graceful fallback: caller used the default, so fall back to
			// public channels only rather than breaking existing integrations.
			types = "public_channel"
		} else {
			slackUserID, err := a.conn.lookupSlackUserByEmail(ctx, req.Credentials, req.UserEmail)
			if err != nil {
				return nil, fmt.Errorf("unable to verify Slack identity: %w", err)
			}
			if slackUserID == "" {
				return nil, &connectors.ValidationError{
					Message: fmt.Sprintf("no Slack user found matching email %q — ensure your Permission Slip email matches your Slack account", req.UserEmail),
				}
			}
			// Only fetch memberships for private channel types — public channels
			// are never filtered, so including them wastes API quota.
			privateTypes := filterPrivateTypes(types)
			userChannelIDs, err = a.conn.getUserChannelIDs(ctx, req.Credentials, slackUserID, privateTypes)
			if err != nil {
				return nil, fmt.Errorf("fetching user channel memberships: %w", err)
			}
		}
	}

	body := listChannelsRequest{
		Types:           types,
		Limit:           params.Limit,
		Cursor:          params.Cursor,
		ExcludeArchived: excludeArchived,
	}
	if body.Limit == 0 {
		body.Limit = 100
	}

	var resp listChannelsResponse
	if err := a.conn.doPost(ctx, "conversations.list", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	result := listChannelsResult{
		Channels: make([]listChannelSummary, 0, len(resp.Channels)),
	}
	for _, ch := range resp.Channels {
		// For private channel types, filter to only channels the user is a member of.
		// Uses the pre-fetched userChannelIDs set for O(1) lookups instead of per-channel API calls.
		if userChannelIDs != nil && (ch.IsPrivate || strings.HasPrefix(ch.ID, "D") || strings.HasPrefix(ch.ID, "G")) {
			if !userChannelIDs[ch.ID] {
				continue
			}
		}
		result.Channels = append(result.Channels, listChannelSummary{
			ID:         ch.ID,
			Name:       ch.Name,
			User:       ch.User,
			IsPrivate:  ch.IsPrivate,
			Topic:      ch.Topic.Value,
			Purpose:    ch.Purpose.Value,
			NumMembers: ch.NumMembers,
		})
	}
	if resp.Meta != nil {
		result.NextCursor = resp.Meta.NextCursor
	}

	return connectors.JSONResult(result)
}
