package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getPostAnalyticsAction implements connectors.Action for linkedin.get_post_analytics.
// It retrieves engagement metrics for a post via GET /rest/socialActions/{post_urn}.
type getPostAnalyticsAction struct {
	conn *LinkedInConnector
}

// getPostAnalyticsParams is the user-facing parameter schema.
type getPostAnalyticsParams struct {
	PostURN string `json:"post_urn"`
}

func (p *getPostAnalyticsParams) validate() error {
	if p.PostURN == "" {
		return &connectors.ValidationError{Message: "missing required parameter: post_urn"}
	}
	if err := validatePostURN(p.PostURN); err != nil {
		return err
	}
	return nil
}

// socialActionsResponse is the LinkedIn REST API response for social actions.
type socialActionsResponse struct {
	LikesSummary    socialActionsSummary `json:"likesSummary"`
	CommentsSummary socialActionsSummary `json:"commentsSummary"`
}

type socialActionsSummary struct {
	TotalLikes    int `json:"totalLikes"`
	LikedByCurrentUser bool `json:"likedByCurrentUser"`
	TotalComments int `json:"totalComments"`
}

// Execute retrieves engagement metrics for a LinkedIn post.
func (a *getPostAnalyticsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getPostAnalyticsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	encodedURN := url.PathEscape(params.PostURN)
	apiURL := a.conn.restBaseURL + "/socialActions/" + encodedURN

	var resp socialActionsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, apiURL, nil, &resp, true); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"post_urn":              params.PostURN,
		"likes":                 resp.LikesSummary.TotalLikes,
		"liked_by_current_user": resp.LikesSummary.LikedByCurrentUser,
		"comments":              resp.CommentsSummary.TotalComments,
	})
}
