package slack

import (
	"context"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

var _ connectors.ChannelLister = (*SlackConnector)(nil)

const slackChannelListMaxPages = 50

// ChannelListCredentialActionType returns the action type used for credential
// resolution on the agent connector channels API.
func (c *SlackConnector) ChannelListCredentialActionType() string {
	return "slack.list_channels"
}

// ListChannels returns channels for dashboard pickers, using the same listing
// rules as slack.list_channels. Paginates until Slack returns no next_cursor
// or a safety cap is hit.
func (c *SlackConnector) ListChannels(ctx context.Context, creds connectors.Credentials, userEmail string) ([]connectors.ChannelListItem, error) {
	var all []listChannelSummary
	cursor := ""
	for page := 0; page < slackChannelListMaxPages; page++ {
		params := listChannelsParams{
			Limit:           200,
			Cursor:          cursor,
			ExcludeArchived: boolPtr(true),
		}
		batch, err := c.listChannelsMerged(ctx, creds, userEmail, params)
		if err != nil {
			return nil, err
		}
		all = append(all, batch.Channels...)
		if batch.NextCursor == "" {
			break
		}
		cursor = batch.NextCursor
	}
	if cursor != "" {
		return nil, fmt.Errorf("slack channel list exceeded max pages (%d)", slackChannelListMaxPages)
	}
	out := make([]connectors.ChannelListItem, 0, len(all))
	for _, ch := range all {
		out = append(out, connectors.ChannelListItem{
			ID:           ch.ID,
			Name:         ch.Name,
			User:         ch.User,
			IsPrivate:    ch.IsPrivate,
			IsIM:         ch.IsIM,
			IsMPIM:       ch.IsMPIM,
			NumMembers:   ch.NumMembers,
			DisplayLabel: slackChannelPickerLabel(ch),
		})
	}
	return out, nil
}

func boolPtr(v bool) *bool { return &v }

// slackChannelPickerLabel matches approval-time resolution: # prefix for public
// channels, bare name for private, human-readable labels for DMs.
func slackChannelPickerLabel(ch listChannelSummary) string {
	if ch.IsIM || (len(ch.ID) > 0 && ch.ID[0] == 'D') {
		if ch.Name != "" {
			return ch.Name
		}
		if ch.User != "" {
			return "DM · " + ch.User
		}
		return "Direct message"
	}
	if ch.IsMPIM {
		if ch.Name != "" {
			return ch.Name
		}
		return "Group DM"
	}
	if ch.IsPrivate {
		if ch.Name != "" {
			return ch.Name
		}
		return ch.ID
	}
	if ch.Name != "" {
		return "#" + ch.Name
	}
	return ch.ID
}
