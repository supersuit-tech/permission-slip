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

// searchTweetsAction implements connectors.Action for x.search_tweets.
// It searches tweets via GET /2/tweets/search/recent.
type searchTweetsAction struct {
	conn *XConnector
}

// searchTweetsParams are the parameters parsed from ActionRequest.Parameters.
type searchTweetsParams struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
	SinceID    string `json:"since_id"`
	SortOrder  string `json:"sort_order"`
}

var validSortOrders = map[string]bool{
	"recency":   true,
	"relevancy": true,
}

func (p *searchTweetsParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query"}
	}
	if p.MaxResults != 0 && (p.MaxResults < 10 || p.MaxResults > 100) {
		return &connectors.ValidationError{Message: "max_results must be between 10 and 100"}
	}
	if p.SortOrder != "" && !validSortOrders[p.SortOrder] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_order %q: must be recency or relevancy", p.SortOrder)}
	}
	return nil
}

// Execute searches recent tweets.
func (a *searchTweetsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchTweetsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	maxResults := params.MaxResults
	if maxResults == 0 {
		maxResults = 10
	}

	path := "/tweets/search/recent?query=" + url.QueryEscape(params.Query) +
		"&max_results=" + strconv.Itoa(maxResults) +
		"&tweet.fields=created_at,public_metrics,author_id"
	if params.SinceID != "" {
		path += "&since_id=" + url.QueryEscape(params.SinceID)
	}
	if params.SortOrder != "" {
		path += "&sort_order=" + url.QueryEscape(params.SortOrder)
	}

	var xResp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &xResp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: xResp}, nil
}
