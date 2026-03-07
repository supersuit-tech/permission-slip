package figma

import (
	"context"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// postCommentAction implements connectors.Action for figma.post_comment.
// It posts a comment on a file via POST /v1/files/:file_key/comments.
type postCommentAction struct {
	conn *FigmaConnector
}

type postCommentParams struct {
	FileKey   string `json:"file_key"`
	Message   string `json:"message"`
	CommentID string `json:"comment_id,omitempty"`
}

func (p *postCommentParams) validate() error {
	p.FileKey = extractFileKey(p.FileKey)
	if err := validateFileKey(p.FileKey); err != nil {
		return err
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	return nil
}

// postCommentRequest is the Figma API request body for posting comments.
type postCommentRequest struct {
	Message   string `json:"message"`
	CommentID string `json:"comment_id,omitempty"`
}

// Execute posts a comment on a Figma file.
func (a *postCommentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params postCommentParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/files/%s/comments", url.PathEscape(params.FileKey))

	body := postCommentRequest{
		Message:   params.Message,
		CommentID: params.CommentID,
	}

	var resp map[string]any
	if err := a.conn.doPost(ctx, path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
