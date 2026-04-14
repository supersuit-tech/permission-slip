package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getUserProfileAction implements connectors.Action for slack.get_user_profile.
// It fetches a user profile via GET /users.info.
type getUserProfileAction struct {
	conn *SlackConnector
}

type getUserProfileParams struct {
	UserID string `json:"user_id"`
}

func (p *getUserProfileParams) validate() error {
	return validateUserID(p.UserID)
}

type getUserProfileResponse struct {
	slackResponse
	User *struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		RealName string `json:"real_name"`
		Profile  struct {
			DisplayName   string `json:"display_name"`
			RealName      string `json:"real_name"`
			Email         string `json:"email"`
			ImageOriginal string `json:"image_original"`
		} `json:"profile"`
	} `json:"user,omitempty"`
}

// Execute returns public profile fields for a Slack user.
func (a *getUserProfileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getUserProfileParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	var resp getUserProfileResponse
	if err := a.conn.doGet(ctx, "users.info", req.Credentials, map[string]string{"user": params.UserID}, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, resp.asError()
	}
	if resp.User == nil {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Slack API returned ok=true but no user data"}
	}
	return connectors.JSONResult(resp.User)
}
