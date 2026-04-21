package context

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// FetchChannelMeta loads channel metadata and a channel-level permalink.
func FetchChannelMeta(ctx context.Context, api SlackAPI, channelID string, creds connectors.Credentials, cache *sessionCache) (*ChannelMeta, error) {
	if channelID == "" {
		return nil, &connectors.ValidationError{Message: "missing channel id for Slack context"}
	}
	sess, err := resolveSession(ctx, api, creds, cache)
	if err != nil {
		return nil, err
	}
	body := conversationsInfoRequest{Channel: channelID}
	var resp conversationsInfoResponse
	if err := api.Post(ctx, "conversations.info", creds, body, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, mapSlackErr(resp.Error)
	}
	if resp.Channel == nil {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Slack conversations.info returned ok=true but no channel"}
	}
	ch := resp.Channel
	meta := &ChannelMeta{
		ID:          ch.ID,
		Name:        ch.Name,
		IsPrivate:   ch.IsPrivate,
		IsDM:        ch.IsIM || ch.IsMPIM,
		Topic:       ch.Topic.Value,
		Purpose:     ch.Purpose.Value,
		MemberCount: ch.NumMembers,
		Permalink:   BuildChannelPermalink(sess.TeamDomain, ch.ID),
	}
	return meta, nil
}

// lastActivityISOFromTS converts a Slack message ts to RFC3339 UTC for last_activity_at.
func lastActivityISOFromTS(ts string) string {
	sec, nsec, ok := parseSlackTSNs(ts)
	if !ok {
		return ""
	}
	t := time.Unix(sec, nsec).UTC()
	return t.Format(time.RFC3339Nano)
}

// parseSlackTSNs parses a Slack message timestamp into Unix seconds and nanoseconds.
func parseSlackTSNs(ts string) (sec int64, nsec int64, ok bool) {
	if ts == "" {
		return 0, 0, false
	}
	dot := strings.IndexByte(ts, '.')
	if dot < 0 {
		s, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		return s, 0, true
	}
	whole := ts[:dot]
	frac := ts[dot+1:]
	for i := 0; i < len(whole); i++ {
		if whole[i] < '0' || whole[i] > '9' {
			return 0, 0, false
		}
	}
	for i := 0; i < len(frac); i++ {
		if frac[i] < '0' || frac[i] > '9' {
			return 0, 0, false
		}
	}
	sec, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	for len(frac) < 6 {
		frac += "0"
	}
	if len(frac) > 6 {
		frac = frac[:6]
	}
	usec, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return sec, usec * 1000, true
}

// WithLastActivity sets LastActivityAt when empty, using the latest ts among messages.
func WithLastActivity(meta *ChannelMeta, messageTS ...string) *ChannelMeta {
	if meta == nil {
		return nil
	}
	if meta.LastActivityAt != "" {
		return meta
	}
	var best string
	var bestSec int64 = -1
	var bestNsec int64
	for _, ts := range messageTS {
		sec, nsec, ok := parseSlackTSNs(ts)
		if !ok {
			continue
		}
		if bestSec < 0 || sec > bestSec || (sec == bestSec && nsec > bestNsec) {
			bestSec, bestNsec = sec, nsec
			best = ts
		}
	}
	if best != "" {
		meta.LastActivityAt = lastActivityISOFromTS(best)
	}
	return meta
}

func validateChannelID(channel string) error {
	if channel == "" {
		return &connectors.ValidationError{Message: "missing required parameter: channel"}
	}
	if len(channel) < 2 {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid channel ID %q: expected a Slack channel ID starting with C, G, or D", channel),
		}
	}
	switch channel[0] {
	case 'C', 'G', 'D':
		return nil
	default:
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid channel ID %q: expected a Slack channel ID starting with C, G, or D", channel),
		}
	}
}
