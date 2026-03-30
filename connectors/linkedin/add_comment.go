package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// addCommentAction implements connectors.Action for linkedin.add_comment.
// It adds a comment on a post via POST /rest/socialActions/{post_urn}/comments.
type addCommentAction struct {
	conn *LinkedInConnector
}

// addCommentParams is the user-facing parameter schema.
type addCommentParams struct {
	PostURN string `json:"post_urn"`
	Text    string `json:"text"`
}

const maxCommentTextLen = 1250

func (p *addCommentParams) validate() error {
	if p.PostURN == "" {
		return &connectors.ValidationError{Message: "missing required parameter: post_urn"}
	}
	if err := validatePostURN(p.PostURN); err != nil {
		return err
	}
	if p.Text == "" {
		return &connectors.ValidationError{Message: "missing required parameter: text"}
	}
	if len(p.Text) > maxCommentTextLen {
		return &connectors.ValidationError{Message: fmt.Sprintf("text exceeds maximum length of %d characters", maxCommentTextLen)}
	}
	return nil
}

// linkedInCommentRequest is the LinkedIn REST API request body for adding a comment.
type linkedInCommentRequest struct {
	Actor   string                      `json:"actor"`
	Message linkedInCommentMessage      `json:"message"`
}

type linkedInCommentMessage struct {
	Text string `json:"text"`
}

// Execute adds a comment on a LinkedIn post.
func (a *addCommentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addCommentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	personURN, err := a.conn.getPersonURN(ctx, req.Credentials)
	if err != nil {
		return nil, err
	}

	body := linkedInCommentRequest{
		Actor:   personURN,
		Message: linkedInCommentMessage{Text: params.Text},
	}

	encodedURN := url.PathEscape(params.PostURN)
	apiURL := a.conn.restBaseURL + "/socialActions/" + encodedURN + "/comments"
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, apiURL, body, nil, true); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":   "created",
		"post_urn": params.PostURN,
	})
}
