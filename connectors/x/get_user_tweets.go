package x

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getUserTweetsAction implements connectors.Action for x.get_user_tweets.
// It fetches tweets via GET /2/users/{user_id}/tweets.
type getUserTweetsAction struct {
	conn *XConnector
}

// getUserTweetsParams are the parameters parsed from ActionRequest.Parameters.
type getUserTweetsParams struct {
	UserID     string `json:"user_id"`
	MaxResults int    `json:"max_results"`
	SinceID    string `json:"since_id"`
	UntilID    string `json:"until_id"`
}

func (p *getUserTweetsParams) validate() error {
	if p.UserID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: user_id"}
	}
	if p.MaxResults != 0 && (p.MaxResults < 1 || p.MaxResults > 100) {
		return &connectors.ValidationError{Message: "max_results must be between 1 and 100"}
	}
	return nil
}

// Execute fetches recent tweets from a user.
func (a *getUserTweetsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getUserTweetsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build query string.
	maxResults := params.MaxResults
	if maxResults == 0 {
		maxResults = 10
	}

	path := "/users/" + url.PathEscape(params.UserID) + "/tweets?max_results=" + strconv.Itoa(maxResults) +
		"&tweet.fields=created_at,public_metrics"
	if params.SinceID != "" {
		path += "&since_id=" + url.QueryEscape(params.SinceID)
	}
	if params.UntilID != "" {
		path += "&until_id=" + url.QueryEscape(params.UntilID)
	}

	var xResp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &xResp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: xResp}, nil
}
