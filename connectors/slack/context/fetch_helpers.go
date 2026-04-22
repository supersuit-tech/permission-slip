package context

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	recentWindowHours = 24
	recentCapChannel  = 10
	recentCapDM       = 10
	threadCap         = 20
)

// RecentMessagesOpts configures fetchRecentMessages.
type RecentMessagesOpts struct {
	// AnchorTS, when set, biases the window around this message (in_response_to_ts).
	AnchorTS string
}

func messageToContextMessage(ctx context.Context, api SlackAPI, creds connectors.Credentials, teamDomain, channelID string, msg slackMessage, mcache *MentionCache) (ContextMessage, error) {
	text := msg.Text.String()
	resolved, err := ResolveMentions(ctx, text, api, creds, mcache)
	if err != nil {
		return ContextMessage{}, err
	}
	body, trunc := TruncateBody(resolved)
	var uref *UserRef
	if msg.User != "" {
		uref, err = fetchUserRef(ctx, api, creds, msg.User)
		if err != nil {
			return ContextMessage{}, err
		}
	}
	isBot := msg.BotID != ""
	cm := ContextMessage{
		User:      uref,
		Text:      body,
		TS:        msg.TS,
		Permalink: BuildPermalink(teamDomain, channelID, msg.TS),
		IsBot:     isBot,
		Truncated: trunc,
		Files:     ExtractFiles(msg),
	}
	return cm, nil
}

func sortMessagesOldestFirst(msgs []slackMessage) {
	sort.SliceStable(msgs, func(i, j int) bool {
		si, ni, okI := parseSlackTSNs(msgs[i].TS)
		sj, nj, okJ := parseSlackTSNs(msgs[j].TS)
		if !okI || !okJ {
			return msgs[i].TS < msgs[j].TS
		}
		if si != sj {
			return si < sj
		}
		return ni < nj
	})
}

// FetchRecentMessages returns up to 10 messages from the last 24h in a channel,
// oldest first. Optional AnchorTS narrows the window around that message.
func FetchRecentMessages(ctx context.Context, api SlackAPI, channelID string, creds connectors.Credentials, cache *sessionCache, opts RecentMessagesOpts, mcache *MentionCache) ([]ContextMessage, error) {
	if err := validateChannelID(channelID); err != nil {
		return nil, err
	}
	sess, err := resolveSession(ctx, api, creds, cache)
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().Add(-recentWindowHours * time.Hour).UTC()
	oldestTS := fmt.Sprintf("%d.%06d", cutoff.Unix(), cutoff.Nanosecond()/1000)

	body := readChannelHistoryRequest{
		Channel: channelID,
		Limit:   100,
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
	raw = sliceAroundAnchor(raw, opts.AnchorTS, recentCapChannel)
	if len(raw) > recentCapChannel {
		raw = raw[len(raw)-recentCapChannel:]
	}
	out := make([]ContextMessage, 0, len(raw))
	for _, m := range raw {
		cm, err := messageToContextMessage(ctx, api, creds, sess.TeamDomain, channelID, m, mcache)
		if err != nil {
			return nil, err
		}
		out = append(out, cm)
	}
	return out, nil
}

func filterMessagesByAge(msgs []slackMessage, cutoff time.Time) []slackMessage {
	var out []slackMessage
	for _, m := range msgs {
		sec, nsec, ok := parseSlackTSNs(m.TS)
		if !ok {
			continue
		}
		t := time.Unix(sec, nsec).UTC()
		if !t.Before(cutoff) {
			out = append(out, m)
		}
	}
	return out
}

// FetchThread returns the parent message and up to 19 most recent replies (20 total),
// oldest first for the reply list (parent separate).
func FetchThread(ctx context.Context, api SlackAPI, channelID, threadTS string, creds connectors.Credentials, cache *sessionCache, mcache *MentionCache) (parent ContextMessage, replies []ContextMessage, truncated bool, err error) {
	if err := validateChannelID(channelID); err != nil {
		return ContextMessage{}, nil, false, err
	}
	if threadTS == "" {
		return ContextMessage{}, nil, false, &connectors.ValidationError{Message: "missing thread_ts for Slack thread context"}
	}
	sess, err := resolveSession(ctx, api, creds, cache)
	if err != nil {
		return ContextMessage{}, nil, false, err
	}
	body := readThreadRequest{
		Channel: channelID,
		TS:      threadTS,
		Limit:   1000,
	}
	var resp messagesResponse
	if err := api.Post(ctx, "conversations.replies", creds, body, &resp); err != nil {
		return ContextMessage{}, nil, false, err
	}
	if !resp.OK {
		return ContextMessage{}, nil, false, mapSlackErr(resp.Error)
	}
	if len(resp.Messages) == 0 {
		return ContextMessage{}, nil, false, &connectors.ExternalError{StatusCode: 200, Message: "Slack thread has no messages"}
	}
	sortMessagesOldestFirst(resp.Messages)
	parentSlack := resp.Messages[0]
	pmsg, err := messageToContextMessage(ctx, api, creds, sess.TeamDomain, channelID, parentSlack, mcache)
	if err != nil {
		return ContextMessage{}, nil, false, err
	}
	replySlack := resp.Messages[1:]
	if len(replySlack) > threadCap-1 {
		truncated = true
		replySlack = replySlack[len(replySlack)-(threadCap-1):]
	}
	replies = make([]ContextMessage, 0, len(replySlack))
	for _, m := range replySlack {
		cm, err := messageToContextMessage(ctx, api, creds, sess.TeamDomain, channelID, m, mcache)
		if err != nil {
			return ContextMessage{}, nil, false, err
		}
		replies = append(replies, cm)
	}
	return pmsg, replies, truncated, nil
}

// FetchDMHistory loads DM history for an IM channel opened with the given peer user ID.
// It returns SentinelSelfDM or SentinelFirstContact when those cases apply (no messages).
func FetchDMHistory(ctx context.Context, api SlackAPI, peerUserID string, creds connectors.Credentials, cache *sessionCache, mcache *MentionCache) (sentinel DMHistorySentinel, messages []ContextMessage, imChannelID string, err error) {
	if peerUserID == "" {
		return SentinelNone, nil, "", &connectors.ValidationError{Message: "missing user id for DM context"}
	}
	sess, err := resolveSession(ctx, api, creds, cache)
	if err != nil {
		return SentinelNone, nil, "", err
	}
	if peerUserID == sess.UserID {
		return SentinelSelfDM, nil, "", nil
	}
	openBody := conversationsOpenRequest{Users: peerUserID}
	var openResp conversationsOpenResponse
	if err := api.Post(ctx, "conversations.open", creds, openBody, &openResp); err != nil {
		return SentinelNone, nil, "", err
	}
	if !openResp.OK {
		return SentinelNone, nil, "", mapSlackErr(openResp.Error)
	}
	imChannelID = openResp.Channel.ID

	cutoff := time.Now().Add(-recentWindowHours * time.Hour).UTC()
	oldestTS := fmt.Sprintf("%d.%06d", cutoff.Unix(), cutoff.Nanosecond()/1000)
	histBody := readChannelHistoryRequest{
		Channel: imChannelID,
		Limit:   recentCapDM + 1,
		Oldest:  oldestTS,
	}
	var histResp messagesResponse
	if err := api.Post(ctx, "conversations.history", creds, histBody, &histResp); err != nil {
		return SentinelNone, nil, imChannelID, err
	}
	if !histResp.OK {
		return SentinelNone, nil, imChannelID, mapSlackErr(histResp.Error)
	}
	raw := filterMessagesByAge(histResp.Messages, cutoff)
	sortMessagesOldestFirst(raw)
	if len(raw) == 0 {
		return SentinelFirstContact, nil, imChannelID, nil
	}
	if len(raw) > recentCapDM {
		raw = raw[len(raw)-recentCapDM:]
	}
	for _, m := range raw {
		cm, err := messageToContextMessage(ctx, api, creds, sess.TeamDomain, imChannelID, m, mcache)
		if err != nil {
			return SentinelNone, nil, imChannelID, err
		}
		messages = append(messages, cm)
	}
	return SentinelNone, messages, imChannelID, nil
}

const (
	surroundBefore = 3
	surroundAfter  = 3
)

// MessageWindowAroundResult is the target message plus nearby channel messages (same 24h window as FetchRecentMessages).
type MessageWindowAroundResult struct {
	Target ContextMessage
	// TargetThreadRootTS is set when the target is a thread reply (Slack thread_ts); empty for top-level messages.
	TargetThreadRootTS string
	RecentSurrounding  []ContextMessage
}

// FetchMessageWindowAroundTS loads messages from the last 24h, finds the message with ts,
// and returns it as Target plus up to surroundBefore/surroundAfter neighbors (oldest first).
// If the target ts is not in the window, Target.TS is empty and RecentSurrounding is nil.
func FetchMessageWindowAroundTS(ctx context.Context, api SlackAPI, channelID, targetTS string, creds connectors.Credentials, cache *sessionCache, mcache *MentionCache) (MessageWindowAroundResult, error) {
	var out MessageWindowAroundResult
	if err := validateChannelID(channelID); err != nil {
		return out, err
	}
	if strings.TrimSpace(targetTS) == "" {
		return out, &connectors.ValidationError{Message: "missing message ts for Slack context"}
	}
	sess, err := resolveSession(ctx, api, creds, cache)
	if err != nil {
		return out, err
	}
	cutoff := time.Now().Add(-recentWindowHours * time.Hour).UTC()
	oldestTS := fmt.Sprintf("%d.%06d", cutoff.Unix(), cutoff.Nanosecond()/1000)
	body := readChannelHistoryRequest{
		Channel: channelID,
		Limit:   100,
		Oldest:  oldestTS,
	}
	var resp messagesResponse
	if err := api.Post(ctx, "conversations.history", creds, body, &resp); err != nil {
		return out, err
	}
	if !resp.OK {
		return out, mapSlackErr(resp.Error)
	}
	raw := filterMessagesByAge(resp.Messages, cutoff)
	sortMessagesOldestFirst(raw)
	idx := -1
	for i := range raw {
		if raw[i].TS == targetTS {
			idx = i
			break
		}
	}
	if idx < 0 {
		return out, nil
	}
	targetSlack := raw[idx]
	tcm, err := messageToContextMessage(ctx, api, creds, sess.TeamDomain, channelID, targetSlack, mcache)
	if err != nil {
		return out, err
	}
	out.Target = tcm
	if targetSlack.ThreadTS != "" && targetSlack.ThreadTS != targetSlack.TS {
		out.TargetThreadRootTS = targetSlack.ThreadTS
	}
	start := idx - surroundBefore
	if start < 0 {
		start = 0
	}
	end := idx + surroundAfter + 1
	if end > len(raw) {
		end = len(raw)
	}
	window := raw[start:end]
	out.RecentSurrounding = make([]ContextMessage, 0, len(window))
	for _, m := range window {
		cm, err := messageToContextMessage(ctx, api, creds, sess.TeamDomain, channelID, m, mcache)
		if err != nil {
			return out, err
		}
		out.RecentSurrounding = append(out.RecentSurrounding, cm)
	}
	return out, nil
}
