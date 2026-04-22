package slack

import (
	"context"
)

// listChannelsMerged lists Slack channels for the user OAuth token (xoxp-).
// Primary source is conversations.list; responses are filtered with
// listChannelEntryMatchesTypes to defend against mis-honored types (#1028).
// When the request includes private types, users.conversations is merged in
// only to add conversations missing from conversations.list (same token;
// no email lookup). Slack scopes each call to the token owner.
func (c *SlackConnector) listChannelsMerged(ctx context.Context, creds connectors.Credentials, userEmail string, params listChannelsParams) (*listChannelsResult, error) {
	_ = userEmail // listing uses token scoping only; email is unused (kept for API stability).

	excludeArchived := true
	if params.ExcludeArchived != nil {
		excludeArchived = *params.ExcludeArchived
	}

	types := params.Types
	if types == "" {
		types = "public_channel,private_channel,mpim,im"
	}

	var userPrivateMerge []listChannelEntry
	if channelTypesIncludePrivate(types) {
		privateTypes := filterPrivateTypes(types)
		userPrivateChs, err := c.getUserPrivateConversations(ctx, creds, privateTypes)
		if err != nil {
			return nil, err
		}
		userPrivateMerge = userPrivateChs
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
		if !listChannelEntryMatchesTypes(types, ch) {
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
