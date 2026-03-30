package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createPostAction implements connectors.Action for linkedin.create_post.
// It creates a post on the authenticated user's LinkedIn feed via
// POST /rest/posts.
type createPostAction struct {
	conn *LinkedInConnector
}

// createPostParams is the user-facing parameter schema.
type createPostParams struct {
	Text               string `json:"text"`
	Visibility         string `json:"visibility"`
	ArticleURL         string `json:"article_url"`
	ArticleTitle       string `json:"article_title"`
	ArticleDescription string `json:"article_description"`
}

const maxPostTextLen = 3000

func (p *createPostParams) validate() error {
	if p.Text == "" {
		return &connectors.ValidationError{Message: "missing required parameter: text"}
	}
	if len(p.Text) > maxPostTextLen {
		return &connectors.ValidationError{Message: fmt.Sprintf("text exceeds maximum length of %d characters", maxPostTextLen)}
	}
	if p.Visibility != "" && p.Visibility != "PUBLIC" && p.Visibility != "CONNECTIONS" {
		return &connectors.ValidationError{Message: "visibility must be \"PUBLIC\" or \"CONNECTIONS\""}
	}
	if err := validateArticleURL(p.ArticleURL); err != nil {
		return err
	}
	return nil
}

// linkedInPostRequest is the LinkedIn REST API request body for creating a post.
// Used by both create_post (personal) and create_company_post (organization).
// Ref: https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/posts-api
type linkedInPostRequest struct {
	Author         string                  `json:"author"`
	Commentary     string                  `json:"commentary"`
	Visibility     string                  `json:"visibility"`
	LifecycleState string                  `json:"lifecycleState"`
	Distribution   linkedInDistribution    `json:"distribution"`
	Content        *linkedInPostContent    `json:"content,omitempty"`
}

type linkedInDistribution struct {
	FeedDistribution string `json:"feedDistribution"`
}

type linkedInPostContent struct {
	Article *linkedInArticle `json:"article,omitempty"`
}

type linkedInArticle struct {
	Source      string `json:"source"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// Execute creates a post on the authenticated user's LinkedIn feed.
func (a *createPostAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPostParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Get the user's person URN from their profile.
	personURN, err := a.conn.getPersonURN(ctx, req.Credentials)
	if err != nil {
		return nil, err
	}

	visibility := params.Visibility
	if visibility == "" {
		visibility = "PUBLIC"
	}

	body := linkedInPostRequest{
		Author:         personURN,
		Commentary:     params.Text,
		Visibility:     visibility,
		LifecycleState: "PUBLISHED",
		Distribution:   linkedInDistribution{FeedDistribution: "MAIN_FEED"},
	}

	// Add article content for link shares.
	if params.ArticleURL != "" {
		body.Content = &linkedInPostContent{
			Article: &linkedInArticle{
				Source:      params.ArticleURL,
				Title:       params.ArticleTitle,
				Description: params.ArticleDescription,
			},
		}
	}

	apiURL := a.conn.restBaseURL + "/posts"
	respHeaders, err := a.conn.doWithHeaders(ctx, req.Credentials, http.MethodPost, apiURL, body, nil, true)
	if err != nil {
		return nil, err
	}

	result := map[string]string{
		"status": "created",
	}
	// LinkedIn returns the post URN in x-restli-id on 201 Created.
	if postURN := respHeaders.Get("x-restli-id"); postURN != "" {
		result["post_urn"] = postURN
	}

	return connectors.JSONResult(result)
}

