package slack

import (
	"context"
	"strconv"

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

	// Track channel IDs processed to avoid duplicates across pages.
	processed := make(map[string]bool)

	// Fetch all private channels once via POST (GET omits private channels for
	// some workspaces). POST ignores pagination limit but we only need all private
	// channels once.
	var privateChannels []listChannelEntry
	privateTypes := filterPrivateTypes(types)
	if privateTypes != "" {
		privateBody := usersConversationsRequest{
			Types: privateTypes,
			Limit: 200,
		}
		var privateResp usersConversationsResponse
		if err := a.conn.doPost(ctx, "users.conversations", req.Credentials, privateBody, &privateResp); err == nil && privateResp.OK {
			privateChannels = privateResp.Channels
		}
	}

	for page := 0; page < maxUserConversationPages; page++ {
		// Use GET for pagination: POST ignores limit/cursor for users.conversations.
		paramsMap := map[string]string{
			"types": types,
			"limit": strconv.Itoa(200),
		}
		if cursor != "" {
			paramsMap["cursor"] = cursor
		}
		var resp usersConversationsResponse
		if err := a.conn.doGet(ctx, "users.conversations", req.Credentials, paramsMap, &resp); err != nil {
			return nil, err
		}
		if !resp.OK {
			return nil, resp.asError()
		}

		for _, ch := range resp.Channels {
			if processed[ch.ID] {
				continue
			}
			processed[ch.ID] = true

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

	// Process private channels that were not already covered by GET pages.
	for _, ch := range privateChannels {
		if processed[ch.ID] {
			continue
		}
		processed[ch.ID] = true

		info, err := a.conn.fetchConversationInfo(ctx, req.Credentials, ch.ID)
		if err != nil {
			return nil, err
		}
		if info.UnreadCountDisplay <= 0 {
			continue
		}
		var previewPtr *latestMessagePreview
		if info.Latest != nil {
			pv := latestMessagePreview{
				Text: truncatePreviewText(info.Latest.Text.String()),
				User: info.Latest.User,
				TS:   info.Latest.TS,
			}
			if info.Latest.BotID != "" && pv.User == "" {
				pv.User = info.Latest.BotID
			}
			previewPtr = &pv
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
