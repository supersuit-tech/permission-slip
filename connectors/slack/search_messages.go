package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// searchMessagesAction implements connectors.Action for slack.search_messages.
// It searches messages across channels via POST /search.messages.
//
// This Slack endpoint requires a user token (xoxp-) with the granular
// search:read.* scopes (public/private/im/mpim/files). The legacy monolithic
// search:read scope is no longer sufficient and causes invalid_arguments.
type searchMessagesAction struct {
	conn *SlackConnector
}

// searchMessagesParams is the user-facing parameter schema.
type searchMessagesParams struct {
	Query   string `json:"query"`
	Channel string `json:"channel,omitempty"` // optional scope for resolver + summary
	Count   int    `json:"count,omitempty"`
	Page    int    `json:"page,omitempty"`
	Sort    string `json:"sort,omitempty"`
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
	User      string            `json:"user"`
	Username  string            `json:"username"`
	Text      slackNullableText `json:"text"`
	TS        string            `json:"ts"`
	Permalink string            `json:"permalink"`
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

// Execute searches messages across Slack channels. It prefers the user access
// token (xoxp-) over the bot token since Slack's search.messages endpoint
// requires a user token.
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
		return nil, resp.asError()
	}

	filtered := make([]searchMatchSummary, 0, len(resp.Messages.Matches))
	for _, m := range resp.Messages.Matches {
		chID := m.Channel.ID
		filtered = append(filtered, searchMatchSummary{
			ChannelID:   chID,
			ChannelName: m.Channel.Name,
			User:        m.User,
			Username:    m.Username,
			Text:        m.Text.String(),
			TS:          m.TS,
			Permalink:   m.Permalink,
		})
	}

	result := searchMessagesResult{
		Matches: filtered,
		Total:   len(filtered),
		Page:    resp.Messages.Paging.Page,
		Pages:   resp.Messages.Paging.Pages,
	}

	return connectors.JSONResult(result)
}
