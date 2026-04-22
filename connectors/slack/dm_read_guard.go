package slack

import (
	"context"
	neturl "net/url"
	"slices"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ensureUserCanReadDMChannel verifies that the Permission Slip user (identified
// by email) is a member of the Slack DM channel before history/replies calls.
// Slack scopes conversations.history to participants; this check fails fast
// with a clear error when the token owner is not the same person as the
// Permission Slip caller (e.g. delegated tokens). For non-D channels or when no
// email is provided, this is a no-op.
func (c *SlackConnector) ensureUserCanReadDMChannel(ctx context.Context, creds connectors.Credentials, channelID, permissionSlipUserEmail string) error {
	if len(channelID) == 0 || channelID[0] != 'D' {
		return nil
	}
	email := strings.TrimSpace(permissionSlipUserEmail)
	if email == "" {
		return nil
	}

	slackUserID, err := c.lookupSlackUserIDByEmail(ctx, creds, email)
	if err != nil {
		return err
	}
	member, err := c.slackUserIsConversationMember(ctx, creds, channelID, slackUserID)
	if err != nil {
		return err
	}
	if !member {
		return &connectors.AuthError{
			Message: "Slack user for this Permission Slip account is not a member of this DM; re-authorize Slack as the user who should read this conversation, or open the correct DM channel ID.",
		}
	}
	return nil
}

type usersLookupByEmailResponse struct {
	slackResponse
	User *struct {
		ID string `json:"id"`
	} `json:"user,omitempty"`
}

func (c *SlackConnector) lookupSlackUserIDByEmail(ctx context.Context, creds connectors.Credentials, email string) (string, error) {
	token, err := c.getToken(creds)
	if err != nil {
		return "", err
	}
	q := neturl.Values{}
	q.Set("email", email)
	fullURL := c.baseURL + "/users.lookupByEmail?" + q.Encode()

	var resp usersLookupByEmailResponse
	if err := c.doGetURL(ctx, fullURL, token, &resp); err != nil {
		return "", err
	}
	if !resp.OK {
		return "", resp.asError()
	}
	if resp.User == nil || resp.User.ID == "" {
		return "", &connectors.ExternalError{StatusCode: 200, Message: "Slack users.lookupByEmail returned ok=true but no user id"}
	}
	return resp.User.ID, nil
}

type conversationsMembersRequest struct {
	Channel string `json:"channel"`
	Limit   int    `json:"limit,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
}

type conversationsMembersResponse struct {
	slackResponse
	Members []string        `json:"members,omitempty"`
	Meta    *paginationMeta `json:"response_metadata,omitempty"`
}

const maxConversationMemberPages = 50

func (c *SlackConnector) slackUserIsConversationMember(ctx context.Context, creds connectors.Credentials, channelID, slackUserID string) (bool, error) {
	cursor := ""
	for page := 0; page < maxConversationMemberPages; page++ {
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
		if slices.Contains(resp.Members, slackUserID) {
			return true, nil
		}
		if resp.Meta == nil || resp.Meta.NextCursor == "" {
			return false, nil
		}
		cursor = resp.Meta.NextCursor
	}
	return false, nil
}
