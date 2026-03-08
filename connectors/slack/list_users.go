package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listUsersAction implements connectors.Action for slack.list_users.
// It lists workspace users via POST /users.list.
type listUsersAction struct {
	conn *SlackConnector
}

// listUsersParams defines the user-facing parameter schema.
type listUsersParams struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

func (p *listUsersParams) validate() error {
	return validateLimit(p.Limit)
}

// listUsersRequest is the Slack API request body for users.list.
type listUsersRequest struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type listUsersResponse struct {
	slackResponse
	Members []listUserEntry `json:"members,omitempty"`
	Meta    *paginationMeta `json:"response_metadata,omitempty"`
}

type listUserEntry struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	RealName string          `json:"real_name"`
	Deleted  bool            `json:"deleted"`
	IsBot    bool            `json:"is_bot"`
	IsAdmin  bool            `json:"is_admin"`
	Profile  listUserProfile `json:"profile"`
}

type listUserProfile struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

// listUsersResult is the action output.
type listUsersResult struct {
	Users      []listUserSummary `json:"users"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type listUserSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RealName    string `json:"real_name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Email       string `json:"email,omitempty"`
	IsBot       bool   `json:"is_bot"`
	Deleted     bool   `json:"deleted"`
}

// Execute lists workspace users visible to the bot.
func (a *listUsersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listUsersParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := listUsersRequest{
		Limit:  params.Limit,
		Cursor: params.Cursor,
	}
	if body.Limit == 0 {
		body.Limit = 100
	}

	var resp listUsersResponse
	if err := a.conn.doPost(ctx, "users.list", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	result := listUsersResult{
		Users: make([]listUserSummary, 0, len(resp.Members)),
	}
	for _, u := range resp.Members {
		result.Users = append(result.Users, listUserSummary{
			ID:          u.ID,
			Name:        u.Name,
			RealName:    u.RealName,
			DisplayName: u.Profile.DisplayName,
			Email:       u.Profile.Email,
			IsBot:       u.IsBot,
			Deleted:     u.Deleted,
		})
	}
	if resp.Meta != nil {
		result.NextCursor = resp.Meta.NextCursor
	}

	return connectors.JSONResult(result)
}
