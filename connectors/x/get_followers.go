package x

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getFollowersAction implements connectors.Action for x.get_followers.
// It fetches a user's followers via GET /2/users/{user_id}/followers.
type getFollowersAction struct {
	conn *XConnector
}

// getFollowersParams are the parameters parsed from ActionRequest.Parameters.
type getFollowersParams struct {
	// UserID is optional; if omitted the authenticated user's ID is resolved via /users/me.
	UserID          string `json:"user_id"`
	MaxResults      int    `json:"max_results"`
	PaginationToken string `json:"pagination_token"`
}

func (p *getFollowersParams) validate() error {
	if p.MaxResults != 0 && (p.MaxResults < 1 || p.MaxResults > 1000) {
		return &connectors.ValidationError{Message: "max_results must be between 1 and 1000"}
	}
	return nil
}

// Execute fetches a user's followers list.
func (a *getFollowersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getFollowersParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	userID, err := a.conn.resolveUserID(ctx, req.Credentials, params.UserID)
	if err != nil {
		return nil, err
	}

	maxResults := params.MaxResults
	if maxResults == 0 {
		maxResults = 100
	}

	path := "/users/" + url.PathEscape(userID) + "/followers" +
		"?max_results=" + strconv.Itoa(maxResults) +
		"&user.fields=id,name,username,description,public_metrics"
	if params.PaginationToken != "" {
		path += "&pagination_token=" + url.QueryEscape(params.PaginationToken)
	}

	var xResp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &xResp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: xResp}, nil
}

// getFollowingAction implements connectors.Action for x.get_following.
// It fetches users that a user follows via GET /2/users/{user_id}/following.
type getFollowingAction struct {
	conn *XConnector
}

// Execute fetches the list of users that a user follows.
func (a *getFollowingAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getFollowersParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	userID, err := a.conn.resolveUserID(ctx, req.Credentials, params.UserID)
	if err != nil {
		return nil, err
	}

	maxResults := params.MaxResults
	if maxResults == 0 {
		maxResults = 100
	}

	path := "/users/" + url.PathEscape(userID) + "/following" +
		"?max_results=" + strconv.Itoa(maxResults) +
		"&user.fields=id,name,username,description,public_metrics"
	if params.PaginationToken != "" {
		path += "&pagination_token=" + url.QueryEscape(params.PaginationToken)
	}

	var xResp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &xResp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: xResp}, nil
}
