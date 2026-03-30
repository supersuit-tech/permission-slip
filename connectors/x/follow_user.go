package x

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// followUserAction implements connectors.Action for x.follow_user.
// It follows a user via POST /2/users/{user_id}/following.
type followUserAction struct {
	conn *XConnector
}

// Execute follows a user and returns the result.
func (a *followUserAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params userFollowParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, errBadJSON(err)
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	userID, err := a.conn.resolveUserID(ctx, req.Credentials, params.UserID)
	if err != nil {
		return nil, err
	}

	body := map[string]string{"target_user_id": params.TargetUserID}

	var xResp struct {
		Data struct {
			Following     bool `json:"following"`
			PendingFollow bool `json:"pending_follow"`
		} `json:"data"`
	}

	path := "/users/" + url.PathEscape(userID) + "/following"
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
	var params userFollowParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, errBadJSON(err)
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	userID, err := a.conn.resolveUserID(ctx, req.Credentials, params.UserID)
	if err != nil {
		return nil, err
	}

	var xResp struct {
		Data struct {
			Following bool `json:"following"`
		} `json:"data"`
	}

	path := "/users/" + url.PathEscape(userID) + "/following/" + url.PathEscape(params.TargetUserID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}
