package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// replyInstagramCommentAction implements connectors.Action for meta.reply_instagram_comment.
// It posts a reply to an Instagram comment via POST /{comment_id}/replies.
type replyInstagramCommentAction struct {
	conn *MetaConnector
}

type replyInstagramCommentParams struct {
	CommentID string `json:"comment_id"`
	Message   string `json:"message"`
}

func (p *replyInstagramCommentParams) validate() error {
	if p.CommentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: comment_id"}
	}
	if !isValidGraphID(p.CommentID) {
		return &connectors.ValidationError{Message: "comment_id contains invalid characters"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	if len(p.Message) > 2200 {
		return &connectors.ValidationError{Message: "message exceeds maximum length of 2200 characters"}
	}
	return nil
}

type replyInstagramCommentRequest struct {
	Message string `json:"message"`
}

type replyInstagramCommentResponse struct {
	ID string `json:"id"`
}

func (a *replyInstagramCommentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params replyInstagramCommentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := replyInstagramCommentRequest{Message: params.Message}
	var resp replyInstagramCommentResponse
	reqURL := fmt.Sprintf("%s/%s/replies", a.conn.baseURL, params.CommentID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, reqURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id": resp.ID,
	})
}
