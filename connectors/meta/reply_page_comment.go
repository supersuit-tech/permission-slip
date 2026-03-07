package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// replyPageCommentAction implements connectors.Action for meta.reply_page_comment.
// It replies to a comment on a Facebook Page post via POST /{comment_id}/comments.
type replyPageCommentAction struct {
	conn *MetaConnector
}

type replyPageCommentParams struct {
	CommentID string `json:"comment_id"`
	Message   string `json:"message"`
}

func (p *replyPageCommentParams) validate() error {
	if p.CommentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: comment_id"}
	}
	if !isValidGraphID(p.CommentID) {
		return &connectors.ValidationError{Message: "comment_id contains invalid characters"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	return nil
}

type replyPageCommentRequest struct {
	Message string `json:"message"`
}

type replyPageCommentResponse struct {
	ID string `json:"id"`
}

func (a *replyPageCommentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params replyPageCommentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := replyPageCommentRequest{Message: params.Message}
	var resp replyPageCommentResponse

	reqURL := fmt.Sprintf("%s/%s/comments", a.conn.baseURL, params.CommentID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, reqURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id": resp.ID,
	})
}
