package x

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// followUserAction implements connectors.Action for x.follow_user.
// It follows a user via POST /2/users/{user_id}/following.
type followUserAction struct {
	conn *XConnector
}

// followUserParams are the parameters parsed from ActionRequest.Parameters.
type followUserParams struct {
	UserID       string `json:"user_id"`
	TargetUserID string `json:"target_user_id"`
}

func (p *followUserParams) validate() error {
	if p.UserID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: user_id"}
	}
	if p.TargetUserID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: target_user_id"}
	}
	return nil
}

// Execute follows a user and returns the result.
func (a *followUserAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params followUserParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]string{"target_user_id": params.TargetUserID}

	var xResp struct {
		Data struct {
			Following            bool `json:"following"`
			PendingFollow        bool `json:"pending_follow"`
		} `json:"data"`
	}

	path := "/users/" + url.PathEscape(params.UserID) + "/following"
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}

// unfollowUserAction implements connectors.Action for x.unfollow_user.
// It unfollows a user via DELETE /2/users/{user_id}/following/{target_user_id}.
type unfollowUserAction struct {
	conn *XConnector
}

// Execute unfollows a user and returns the result.
func (a *unfollowUserAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params followUserParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var xResp struct {
		Data struct {
			Following bool `json:"following"`
		} `json:"data"`
	}

	path := "/users/" + url.PathEscape(params.UserID) + "/following/" + url.PathEscape(params.TargetUserID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}
