package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// clientCountsResponse is the response from the undocumented client.counts endpoint.
// This endpoint is used by the Slack web client to get per-channel and per-DM unread
// state. It is undocumented and may change without notice.
type clientCountsResponse struct {
	slackResponse
	Channels []clientCountChannel `json:"channels"`
	Mpims    []clientCountChannel `json:"mpims"`
	Ims      []clientCountChannel `json:"ims"`
	Threads  clientCountThreads   `json:"threads"`
}

type clientCountChannel struct {
	ID            string  `json:"id"`
	HasUnreads    bool    `json:"has_unreads"`
	MentionCount  int     `json:"mention_count"`
	Latest        string  `json:"latest"`
}

type clientCountThreads struct {
	HasUnreads   bool `json:"has_unreads"`
	MentionCount int  `json:"mention_count"`
}

// clientCountsAction implements slack.client_counts.
type clientCountsAction struct {
	conn *SlackConnector
}

type clientCountsParams struct{}

func (p *clientCountsParams) validate() error { return nil }

type channelUnreadEntry struct {
	ChannelID   string `json:"channel_id"`
	ChannelType string `json:"channel_type"`
	HasUnreads  bool   `json:"has_unreads"`
	MentionCount int   `json:"mention_count"`
	LatestTS    string `json:"latest_ts,omitempty"`
}

type clientCountsResult struct {
	Channels    []channelUnreadEntry `json:"channels"`
	MPIMs       []channelUnreadEntry `json:"mpims"`
	IMs         []channelUnreadEntry `json:"ims"`
	ThreadsUnread bool               `json:"threads_unread"`
}

func (a *clientCountsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params clientCountsParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	var resp clientCountsResponse
	// client.counts is a GET endpoint — no parameters needed.
	if err := a.conn.doGet(ctx, "client.counts", req.Credentials, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, resp.asError()
	}

	entries := func(channels []clientCountChannel, channelType string) []channelUnreadEntry {
		var entries []channelUnreadEntry
		for _, ch := range channels {
			entries = append(entries, channelUnreadEntry{
				ChannelID:    ch.ID,
				ChannelType:  channelType,
				HasUnreads:   ch.HasUnreads,
				MentionCount: ch.MentionCount,
				LatestTS:     ch.Latest,
			})
		}
		return entries
	}

	return connectors.JSONResult(clientCountsResult{
		Channels:     entries(resp.Channels, "public_channel"),
		MPIMs:        entries(resp.Mpims, "mpim"),
		IMs:          entries(resp.Ims, "im"),
		ThreadsUnread: resp.Threads.HasUnreads,
	})
}
