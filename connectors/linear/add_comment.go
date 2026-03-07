package linear

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// addCommentAction implements connectors.Action for linear.add_comment.
type addCommentAction struct {
	conn *LinearConnector
}

type addCommentParams struct {
	IssueID string `json:"issue_id"`
	Body    string `json:"body"`
}

func (p *addCommentParams) validate() error {
	if p.IssueID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_id"}
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

const addCommentMutation = `mutation CommentCreate($input: CommentCreateInput!) {
	commentCreate(input: $input) {
		success
		comment {
			id
			body
			url
		}
	}
}`

type addCommentResponse struct {
	CommentCreate struct {
		Success bool `json:"success"`
		Comment struct {
			ID   string `json:"id"`
			Body string `json:"body"`
			URL  string `json:"url"`
		} `json:"comment"`
	} `json:"commentCreate"`
}

func (a *addCommentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addCommentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	input := map[string]any{
		"issueId": params.IssueID,
		"body":    params.Body,
	}

	var resp addCommentResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, addCommentMutation, map[string]any{"input": input}, &resp); err != nil {
		return nil, err
	}

	if !resp.CommentCreate.Success {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Linear commentCreate returned success=false"}
	}

	return connectors.JSONResult(map[string]string{
		"id":   resp.CommentCreate.Comment.ID,
		"body": resp.CommentCreate.Comment.Body,
		"url":  resp.CommentCreate.Comment.URL,
	})
}
