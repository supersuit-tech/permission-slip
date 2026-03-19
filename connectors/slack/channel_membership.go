package slack

import (
	"context"
	"fmt"
	neturl "net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
		return "", mapSlackError(resp.Error)
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
	Members []string       `json:"members"`
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
			return false, mapSlackError(resp.Error)
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
	// Public channels (C-prefixed) are accessible to everyone in the workspace.
	// We still verify for private channels that happen to start with C by
	// checking membership, but the common case for C-channels is public.
	// G-prefixed = private channel or group DM, D-prefixed = DM.
	if len(channelID) > 0 && channelID[0] == 'C' {
		// For C-channels, check if it's actually private via conversations.info.
		// Most C-channels are public, so skip the membership check for public ones.
		isPrivate, err := c.isChannelPrivate(ctx, creds, channelID)
		if err != nil {
			// If we can't determine the channel type, deny access as a safe default.
			return &connectors.ValidationError{
				Message: fmt.Sprintf("unable to verify access to channel %s: %v", channelID, err),
			}
		}
		if !isPrivate {
			return nil // public channel — access allowed
		}
	}

	// For private channels (G), DMs (D), and private C-channels: require
	// email-based membership verification.
	chType := describeChannelType(channelID)
	if userEmail == "" {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("this action accesses a %s (%s), but your Permission Slip profile has no email address — add an email that matches your Slack account to proceed", chType, channelID),
		}
	}

	slackUserID, err := c.lookupSlackUserByEmail(ctx, creds, userEmail)
	if err != nil {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("unable to verify Slack identity for %s access: %v", chType, err),
		}
	}
	if slackUserID == "" {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("no Slack user found matching email %q — ensure your Permission Slip email matches your Slack account to access this %s", userEmail, chType),
		}
	}

	isMember, err := c.isUserInChannel(ctx, creds, channelID, slackUserID)
	if err != nil {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("unable to verify membership in %s %s: %v", chType, channelID, err),
		}
	}
	if !isMember {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("access denied: your Slack account is not a member of %s %s", chType, channelID),
		}
	}

	return nil
}

// isChannelPrivate checks whether a C-prefixed channel is private by calling
// conversations.info and inspecting the is_private field.
func (c *SlackConnector) isChannelPrivate(ctx context.Context, creds connectors.Credentials, channelID string) (bool, error) {
	body := struct {
		Channel string `json:"channel"`
	}{Channel: channelID}

	var resp struct {
		slackResponse
		Channel struct {
			IsPrivate bool `json:"is_private"`
		} `json:"channel"`
	}
	if err := c.doPost(ctx, "conversations.info", creds, body, &resp); err != nil {
		return false, err
	}
	if !resp.OK {
		return false, mapSlackError(resp.Error)
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
