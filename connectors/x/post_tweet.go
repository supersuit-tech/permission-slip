package x

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// postTweetAction implements connectors.Action for x.post_tweet.
// It posts a tweet via POST /2/tweets.
type postTweetAction struct {
	conn *XConnector
}

// postTweetParams are the parameters parsed from ActionRequest.Parameters.
type postTweetParams struct {
	Text           string   `json:"text"`
	ReplyToTweetID string   `json:"reply_to_tweet_id"`
	QuoteTweetID   string   `json:"quote_tweet_id"`
	MediaIDs       []string `json:"media_ids"`
}

func (p *postTweetParams) validate() error {
	if p.Text == "" {
		return &connectors.ValidationError{Message: "missing required parameter: text"}
	}
	if len(p.Text) > 280 {
		return &connectors.ValidationError{Message: "text exceeds 280 character limit"}
	}
	return nil
}

// Execute posts a tweet and returns the created tweet data.
func (a *postTweetAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params postTweetParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build the X API request body.
	body := map[string]any{"text": params.Text}
	if params.ReplyToTweetID != "" {
		body["reply"] = map[string]string{"in_reply_to_tweet_id": params.ReplyToTweetID}
	}
	if params.QuoteTweetID != "" {
		body["quote_tweet_id"] = params.QuoteTweetID
	}
	if len(params.MediaIDs) > 0 {
		body["media"] = map[string]any{"media_ids": params.MediaIDs}
	}

	var xResp struct {
		Data struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"data"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/tweets", body, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}
