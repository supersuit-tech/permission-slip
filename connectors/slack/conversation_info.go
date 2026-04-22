package slack

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// conversationInfoChannel is the subset of conversations.channel used for
// unread state and display (conversations.info).
type conversationInfoChannel struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	User               string        `json:"user"`
	IsPrivate          bool          `json:"is_private"`
	IsIM               bool          `json:"is_im"`
	IsMPIM             bool          `json:"is_mpim"`
	LastRead           string        `json:"last_read"`
	UnreadCountDisplay int           `json:"unread_count_display"`
	Latest             *slackMessage `json:"latest"`
}

type conversationInfoResponse struct {
	slackResponse
	Channel conversationInfoChannel `json:"channel"`
}

// fetchConversationInfo loads a single channel via GET conversations.info.
func (c *SlackConnector) fetchConversationInfo(ctx context.Context, creds connectors.Credentials, channelID string) (*conversationInfoChannel, error) {
	var resp conversationInfoResponse
	if err := c.doGet(ctx, "conversations.info", creds, map[string]string{"channel": channelID}, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, resp.asError()
	}
	return &resp.Channel, nil
}

const maxLatestPreviewRunes = 200

func channelDisplayName(ch *conversationInfoChannel) string {
	if ch == nil {
		return ""
	}
	if ch.Name != "" {
		return ch.Name
	}
	if ch.IsIM && ch.User != "" {
		return ch.User
	}
	return ""
}

func channelTypeLabel(ch *conversationInfoChannel) string {
	if ch == nil {
		return "unknown"
	}
	switch {
	case ch.IsIM:
		return "im"
	case ch.IsMPIM:
		return "mpim"
	case ch.IsPrivate:
		return "private_channel"
	default:
		return "public_channel"
	}
}

func truncatePreviewText(s string) string {
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxLatestPreviewRunes {
		return strings.TrimSpace(s)
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:maxLatestPreviewRunes])) + "…"
}
