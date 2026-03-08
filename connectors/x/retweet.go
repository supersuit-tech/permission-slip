package x

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// retweetAction implements connectors.Action for x.retweet.
// It retweets a tweet via POST /2/users/{user_id}/retweets.
type retweetAction struct {
	conn *XConnector
}

// Execute retweets a tweet and returns the result.
func (a *retweetAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params userTweetParams
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

	body := map[string]string{"tweet_id": params.TweetID}

	var xResp struct {
		Data struct {
			Retweeted bool `json:"retweeted"`
		} `json:"data"`
	}

	path := "/users/" + url.PathEscape(userID) + "/retweets"
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}

// unretweetAction implements connectors.Action for x.unretweet.
// It undoes a retweet via DELETE /2/users/{user_id}/retweets/{source_tweet_id}.
type unretweetAction struct {
	conn *XConnector
}

// Execute undoes a retweet and returns the result.
func (a *unretweetAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params userTweetParams
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
			Retweeted bool `json:"retweeted"`
		} `json:"data"`
	}

	path := "/users/" + url.PathEscape(userID) + "/retweets/" + url.PathEscape(params.TweetID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}
