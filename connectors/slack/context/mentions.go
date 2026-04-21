package context

import (
	"context"
	"regexp"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// MentionCache stores resolved display strings for a single approval request.
type MentionCache struct {
	Users    map[string]string // Slack user ID -> @handle
	Channels map[string]string // Slack channel ID -> #name
}

func (m *MentionCache) userKey(id string) string {
	if m.Users == nil {
		m.Users = make(map[string]string)
	}
	return id
}

func (m *MentionCache) channelKey(id string) string {
	if m.Channels == nil {
		m.Channels = make(map[string]string)
	}
	return id
}

var (
	reUserMention    = regexp.MustCompile(`<@([UW][A-Z0-9]+)(?:\|([^>]*))?>`)
	reChannelMention = regexp.MustCompile(`<#(C[A-Z0-9]+)(?:\|([^>]*))?>`)
	reSubteamLabeled = regexp.MustCompile(`<!subteam\^(S[A-Z0-9]+)\|([^>]+)>`)
	reSubteamBare    = regexp.MustCompile(`<!subteam\^(S[A-Z0-9]+)>`)
)

// ResolveMentions replaces Slack mrkdwn mention tokens with human-readable
// text: users and channels via API (batched with MentionCache), broadcast
// keywords and labeled subteams without extra calls.
func ResolveMentions(ctx context.Context, text string, api SlackAPI, creds connectors.Credentials, cache *MentionCache) (string, error) {
	if text == "" {
		return "", nil
	}
	out := text
	out = strings.ReplaceAll(out, "<!here>", "@here")
	out = strings.ReplaceAll(out, "<!channel>", "@channel")
	out = reSubteamLabeled.ReplaceAllStringFunc(out, func(s string) string {
		parts := reSubteamLabeled.FindStringSubmatch(s)
		if len(parts) >= 3 && parts[2] != "" {
			return "@" + parts[2]
		}
		return s
	})
	out = reSubteamBare.ReplaceAllString(out, "@subteam")

	userMatches := reUserMention.FindAllStringSubmatch(out, -1)
	seenUsers := map[string]struct{}{}
	for _, m := range userMatches {
		if len(m) >= 2 {
			seenUsers[m[1]] = struct{}{}
		}
	}
	for uid := range seenUsers {
		if cache != nil && cache.Users != nil {
			if _, ok := cache.Users[uid]; ok {
				continue
			}
		}
		ref, err := fetchUserRef(ctx, api, creds, uid)
		if err != nil {
			return "", err
		}
		label := "@" + uid
		if ref != nil && ref.Name != "" {
			label = "@" + ref.Name
		}
		if cache != nil {
			cache.Users[cache.userKey(uid)] = label
		}
	}

	out = reUserMention.ReplaceAllStringFunc(out, func(s string) string {
		parts := reUserMention.FindStringSubmatch(s)
		if len(parts) < 2 {
			return s
		}
		uid := parts[1]
		if len(parts) >= 3 && parts[2] != "" {
			return "@" + parts[2]
		}
		if cache != nil && cache.Users != nil {
			if lab, ok := cache.Users[uid]; ok {
				return lab
			}
		}
		return "@" + uid
	})

	chMatches := reChannelMention.FindAllStringSubmatch(out, -1)
	seenCh := map[string]struct{}{}
	for _, m := range chMatches {
		if len(m) >= 2 {
			seenCh[m[1]] = struct{}{}
		}
	}
	for cid := range seenCh {
		if cache != nil && cache.Channels != nil {
			if _, ok := cache.Channels[cid]; ok {
				continue
			}
		}
		var resp conversationsInfoResponse
		body := conversationsInfoRequest{Channel: cid}
		if err := api.Post(ctx, "conversations.info", creds, body, &resp); err != nil {
			return "", err
		}
		if !resp.OK {
			return "", mapSlackErr(resp.Error)
		}
		label := "#" + cid
		if resp.Channel != nil && resp.Channel.Name != "" {
			label = "#" + resp.Channel.Name
		}
		if cache != nil {
			cache.Channels[cache.channelKey(cid)] = label
		}
	}

	out = reChannelMention.ReplaceAllStringFunc(out, func(s string) string {
		parts := reChannelMention.FindStringSubmatch(s)
		if len(parts) < 2 {
			return s
		}
		cid := parts[1]
		if len(parts) >= 3 && parts[2] != "" {
			return "#" + parts[2]
		}
		if cache != nil && cache.Channels != nil {
			if lab, ok := cache.Channels[cid]; ok {
				return lab
			}
		}
		return "#" + cid
	})

	return out, nil
}
