package slack

import (
	"context"
	"strings"
)

// usersConversationsRequest is the Slack API request for users.conversations.
// Leave User empty on user-token (xoxp-) calls so Slack returns the token
// owner's own conversations. Passing the token owner's own ID triggers the
// "browse another user" path and can silently return empty — see #1031.
type usersConversationsRequest struct {
	User   string `json:"user,omitempty"`
	Types  string `json:"types,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type usersConversationsResponse struct {
	slackResponse
	Channels []listChannelEntry `json:"channels,omitempty"`
	Meta     *paginationMeta    `json:"response_metadata,omitempty"`
}

// maxUserConversationPages limits pagination when fetching a user's channel list.
const maxUserConversationPages = 50

// getUserPrivateConversations returns the token owner's conversations for the
// requested types via users.conversations. The Slack user token (xoxp-)
// implicitly scopes the response to the caller, so we must NOT pass a `user`
// parameter — doing so switches Slack to the admin-style "browse another
// user" path and returns empty for non-admin tokens (#1031).
func (c *SlackConnector) getUserPrivateConversations(ctx context.Context, creds connectors.Credentials, types string) ([]listChannelEntry, error) {
	var out []listChannelEntry
	cursor := ""
	for page := 0; page < maxUserConversationPages; page++ {
		body := usersConversationsRequest{
			Types:  types,
			Limit:  200,
			Cursor: cursor,
		}

		var resp usersConversationsResponse
		if err := c.doPost(ctx, "users.conversations", creds, body, &resp); err != nil {
			return nil, err
		}
		if !resp.OK {
			return nil, resp.asError()
		}

		out = append(out, resp.Channels...)

		if resp.Meta == nil || resp.Meta.NextCursor == "" {
			break
		}
		cursor = resp.Meta.NextCursor
	}

	return out, nil
}

// channelTypesIncludePrivate checks whether a comma-separated channel types
// string includes private channel types (private_channel, mpim, im).
func channelTypesIncludePrivate(types string) bool {
	for _, t := range strings.Split(types, ",") {
		t = strings.TrimSpace(t)
		if t == "private_channel" || t == "mpim" || t == "im" {
			return true
		}
	}
	return false
}

// filterPrivateTypes strips public_channel from a comma-separated types string,
// returning only private types. Used to avoid fetching unnecessary public channel
// memberships from users.conversations.
func filterPrivateTypes(types string) string {
	var private []string
	for _, t := range strings.Split(types, ",") {
		t = strings.TrimSpace(t)
		if t != "" && t != "public_channel" {
			private = append(private, t)
		}
	}
	return strings.Join(private, ",")
}
