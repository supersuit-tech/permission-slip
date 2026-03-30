package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createCompanyPostAction implements connectors.Action for linkedin.create_company_post.
// It creates a post on behalf of a company page via POST /rest/posts.
type createCompanyPostAction struct {
	conn *LinkedInConnector
}

// createCompanyPostParams is the user-facing parameter schema.
type createCompanyPostParams struct {
	OrganizationID     string `json:"organization_id"`
	Text               string `json:"text"`
	Visibility         string `json:"visibility"`
	ArticleURL         string `json:"article_url"`
	ArticleTitle       string `json:"article_title"`
}

func (p *createCompanyPostParams) validate() error {
	if p.OrganizationID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: organization_id"}
	}
	if err := validateOrganizationID(p.OrganizationID); err != nil {
		return err
	}
	if p.Text == "" {
		return &connectors.ValidationError{Message: "missing required parameter: text"}
	}
	if len(p.Text) > maxPostTextLen {
		return &connectors.ValidationError{Message: fmt.Sprintf("text exceeds maximum length of %d characters", maxPostTextLen)}
	}
	// LinkedIn requires company page posts to be PUBLIC — the API rejects
	// other visibility values for organization authors.
	if p.Visibility != "" && p.Visibility != "PUBLIC" {
		return &connectors.ValidationError{Message: "visibility for company posts must be \"PUBLIC\""}
	}
	if err := validateArticleURL(p.ArticleURL); err != nil {
		return err
	}
	return nil
}

// Execute creates a post on behalf of a company page.
func (a *createCompanyPostAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCompanyPostParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := linkedInPostRequest{
		Author:         "urn:li:organization:" + params.OrganizationID,
		Commentary:     params.Text,
		Visibility:     "PUBLIC",
		LifecycleState: "PUBLISHED",
		Distribution:   linkedInDistribution{FeedDistribution: "MAIN_FEED"},
	}

	if params.ArticleURL != "" {
		body.Content = &linkedInPostContent{
			Article: &linkedInArticle{
				Source: params.ArticleURL,
				Title:  params.ArticleTitle,
			},
		}
	}

	apiURL := a.conn.restBaseURL + "/posts"
	respHeaders, err := a.conn.doWithHeaders(ctx, req.Credentials, http.MethodPost, apiURL, body, nil, true)
	if err != nil {
		return nil, err
	}

	result := map[string]string{
		"status":          "created",
		"organization_id": params.OrganizationID,
	}
	if postURN := respHeaders.Get("x-restli-id"); postURN != "" {
		result["post_urn"] = postURN
	}

	return connectors.JSONResult(result)
}
