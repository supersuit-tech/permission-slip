package slack

import (
	"context"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchMessagesAction implements connectors.Action for slack.search_messages.
// It searches messages across channels via POST /search.messages.
//
// This Slack endpoint requires a user token (xoxp-) with the search:read.*
// scopes. Bot tokens (xoxb-) do NOT support search.messages. The action
// automatically uses the user_access_token credential (populated by the Slack
// OAuth v2 flow) when available, falling back to the default access_token
// with a clear error if search fails due to missing permissions.
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

// Execute searches messages across Slack channels. It prefers the user access
// token (xoxp-) over the bot token since Slack's search.messages endpoint
// requires a user token.
func (a *searchMessagesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchMessagesParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	// Search results may include DMs and private channels. Require the user
	// to have a verified Slack identity so results can be filtered.
	if req.UserEmail == "" {
		return nil, &connectors.ValidationError{
			Message: "search_messages may return private content — add an email to your Permission Slip profile that matches your Slack account to proceed",
		}
	}

	slackUserID, err := a.conn.lookupSlackUserByEmail(ctx, req.Credentials, req.UserEmail)
	if err != nil {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("unable to verify Slack identity: %v", err),
		}
	}
	if slackUserID == "" {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("no Slack user found matching email %q — ensure your Permission Slip email matches your Slack account", req.UserEmail),
		}
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

	// Use the user access token for search (bot tokens don't support search.messages).
	// If no user token is available, fall back to the default token — the Slack API
	// will return a clear permission error.
	creds := req.Credentials
	if userToken, ok := creds.Get(credKeyUserAccessToken); ok && userToken != "" {
		creds = connectors.NewCredentials(map[string]string{credKeyAccessToken: userToken})
	}

	var resp searchMessagesResponse
	if err := a.conn.doPost(ctx, "search.messages", creds, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	// Filter search results to only include matches from channels the user
	// has access to. Cache membership checks to avoid repeated API calls for
	// multiple matches in the same channel.
	membershipCache := make(map[string]bool)
	filtered := make([]searchMatchSummary, 0, len(resp.Messages.Matches))
	for _, m := range resp.Messages.Matches {
		chID := m.Channel.ID
		allowed, cached := membershipCache[chID]
		if !cached {
			// Public channels are accessible; private/DM channels need a check.
			if len(chID) > 0 && chID[0] == 'C' {
				isPrivate, privErr := a.conn.isChannelPrivate(ctx, req.Credentials, chID)
				if privErr != nil {
					return nil, fmt.Errorf("checking channel %s visibility: %w", chID, privErr)
				}
				if !isPrivate {
					allowed = true
				} else {
					var memberErr error
					allowed, memberErr = a.conn.isUserInChannel(ctx, req.Credentials, chID, slackUserID)
					if memberErr != nil {
						return nil, fmt.Errorf("checking membership for channel %s: %w", chID, memberErr)
					}
				}
			} else {
				var memberErr error
				allowed, memberErr = a.conn.isUserInChannel(ctx, req.Credentials, chID, slackUserID)
				if memberErr != nil {
					return nil, fmt.Errorf("checking membership for channel %s: %w", chID, memberErr)
				}
			}
			membershipCache[chID] = allowed
		}
		if !allowed {
			continue
		}
		filtered = append(filtered, searchMatchSummary{
			ChannelID:   chID,
			ChannelName: m.Channel.Name,
			User:        m.User,
			Username:    m.Username,
			Text:        m.Text,
			TS:          m.TS,
			Permalink:   m.Permalink,
		})
	}

	result := searchMessagesResult{
		Matches: filtered,
		Total:   len(filtered),
		Page:    resp.Messages.Paging.Page,
		Pages:   resp.Messages.Paging.Pages, // upper bound; actual pages may be fewer after membership filtering
	}

	return connectors.JSONResult(result)
}
