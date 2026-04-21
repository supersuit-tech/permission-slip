package slack

import (
	"context"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
	slackctx "github.com/supersuit-tech/permission-slip/connectors/slack/context"
)

func buildSendMessageContext(ctx context.Context, c *SlackConnector, creds connectors.Credentials, params json.RawMessage, cache *slackctx.SessionCache) (*slackctx.SlackContext, error) {
	var p sendMessageParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	mcache := &slackctx.MentionCache{}
	chMeta, err := slackctx.FetchChannelMeta(ctx, c, p.Channel, creds, cache)
	if err != nil {
		if sc, ok := slackctx.HandleRateLimit(err); ok {
			if chMeta != nil {
				sc.Channel = chMeta
			}
			return sc, nil
		}
		return nil, err
	}
	out := &slackctx.SlackContext{Channel: chMeta}
	if p.ThreadTS != "" {
		parent, replies, truncated, err := slackctx.FetchThread(ctx, c, p.Channel, p.ThreadTS, creds, cache, mcache)
		if err != nil {
			if sc, ok := slackctx.HandleRateLimit(err); ok {
				out.ContextScope = slackctx.ScopeMetadataOnly
				return out, nil
			}
			return nil, err
		}
		out.ContextScope = slackctx.ScopeThread
		out.Thread = &slackctx.ThreadBlock{Parent: &parent, Replies: replies, Truncated: truncated}
		slackctx.WithLastActivity(out.Channel, append(threadTSToList(parent.TS, replies)...)...)
		return out, nil
	}
	opts := slackctx.RecentMessagesOpts{}
	if p.InResponseToTS != "" {
		opts.AnchorTS = p.InResponseToTS
	}
	recent, err := slackctx.FetchRecentMessages(ctx, c, p.Channel, creds, cache, opts, mcache)
	if err != nil {
		if sc, ok := slackctx.HandleRateLimit(err); ok {
			out.ContextScope = slackctx.ScopeMetadataOnly
			return out, nil
		}
		return nil, err
	}
	out.ContextScope = slackctx.ScopeRecentChannel
	out.RecentMessages = recent
	out.ContextWindow = &slackctx.ContextWindow{MessageCount: len(recent), Hours: 24}
	slackctx.WithLastActivity(out.Channel, tsListFromMessages(recent)...)
	return out, nil
}

func buildScheduleMessageContext(ctx context.Context, c *SlackConnector, creds connectors.Credentials, params json.RawMessage, cache *slackctx.SessionCache) (*slackctx.SlackContext, error) {
	var p scheduleMessageParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	mcache := &slackctx.MentionCache{}
	chMeta, err := slackctx.FetchChannelMeta(ctx, c, p.Channel, creds, cache)
	if err != nil {
		if sc, ok := slackctx.HandleRateLimit(err); ok {
			if chMeta != nil {
				sc.Channel = chMeta
			}
			return sc, nil
		}
		return nil, err
	}
	out := &slackctx.SlackContext{Channel: chMeta}
	if p.ThreadTS != "" {
		parent, replies, truncated, err := slackctx.FetchThread(ctx, c, p.Channel, p.ThreadTS, creds, cache, mcache)
		if err != nil {
			if sc, ok := slackctx.HandleRateLimit(err); ok {
				out.ContextScope = slackctx.ScopeMetadataOnly
				return out, nil
			}
			return nil, err
		}
		out.ContextScope = slackctx.ScopeThread
		out.Thread = &slackctx.ThreadBlock{Parent: &parent, Replies: replies, Truncated: truncated}
		slackctx.WithLastActivity(out.Channel, append(threadTSToList(parent.TS, replies)...)...)
		return out, nil
	}
	opts := slackctx.RecentMessagesOpts{}
	if p.InResponseToTS != "" {
		opts.AnchorTS = p.InResponseToTS
	}
	recent, err := slackctx.FetchRecentMessages(ctx, c, p.Channel, creds, cache, opts, mcache)
	if err != nil {
		if sc, ok := slackctx.HandleRateLimit(err); ok {
			out.ContextScope = slackctx.ScopeMetadataOnly
			return out, nil
		}
		return nil, err
	}
	out.ContextScope = slackctx.ScopeRecentChannel
	out.RecentMessages = recent
	out.ContextWindow = &slackctx.ContextWindow{MessageCount: len(recent), Hours: 24}
	slackctx.WithLastActivity(out.Channel, tsListFromMessages(recent)...)
	return out, nil
}

func buildSendDMContext(ctx context.Context, c *SlackConnector, creds connectors.Credentials, params json.RawMessage, cache *slackctx.SessionCache) (*slackctx.SlackContext, error) {
	var p sendDMParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	mcache := &slackctx.MentionCache{}
	recipient, err := slackctx.FetchUserRef(ctx, c, creds, p.UserID)
	if err != nil {
		if sc, ok := slackctx.HandleRateLimit(err); ok {
			return sc, nil
		}
		return nil, err
	}
	out := &slackctx.SlackContext{Recipient: recipient}
	sentinel, msgs, imCh, err := slackctx.FetchDMHistory(ctx, c, p.UserID, creds, cache, mcache)
	if err != nil {
		if sc, ok := slackctx.HandleRateLimit(err); ok {
			out.ContextScope = slackctx.ScopeMetadataOnly
			return out, nil
		}
		return nil, err
	}
	switch sentinel {
	case slackctx.SentinelSelfDM:
		out.ContextScope = slackctx.ScopeSelfDM
		imID := imCh
		if imID == "" {
			if opened, err := slackctx.OpenIMChannel(ctx, c, p.UserID, creds); err == nil {
				imID = opened
			} else if _, ok := slackctx.HandleRateLimit(err); ok {
				out.ContextScope = slackctx.ScopeMetadataOnly
				return out, nil
			}
		}
		if imID != "" {
			if chMeta, err := slackctx.FetchChannelMeta(ctx, c, imID, creds, cache); err == nil {
				out.Channel = chMeta
			} else if _, ok := slackctx.HandleRateLimit(err); ok {
				out.ContextScope = slackctx.ScopeMetadataOnly
			}
		}
		return out, nil
	case slackctx.SentinelFirstContact:
		out.ContextScope = slackctx.ScopeFirstContactDM
		if imCh != "" {
			if chMeta, err := slackctx.FetchChannelMeta(ctx, c, imCh, creds, cache); err == nil {
				out.Channel = chMeta
			} else if _, ok := slackctx.HandleRateLimit(err); ok {
				out.ContextScope = slackctx.ScopeMetadataOnly
			}
		}
		return out, nil
	}
	out.ContextScope = slackctx.ScopeRecentDM
	out.RecentMessages = msgs
	out.ContextWindow = &slackctx.ContextWindow{MessageCount: len(msgs), Hours: 24}
	if imCh != "" {
		if chMeta, err := slackctx.FetchChannelMeta(ctx, c, imCh, creds, cache); err == nil {
			slackctx.WithLastActivity(chMeta, tsListFromMessages(msgs)...)
			out.Channel = chMeta
		} else if _, ok := slackctx.HandleRateLimit(err); ok {
			out.ContextScope = slackctx.ScopeMetadataOnly
		}
	}
	return out, nil
}

func buildUpdateMessageContext(ctx context.Context, c *SlackConnector, creds connectors.Credentials, params json.RawMessage, cache *slackctx.SessionCache) (*slackctx.SlackContext, error) {
	var p updateMessageParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	return buildMessageTargetContext(ctx, c, creds, p.Channel, p.TS, cache)
}

func buildDeleteMessageContext(ctx context.Context, c *SlackConnector, creds connectors.Credentials, params json.RawMessage, cache *slackctx.SessionCache) (*slackctx.SlackContext, error) {
	var p deleteMessageParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	return buildMessageTargetContext(ctx, c, creds, p.Channel, p.TS, cache)
}

func buildMessageTargetContext(ctx context.Context, c *SlackConnector, creds connectors.Credentials, channelID, ts string, cache *slackctx.SessionCache) (*slackctx.SlackContext, error) {
	mcache := &slackctx.MentionCache{}
	chMeta, err := slackctx.FetchChannelMeta(ctx, c, channelID, creds, cache)
	if err != nil {
		if sc, ok := slackctx.HandleRateLimit(err); ok {
			if chMeta != nil {
				sc.Channel = chMeta
			}
			return sc, nil
		}
		return nil, err
	}
	out := &slackctx.SlackContext{Channel: chMeta}
	win, err := slackctx.FetchMessageWindowAroundTS(ctx, c, channelID, ts, creds, cache, mcache)
	if err != nil {
		if sc, ok := slackctx.HandleRateLimit(err); ok {
			out.ContextScope = slackctx.ScopeMetadataOnly
			return out, nil
		}
		return nil, err
	}
	if win.Target.TS == "" {
		out.ContextScope = slackctx.ScopeMetadataOnly
		return out, nil
	}
	if win.TargetThreadRootTS != "" {
		parent, replies, truncated, err := slackctx.FetchThread(ctx, c, channelID, win.TargetThreadRootTS, creds, cache, mcache)
		if err != nil {
			if sc, ok := slackctx.HandleRateLimit(err); ok {
				out.ContextScope = slackctx.ScopeMetadataOnly
				return out, nil
			}
			return nil, err
		}
		out.ContextScope = slackctx.ScopeThread
		out.Thread = &slackctx.ThreadBlock{Parent: &parent, Replies: replies, Truncated: truncated}
		out.TargetMessage = findTargetInThread(parent, replies, ts)
		if out.TargetMessage == nil {
			out.TargetMessage = &win.Target
		}
		slackctx.WithLastActivity(out.Channel, append(threadTSToList(parent.TS, replies)...)...)
		return out, nil
	}
	out.ContextScope = slackctx.ScopeRecentChannel
	out.TargetMessage = &win.Target
	out.RecentMessages = win.RecentSurrounding
	out.ContextWindow = &slackctx.ContextWindow{MessageCount: len(win.RecentSurrounding), Hours: 24}
	slackctx.WithLastActivity(out.Channel, tsListFromMessages(win.RecentSurrounding)...)
	return out, nil
}

func threadTSToList(parentTS string, replies []slackctx.ContextMessage) []string {
	out := []string{parentTS}
	for _, r := range replies {
		out = append(out, r.TS)
	}
	return out
}

func tsListFromMessages(msgs []slackctx.ContextMessage) []string {
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.TS)
	}
	return out
}

func findTargetInThread(parent slackctx.ContextMessage, replies []slackctx.ContextMessage, ts string) *slackctx.ContextMessage {
	if parent.TS == ts {
		return &parent
	}
	for i := range replies {
		if replies[i].TS == ts {
			return &replies[i]
		}
	}
	return nil
}
