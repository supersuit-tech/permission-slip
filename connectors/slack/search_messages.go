package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchMessagesAction implements connectors.Action for slack.search_messages.
// It searches messages across channels via POST /search.messages.
//
// IMPORTANT: This Slack endpoint requires a user token (xoxp-) with the
// search:read scope. Bot tokens (xoxb-) do NOT support search.messages.
// For this action to work, the OAuth flow must persist the authed_user
// access_token (returned alongside the bot token in Slack's OAuth v2
// response) into the connector's credentials. Until user-token credential
// support is added, this action will return a missing_scope / not_allowed
// error when invoked with a bot token. See the manifest description for
// user-facing guidance.
type searchMessagesAction struct {
	conn *SlackConnector
}

// searchMessagesParams is the user-facing parameter schema.
type searchMessagesParams struct {
	Query string `json:"query"`
	Count int    `json:"count,omitempty"`
	Page  int    `json:"page,omitempty"`
	Sort  string `json:"sort,omitempty"`
}

func (p *searchMessagesParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query"}
	}
	if p.Count != 0 && (p.Count < 1 || p.Count > 100) {
		return &connectors.ValidationError{Message: "count must be between 1 and 100"}
	}
	if p.Page < 0 {
		return &connectors.ValidationError{Message: "page must be at least 1"}
	}
	if p.Sort != "" && p.Sort != "score" && p.Sort != "timestamp" {
		return &connectors.ValidationError{Message: "sort must be \"score\" (relevance) or \"timestamp\""}
	}
	return nil
}

// searchMessagesRequest is the Slack API request body for search.messages.
type searchMessagesRequest struct {
	Query string `json:"query"`
	Count int    `json:"count,omitempty"`
	Page  int    `json:"page,omitempty"`
	Sort  string `json:"sort,omitempty"`
}

type searchMessagesResponse struct {
	slackResponse
	Messages struct {
		Matches []searchMatch       `json:"matches,omitempty"`
		Paging  searchMessagePaging `json:"paging"`
	} `json:"messages"`
}

type searchMatch struct {
	Channel struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channel"`
	User      string `json:"user"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	TS        string `json:"ts"`
	Permalink string `json:"permalink"`
}

type searchMessagePaging struct {
	Count int `json:"count"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Pages int `json:"pages"`
}

// searchMessagesResult is the action output.
type searchMessagesResult struct {
	Matches []searchMatchSummary `json:"matches"`
	Total   int                  `json:"total"`
	Page    int                  `json:"page"`
	Pages   int                  `json:"pages"`
}

type searchMatchSummary struct {
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	User        string `json:"user"`
	Username    string `json:"username,omitempty"`
	Text        string `json:"text"`
	TS          string `json:"ts"`
	Permalink   string `json:"permalink,omitempty"`
}

// Execute searches messages across Slack channels.
func (a *searchMessagesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchMessagesParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := searchMessagesRequest{
		Query: params.Query,
		Count: params.Count,
		Page:  params.Page,
		Sort:  params.Sort,
	}
	if body.Count == 0 {
		body.Count = 20
	}
	if body.Page == 0 {
		body.Page = 1
	}

	var resp searchMessagesResponse
	if err := a.conn.doPost(ctx, "search.messages", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	result := searchMessagesResult{
		Matches: make([]searchMatchSummary, 0, len(resp.Messages.Matches)),
		Total:   resp.Messages.Paging.Total,
		Page:    resp.Messages.Paging.Page,
		Pages:   resp.Messages.Paging.Pages,
	}
	for _, m := range resp.Messages.Matches {
		result.Matches = append(result.Matches, searchMatchSummary{
			ChannelID:   m.Channel.ID,
			ChannelName: m.Channel.Name,
			User:        m.User,
			Username:    m.Username,
			Text:        m.Text,
			TS:          m.TS,
			Permalink:   m.Permalink,
		})
	}

	return connectors.JSONResult(result)
}
