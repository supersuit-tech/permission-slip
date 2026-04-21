package context

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	// reactionSurroundingWindow is ±3 around the target message (max 7 messages).
	reactionSurroundingWindow = 7
	archiveRecentCap          = 5
)

// BuildReactionContext loads channel metadata, the target message, and up to 7
// surrounding messages (±3) from the last 24h when the target appears in that window.
func BuildReactionContext(ctx context.Context, api SlackAPI, channelID, targetTS string, creds connectors.Credentials, cache *sessionCache, mcache *MentionCache) (*SlackContext, error) {
	if err := validateChannelID(channelID); err != nil {
		return nil, err
	}
	if err := validateSlackMessageTSParam(targetTS); err != nil {
		return nil, err
	}

	meta, err := FetchChannelMeta(ctx, api, channelID, creds, cache)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			return sc, nil
		}
		return nil, err
	}

	targetSlack, err := fetchSingleChannelMessage(ctx, api, channelID, targetTS, creds)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			sc.Channel = meta
			return sc, nil
		}
		return nil, err
	}

	var tmsg *ContextMessage
	if targetSlack != nil {
		cm, err := cmForMessage(ctx, api, creds, cache, channelID, *targetSlack, mcache)
		if err != nil {
			if sc, ok := HandleRateLimit(err); ok {
				sc.Channel = meta
				return sc, nil
			}
			return nil, err
		}
		tmsg = &cm
	}

	raw, err := fetchChannelHistory24h(ctx, api, channelID, creds, 100)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			sc.Channel = meta
			sc.TargetMessage = tmsg
			return sc, nil
		}
		return nil, err
	}

	window := sliceAroundAnchor(raw, targetTS, reactionSurroundingWindow)
	recent := make([]ContextMessage, 0, len(window))
	for _, m := range window {
		if m.TS == targetTS {
			continue
		}
		cm, err := cmForMessage(ctx, api, creds, cache, channelID, m, mcache)
		if err != nil {
			if sc, ok := HandleRateLimit(err); ok {
				sc.Channel = meta
				sc.TargetMessage = tmsg
				return sc, nil
			}
			return nil, err
		}
		recent = append(recent, cm)
	}

	tsList := make([]string, 0, len(window)+1)
	tsList = append(tsList, targetTS)
	for _, m := range window {
		tsList = append(tsList, m.TS)
	}
	WithLastActivity(meta, tsList...)

	return &SlackContext{
		Channel:        meta,
		TargetMessage:  tmsg,
		RecentMessages: recent,
		ContextScope:   ScopeRecentChannel,
		ContextWindow: &ContextWindow{
			MessageCount: len(window),
			Hours:        recentWindowHours,
		},
	}, nil
}

// BuildPinUnpinContext loads channel metadata and the message being pinned or unpinned.
func BuildPinUnpinContext(ctx context.Context, api SlackAPI, channelID, targetTS string, creds connectors.Credentials, cache *sessionCache, mcache *MentionCache) (*SlackContext, error) {
	if err := validateChannelID(channelID); err != nil {
		return nil, err
	}
	if err := validateSlackMessageTSParam(targetTS); err != nil {
		return nil, err
	}

	meta, err := FetchChannelMeta(ctx, api, channelID, creds, cache)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			return sc, nil
		}
		return nil, err
	}

	targetSlack, err := fetchSingleChannelMessage(ctx, api, channelID, targetTS, creds)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			sc.Channel = meta
			return sc, nil
		}
		return nil, err
	}
	var target *ContextMessage
	if targetSlack != nil {
		cm, err := cmForMessage(ctx, api, creds, cache, channelID, *targetSlack, mcache)
		if err != nil {
			if sc, ok := HandleRateLimit(err); ok {
				sc.Channel = meta
				return sc, nil
			}
			return nil, err
		}
		target = &cm
		WithLastActivity(meta, targetSlack.TS)
	}

	return &SlackContext{
		Channel:       meta,
		TargetMessage: target,
		ContextScope:  ScopeRecentChannel,
	}, nil
}

// BuildArchiveContext loads channel metadata and up to 5 recent messages (24h window).
func BuildArchiveContext(ctx context.Context, api SlackAPI, channelID string, creds connectors.Credentials, cache *sessionCache, mcache *MentionCache) (*SlackContext, error) {
	if err := validateChannelID(channelID); err != nil {
		return nil, err
	}

	meta, err := FetchChannelMeta(ctx, api, channelID, creds, cache)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			return sc, nil
		}
		return nil, err
	}

	raw, err := fetchChannelHistory24h(ctx, api, channelID, creds, 100)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			sc.Channel = meta
			return sc, nil
		}
		return nil, err
	}
	if len(raw) > archiveRecentCap {
		raw = raw[len(raw)-archiveRecentCap:]
	}
	recent := make([]ContextMessage, 0, len(raw))
	tsList := make([]string, 0, len(raw))
	for _, m := range raw {
		tsList = append(tsList, m.TS)
		cm, err := cmForMessage(ctx, api, creds, cache, channelID, m, mcache)
		if err != nil {
			if sc, ok := HandleRateLimit(err); ok {
				sc.Channel = meta
				return sc, nil
			}
			return nil, err
		}
		recent = append(recent, cm)
	}
	WithLastActivity(meta, tsList...)

	return &SlackContext{
		Channel:        meta,
		RecentMessages: recent,
		ContextScope:   ScopeRecentChannel,
		ContextWindow: &ContextWindow{
			MessageCount: len(recent),
			Hours:        recentWindowHours,
		},
	}, nil
}

// BuildInviteContext loads channel metadata and recipient profile for invite approvals.
// When multiple user IDs are listed, the first ID is used as recipient (batch invites).
func BuildInviteContext(ctx context.Context, api SlackAPI, channelID, usersCSV string, creds connectors.Credentials, cache *sessionCache) (*SlackContext, error) {
	if err := validateChannelID(channelID); err != nil {
		return nil, err
	}
	uid := firstUserIDFromCSV(usersCSV)
	if uid == "" {
		return nil, &connectors.ValidationError{Message: "missing users for invite context"}
	}

	meta, err := FetchChannelMeta(ctx, api, channelID, creds, cache)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			return sc, nil
		}
		return nil, err
	}
	recipient, err := fetchUserRef(ctx, api, creds, uid)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			sc.Channel = meta
			return sc, nil
		}
		return nil, err
	}
	return &SlackContext{
		Channel:      meta,
		Recipient:    recipient,
		ContextScope: ScopeRecentChannel,
	}, nil
}

// BuildRemoveFromChannelContext loads channel metadata and the user being removed.
func BuildRemoveFromChannelContext(ctx context.Context, api SlackAPI, channelID, userID string, creds connectors.Credentials, cache *sessionCache) (*SlackContext, error) {
	if err := validateChannelID(channelID); err != nil {
		return nil, err
	}
	if err := validateUserIDParam(userID); err != nil {
		return nil, err
	}

	meta, err := FetchChannelMeta(ctx, api, channelID, creds, cache)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			return sc, nil
		}
		return nil, err
	}
	recipient, err := fetchUserRef(ctx, api, creds, userID)
	if err != nil {
		if sc, ok := HandleRateLimit(err); ok {
			sc.Channel = meta
			return sc, nil
		}
		return nil, err
	}
	return &SlackContext{
		Channel:      meta,
		Recipient:    recipient,
		ContextScope: ScopeRecentChannel,
	}, nil
}

func cmForMessage(ctx context.Context, api SlackAPI, creds connectors.Credentials, cache *sessionCache, channelID string, msg slackMessage, mcache *MentionCache) (ContextMessage, error) {
	sess, err := resolveSession(ctx, api, creds, cache)
	if err != nil {
		return ContextMessage{}, err
	}
	return messageToContextMessage(ctx, api, creds, sess.TeamDomain, channelID, msg, mcache)
}

func fetchSingleChannelMessage(ctx context.Context, api SlackAPI, channelID, targetTS string, creds connectors.Credentials) (*slackMessage, error) {
	body := readChannelHistoryRequest{
		Channel:   channelID,
		Latest:    targetTS,
		Limit:     1,
		Inclusive: true,
	}
	var resp messagesResponse
	if err := api.Post(ctx, "conversations.history", creds, body, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, mapSlackErr(resp.Error)
	}
	if len(resp.Messages) == 0 {
		return nil, nil
	}
	return &resp.Messages[0], nil
}

func fetchChannelHistory24h(ctx context.Context, api SlackAPI, channelID string, creds connectors.Credentials, limit int) ([]slackMessage, error) {
	cutoff := time.Now().Add(-recentWindowHours * time.Hour).UTC()
	oldestTS := fmt.Sprintf("%d.%06d", cutoff.Unix(), cutoff.Nanosecond()/1000)
	body := readChannelHistoryRequest{
		Channel: channelID,
		Limit:   limit,
		Oldest:  oldestTS,
	}
	var resp messagesResponse
	if err := api.Post(ctx, "conversations.history", creds, body, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, mapSlackErr(resp.Error)
	}
	raw := filterMessagesByAge(resp.Messages, cutoff)
	sortMessagesOldestFirst(raw)
	return raw, nil
}

func firstUserIDFromCSV(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if len(part) >= 2 && (part[0] == 'U' || part[0] == 'W') {
			return part
		}
	}
	return ""
}

func validateSlackMessageTSParam(ts string) error {
	if ts == "" {
		return &connectors.ValidationError{Message: "missing message timestamp for Slack context"}
	}
	parts := strings.SplitN(ts, ".", 3)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return &connectors.ValidationError{
			Message: "invalid message timestamp for Slack context",
		}
	}
	for _, part := range parts {
		for _, c := range part {
			if c < '0' || c > '9' {
				return &connectors.ValidationError{
					Message: "invalid message timestamp for Slack context",
				}
			}
		}
	}
	return nil
}

func validateUserIDParam(userID string) error {
	if userID == "" {
		return &connectors.ValidationError{Message: "missing user id for Slack context"}
	}
	if len(userID) < 2 || (userID[0] != 'U' && userID[0] != 'W') {
		return &connectors.ValidationError{Message: "invalid user ID for Slack context"}
	}
	return nil
}
