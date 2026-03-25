package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

var _ connectors.UserLister = (*SlackConnector)(nil)

const slackUserListMaxPages = 50

// UserListCredentialActionType returns the action type used for credential
// resolution on the agent connector users API.
func (c *SlackConnector) UserListCredentialActionType() string {
	return "slack.list_users"
}

// ListUsers returns workspace users for dashboard pickers, paginating until
// Slack returns no next_cursor or a safety cap is hit.
func (c *SlackConnector) ListUsers(ctx context.Context, creds connectors.Credentials) ([]connectors.UserListItem, error) {
	var all []listUserSummary
	seen := make(map[string]bool)
	cursor := ""
	for page := 0; page < slackUserListMaxPages; page++ {
		body := listUsersRequest{
			Limit:  200,
			Cursor: cursor,
		}

		var resp listUsersResponse
		if err := c.doPost(ctx, "users.list", creds, body, &resp); err != nil {
			return nil, err
		}
		if !resp.OK {
			return nil, resp.asError()
		}

		for _, u := range resp.Members {
			if seen[u.ID] {
				continue
			}
			// Skip deleted users and bots from picker.
			if u.Deleted || u.IsBot {
				continue
			}
			seen[u.ID] = true
			all = append(all, listUserSummary{
				ID:          u.ID,
				Name:        u.Name,
				RealName:    u.RealName,
				DisplayName: u.Profile.DisplayName,
				Email:       u.Profile.Email,
				IsBot:       u.IsBot,
				IsAdmin:     u.IsAdmin,
				Deleted:     u.Deleted,
			})
		}

		if resp.Meta == nil || resp.Meta.NextCursor == "" {
			break
		}
		cursor = resp.Meta.NextCursor
	}

	out := make([]connectors.UserListItem, 0, len(all))
	for _, u := range all {
		out = append(out, connectors.UserListItem{
			ID:           u.ID,
			Name:         u.Name,
			RealName:     u.RealName,
			DisplayName:  u.DisplayName,
			Email:        u.Email,
			IsBot:        u.IsBot,
			DisplayLabel: slackUserPickerLabel(u),
		})
	}
	return out, nil
}

// slackUserPickerLabel builds a human-readable label for the user picker.
func slackUserPickerLabel(u listUserSummary) string {
	name := u.RealName
	if name == "" {
		name = u.DisplayName
	}
	if name == "" {
		name = u.Name
	}
	if name == "" {
		return u.ID
	}
	if u.Email != "" {
		return name + " (" + u.Email + ")"
	}
	return name
}
