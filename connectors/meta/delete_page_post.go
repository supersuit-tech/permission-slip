package meta

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deletePagePostAction implements connectors.Action for meta.delete_page_post.
// It deletes a Facebook Page post via DELETE /{post_id}.
type deletePagePostAction struct {
	conn *MetaConnector
}

type deletePagePostParams struct {
	PostID string `json:"post_id"`
}

func (p *deletePagePostParams) validate() error {
	if p.PostID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: post_id"}
	}
	if !isValidGraphID(p.PostID) {
		return &connectors.ValidationError{Message: "post_id contains invalid characters"}
	}
	return nil
}

func (a *deletePagePostAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deletePagePostParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/%s", a.conn.baseURL, params.PostID)
	if err := a.conn.doDelete(ctx, req.Credentials, reqURL); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"status":  "deleted",
		"post_id": params.PostID,
	})
}
