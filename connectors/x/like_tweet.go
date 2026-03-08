package x

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// likeTweetAction implements connectors.Action for x.like_tweet.
// It likes a tweet via POST /2/users/{user_id}/likes.
type likeTweetAction struct {
	conn *XConnector
}

// likeTweetParams are the parameters parsed from ActionRequest.Parameters.
type likeTweetParams struct {
	// UserID is optional; if omitted the authenticated user's ID is resolved via /users/me.
	UserID  string `json:"user_id"`
	TweetID string `json:"tweet_id"`
}

func (p *likeTweetParams) validate() error {
	if p.TweetID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: tweet_id"}
	}
	return nil
}

// Execute likes a tweet and returns the result.
func (a *likeTweetAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params likeTweetParams
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

	body := map[string]string{"tweet_id": params.TweetID}

	var xResp struct {
		Data struct {
			Liked bool `json:"liked"`
		} `json:"data"`
	}

	path := "/users/" + url.PathEscape(userID) + "/likes"
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}

// unlikeTweetAction implements connectors.Action for x.unlike_tweet.
// It unlikes a tweet via DELETE /2/users/{user_id}/likes/{tweet_id}.
type unlikeTweetAction struct {
	conn *XConnector
}

// Execute unlikes a tweet and returns the result.
func (a *unlikeTweetAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params likeTweetParams
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

	var xResp struct {
		Data struct {
			Liked bool `json:"liked"`
		} `json:"data"`
	}

	path := "/users/" + url.PathEscape(userID) + "/likes/" + url.PathEscape(params.TweetID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}
