package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listChannelsMerged lists Slack channels using the same merge logic as
// slack.list_channels (conversations.list + users.conversations when needed).
func (c *SlackConnector) listChannelsMerged(ctx context.Context, creds connectors.Credentials, userEmail string, params listChannelsParams) (*listChannelsResult, error) {
	excludeArchived := true
	if params.ExcludeArchived != nil {
		excludeArchived = *params.ExcludeArchived
	}

	types := params.Types
	explicitTypes := types != ""
	if types == "" {
		types = "public_channel,private_channel,mpim,im"
	}

	var userChannelIDs map[string]bool
	var userPrivateMerge []listChannelEntry
	if channelTypesIncludePrivate(types) {
		if userEmail == "" {
			if explicitTypes {
				return nil, &connectors.ValidationError{
					Message: "listing private channels, group DMs, or DMs requires your Permission Slip profile to have an email address matching your Slack account",
				}
			}
			types = "public_channel"
		} else {
			slackUserID, err := c.lookupSlackUserByEmail(ctx, creds, userEmail)
			if err != nil {
				return nil, fmt.Errorf("unable to verify Slack identity: %w", err)
			}
			if slackUserID == "" {
				return nil, &connectors.ValidationError{
					Message: fmt.Sprintf("no Slack user found matching email %q — ensure your Permission Slip email matches your Slack account", userEmail),
				}
			}
			privateTypes := filterPrivateTypes(types)
			userPrivateChs, err := c.getUserPrivateConversations(ctx, creds, slackUserID, privateTypes)
			if err != nil {
				return nil, fmt.Errorf("fetching user channel memberships: %w", err)
			}
			userPrivateMerge = userPrivateChs
			userChannelIDs = make(map[string]bool, len(userPrivateChs))
			for _, ch := range userPrivateChs {
				userChannelIDs[ch.ID] = true
			}
		}
	}

	body := listChannelsRequest{
		Types:           types,
		Limit:           params.Limit,
		Cursor:          params.Cursor,
		ExcludeArchived: excludeArchived,
	}
	if body.Limit == 0 {
		body.Limit = 100
	}

	var resp listChannelsResponse
	if err := c.doPost(ctx, "conversations.list", creds, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, resp.asError()
	}

	seen := make(map[string]bool)
	result := listChannelsResult{
		Channels: make([]listChannelSummary, 0, len(resp.Channels)+len(userPrivateMerge)),
	}
	for _, ch := range resp.Channels {
		if userChannelIDs != nil && (ch.IsPrivate || strings.HasPrefix(ch.ID, "D") || strings.HasPrefix(ch.ID, "G")) {
			if !userChannelIDs[ch.ID] {
				continue
			}
		}
		seen[ch.ID] = true
		result.Channels = append(result.Channels, listChannelSummary{
			ID:         ch.ID,
			Name:       ch.Name,
			User:       ch.User,
			IsPrivate:  ch.IsPrivate,
			IsIM:       ch.IsIM,
			IsMPIM:     ch.IsMPIM,
			Topic:      ch.Topic.Value,
			Purpose:    ch.Purpose.Value,
			NumMembers: ch.NumMembers,
		})
	}
	for _, ch := range userPrivateMerge {
		if excludeArchived && ch.IsArchived {
			continue
		}
		if !listChannelEntryMatchesTypes(types, ch) {
			continue
		}
		if seen[ch.ID] {
			continue
		}
		seen[ch.ID] = true
		result.Channels = append(result.Channels, listChannelSummary{
			ID:         ch.ID,
			Name:       ch.Name,
			User:       ch.User,
			IsPrivate:  ch.IsPrivate,
			IsIM:       ch.IsIM,
			IsMPIM:     ch.IsMPIM,
			Topic:      ch.Topic.Value,
			Purpose:    ch.Purpose.Value,
			NumMembers: ch.NumMembers,
		})
	}
	if resp.Meta != nil {
		result.NextCursor = resp.Meta.NextCursor
	}

	return &result, nil
}
