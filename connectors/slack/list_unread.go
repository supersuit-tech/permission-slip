package slack

import (
	"context"

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

	types := "public_channel,private_channel,mpim,im"
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
					Text: truncatePreviewText(info.Latest.Text.String()),
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
