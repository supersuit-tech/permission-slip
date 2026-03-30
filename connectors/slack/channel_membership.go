package slack

import (
	"context"
	"fmt"
	neturl "net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type userLookupByEmailResponse struct {
	slackResponse
	User struct {
		ID string `json:"id"`
	} `json:"user"`
}

// lookupSlackUserByEmail resolves a Permission Slip user's email to their
// Slack user ID via the users.lookupByEmail API. Returns ("", nil) if the
// email doesn't match any Slack user.
func (c *SlackConnector) lookupSlackUserByEmail(ctx context.Context, creds connectors.Credentials, email string) (string, error) {
	if email == "" {
		return "", nil
	}

	// users.lookupByEmail requires query params (GET-style), not JSON body.
	token, err := c.getToken(creds)
	if err != nil {
		return "", err
	}

	url := c.baseURL + "/users.lookupByEmail?" + neturl.Values{"email": {email}}.Encode()
	var resp userLookupByEmailResponse
	if err := c.doGetURL(ctx, url, token, &resp); err != nil {
		return "", err
	}

	if !resp.OK {
		// users_not_found means the email doesn't match a Slack user.
		if resp.Error == "users_not_found" {
			return "", nil
		}
		return "", resp.asError()
	}

	return resp.User.ID, nil
}

// conversationsMembersRequest is the Slack API request for conversations.members.
type conversationsMembersRequest struct {
	Channel string `json:"channel"`
	Limit   int    `json:"limit,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
}

type conversationsMembersResponse struct {
	slackResponse
	Members []string        `json:"members"`
	Meta    *paginationMeta `json:"response_metadata,omitempty"`
}

// maxMembershipPages limits the number of pagination requests when checking
// channel membership. At 200 members per page, this covers channels with up
// to 10,000 members. Larger channels are denied access as a safety measure
// rather than making unbounded API calls.
const maxMembershipPages = 50

// isUserInChannel checks whether the given Slack user ID is a member of the
// specified channel. Paginates through the member list up to maxMembershipPages.
func (c *SlackConnector) isUserInChannel(ctx context.Context, creds connectors.Credentials, channelID, slackUserID string) (bool, error) {
	cursor := ""
	for page := 0; page < maxMembershipPages; page++ {
		body := conversationsMembersRequest{
			Channel: channelID,
			Limit:   200,
			Cursor:  cursor,
		}

		var resp conversationsMembersResponse
		if err := c.doPost(ctx, "conversations.members", creds, body, &resp); err != nil {
			return false, err
		}
		if !resp.OK {
			return false, resp.asError()
		}

		for _, member := range resp.Members {
			if member == slackUserID {
				return true, nil
			}
		}

		if resp.Meta == nil || resp.Meta.NextCursor == "" {
			break
		}
		cursor = resp.Meta.NextCursor
	}

	return false, nil
}

// verifyChannelAccess checks that the Permission Slip user (identified by
// email) has access to the given Slack channel. For public channels, access
// is always granted. For private channels, group DMs, and DMs, the user must
// be a member.
//
// Returns a user-friendly error if access is denied, or nil if access is
// allowed. If the user's email is empty (profile not set), access is denied
// for non-public channels as a safe default.
func (c *SlackConnector) verifyChannelAccess(ctx context.Context, creds connectors.Credentials, channelID, userEmail string) error {
	// Fast path: public C-channels are accessible to everyone without
	// needing the user's identity.
	var knownPrivate bool
	if len(channelID) > 0 && channelID[0] == 'C' {
		isPrivate, err := c.isChannelPrivate(ctx, creds, channelID)
		if err != nil {
			return fmt.Errorf("unable to verify access to channel %s: %w", channelID, err)
		}
		if !isPrivate {
			return nil
		}
		knownPrivate = true
	}

	// Private channels, DMs, and group DMs require email-based identity
	// verification. Resolve the user's Slack identity, then check membership.
	chType := describeChannelType(channelID)
	slackUserID, err := c.resolveSlackUserID(ctx, creds, userEmail, chType, channelID)
	if err != nil {
		return err
	}

	// If we already determined the channel is private via isChannelPrivate,
	// call isUserInChannel directly to avoid a redundant conversations.info call
	// inside hasChannelAccess.
	var allowed bool
	var memberErr error
	if knownPrivate {
		allowed, memberErr = c.isUserInChannel(ctx, creds, channelID, slackUserID)
	} else {
		allowed, memberErr = c.hasChannelAccess(ctx, creds, channelID, slackUserID)
	}
	if memberErr != nil {
		return fmt.Errorf("unable to verify membership in %s %s: %w", chType, channelID, memberErr)
	}
	if !allowed {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("access denied: your Slack account is not a member of %s %s", chType, channelID),
		}
	}

	return nil
}

// resolveSlackUserID maps a Permission Slip email to a Slack user ID, returning
// a user-friendly ValidationError if the email is missing or doesn't match.
func (c *SlackConnector) resolveSlackUserID(ctx context.Context, creds connectors.Credentials, userEmail, chType, channelID string) (string, error) {
	if userEmail == "" {
		return "", &connectors.ValidationError{
			Message: fmt.Sprintf("this action accesses a %s (%s), but your Permission Slip profile has no email address — add an email that matches your Slack account to proceed", chType, channelID),
		}
	}

	slackUserID, err := c.lookupSlackUserByEmail(ctx, creds, userEmail)
	if err != nil {
		return "", fmt.Errorf("unable to verify Slack identity for %s access: %w", chType, err)
	}
	if slackUserID == "" {
		return "", &connectors.ValidationError{
			Message: fmt.Sprintf("no Slack user found matching email %q — ensure your Permission Slip email matches your Slack account to access this %s", userEmail, chType),
		}
	}

	return slackUserID, nil
}

// hasChannelAccess checks whether a Slack user has access to a channel.
// Public C-channels return (true, nil). Private channels, DMs, and group DMs
// check membership via conversations.members. Used by both verifyChannelAccess
// (for single-channel guards) and search result filtering (for batch checks).
func (c *SlackConnector) hasChannelAccess(ctx context.Context, creds connectors.Credentials, channelID, slackUserID string) (bool, error) {
	if len(channelID) > 0 && channelID[0] == 'C' {
		isPrivate, err := c.isChannelPrivate(ctx, creds, channelID)
		if err != nil {
			return false, fmt.Errorf("checking channel %s visibility: %w", channelID, err)
		}
		if !isPrivate {
			return true, nil
		}
	}
	return c.isUserInChannel(ctx, creds, channelID, slackUserID)
}

// isChannelPrivate checks whether a C-prefixed channel is private by calling
// conversations.info and inspecting the is_private field.
func (c *SlackConnector) isChannelPrivate(ctx context.Context, creds connectors.Credentials, channelID string) (bool, error) {
	var resp struct {
		slackResponse
		Channel struct {
			IsPrivate bool `json:"is_private"`
		} `json:"channel"`
	}
	// conversations.info only accepts GET / form-encoded POST, not JSON body.
	// Using doGet avoids the "invalid_arguments" error with user tokens.
	if err := c.doGet(ctx, "conversations.info", creds, map[string]string{"channel": channelID}, &resp); err != nil {
		return false, err
	}
	if !resp.OK {
		return false, resp.asError()
	}

	return resp.Channel.IsPrivate, nil
}

// describeChannelType returns a human-readable description of a Slack channel
// type based on its ID prefix (e.g., "DM", "group DM", "private channel").
func describeChannelType(channelID string) string {
	if len(channelID) == 0 {
		return "channel"
	}
	switch channelID[0] {
	case 'D':
		return "DM"
	case 'G':
		return "group DM or private channel"
	case 'C':
		return "private channel"
	default:
		return "channel"
	}
}

// usersConversationsRequest is the Slack API request for users.conversations.
type usersConversationsRequest struct {
	User   string `json:"user"`
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

// getUserPrivateConversations returns conversation objects for the given Slack user
// and types via users.conversations. This uses the user token (xoxp-), which only
// returns conversations the token owner belongs to. A bot token with the `user`
// parameter would require admin.conversations:read scope — we intentionally
// use the user token to avoid that requirement.
func (c *SlackConnector) getUserPrivateConversations(ctx context.Context, creds connectors.Credentials, slackUserID, types string) ([]listChannelEntry, error) {
	var out []listChannelEntry
	cursor := ""
	for page := 0; page < maxUserConversationPages; page++ {
		body := usersConversationsRequest{
			User:   slackUserID,
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
