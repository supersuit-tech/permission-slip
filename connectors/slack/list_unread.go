package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listUnreadAction implements slack.list_unread.
type listUnreadAction struct {
	conn *SlackConnector
}

type listUnreadParams struct{}

func (p *listUnreadParams) validate() error { return nil }

type latestMessagePreview struct {
	Text string `json:"text,omitempty"`
	User string `json:"user,omitempty"`
	TS   string `json:"ts,omitempty"`
}

type unreadChannelEntry struct {
	ChannelID            string                `json:"channel_id"`
	ChannelName          string                `json:"channel_name,omitempty"`
	ChannelType          string                `json:"channel_type"`
	UnreadCount          int                   `json:"unread_count"`
	LastReadTS           string                `json:"last_read_ts,omitempty"`
	LatestMessagePreview *latestMessagePreview `json:"latest_message_preview,omitempty"`
}

type listUnreadResult struct {
	UnreadChannels []unreadChannelEntry `json:"unread_channels"`
}

func (a *listUnreadAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listUnreadParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	if req.UserEmail == "" {
		return nil, &connectors.ValidationError{
			Message: "listing unread conversations requires your Permission Slip profile to have an email address matching your Slack account",
		}
	}

	slackUserID, err := a.conn.lookupSlackUserByEmail(ctx, req.Credentials, req.UserEmail)
	if err != nil {
		return nil, err
	}
	if slackUserID == "" {
		return nil, &connectors.ValidationError{
			Message: "no Slack user found matching your email — ensure your Permission Slip email matches your Slack account",
		}
	}

	types := "public_channel,private_channel,mpim,im"

	// Verify the user token has the scopes needed to enumerate private
	// conversations. Slack's users.conversations silently returns an empty
	// channel list when the token lacks im:read / mpim:read / groups:read
	// instead of returning missing_scope — same failure mode as #1033.
	privateTypes := filterPrivateTypes(types)
	if required := requiredPrivateTypeScopes(privateTypes); len(required) > 0 {
		granted, scopeErr := a.conn.probeGrantedScopes(ctx, req.Credentials)
		if scopeErr != nil {
			return nil, fmt.Errorf("verifying Slack OAuth scopes: %w", scopeErr)
		}
		if missing := missingScopes(granted, required...); len(missing) > 0 {
			return nil, &connectors.AuthError{
				Message: fmt.Sprintf("Slack token is missing OAuth scope(s) %s required to list unread conversations — re-authorize the Slack connection to grant them", strings.Join(missing, ", ")),
			}
		}
	}
	var entries []unreadChannelEntry
	cursor := ""
	for page := 0; page < maxUserConversationPages; page++ {
		// Omit User: the xoxp- user token implicitly scopes users.conversations
		// to the token owner. Passing the owner's own ID triggers the admin
		// "browse another user" path and returns empty (#1031).
		body := usersConversationsRequest{
			Types:  types,
			Limit:  200,
			Cursor: cursor,
		}
		var resp usersConversationsResponse
		if err := a.conn.doPost(ctx, "users.conversations", req.Credentials, body, &resp); err != nil {
			return nil, err
		}
		if !resp.OK {
			return nil, resp.asError()
		}

		for _, ch := range resp.Channels {
			info, err := a.conn.fetchConversationInfo(ctx, req.Credentials, ch.ID)
			if err != nil {
				return nil, err
			}
			if info.UnreadCountDisplay <= 0 {
				continue
			}
			var previewPtr *latestMessagePreview
			if info.Latest != nil {
				p := latestMessagePreview{
					Text: truncatePreviewText(info.Latest.Text),
					User: info.Latest.User,
					TS:   info.Latest.TS,
				}
				if info.Latest.BotID != "" && p.User == "" {
					p.User = info.Latest.BotID
				}
				previewPtr = &p
			}
			name := channelDisplayName(info)
			if name == "" {
				name = channelDisplayNameFromListEntry(ch)
			}
			entries = append(entries, unreadChannelEntry{
				ChannelID:            info.ID,
				ChannelName:          name,
				ChannelType:          channelTypeLabel(info),
				UnreadCount:          info.UnreadCountDisplay,
				LastReadTS:           info.LastRead,
				LatestMessagePreview: previewPtr,
			})
		}

		if resp.Meta == nil || resp.Meta.NextCursor == "" {
			break
		}
		cursor = resp.Meta.NextCursor
	}

	return connectors.JSONResult(listUnreadResult{UnreadChannels: entries})
}

func channelDisplayNameFromListEntry(ch listChannelEntry) string {
	if ch.Name != "" {
		return ch.Name
	}
	if ch.IsIM && ch.User != "" {
		return ch.User
	}
	return ""
}
