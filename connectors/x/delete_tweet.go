package x

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteTweetAction implements connectors.Action for x.delete_tweet.
// It deletes a tweet via DELETE /2/tweets/{tweet_id}.
type deleteTweetAction struct {
	conn *XConnector
}

// deleteTweetParams are the parameters parsed from ActionRequest.Parameters.
type deleteTweetParams struct {
	TweetID string `json:"tweet_id"`
}

func (p *deleteTweetParams) validate() error {
	if p.TweetID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: tweet_id"}
	}
	return nil
}

// Execute deletes a tweet and returns the deletion confirmation.
func (a *deleteTweetAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteTweetParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var xResp struct {
		Data struct {
			Deleted bool `json:"deleted"`
		} `json:"data"`
	}

	path := "/tweets/" + url.PathEscape(params.TweetID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}
