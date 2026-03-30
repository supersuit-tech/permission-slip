package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// addCommentAction implements connectors.Action for confluence.add_comment.
// It adds a footer comment to a page via POST /wiki/api/v2/pages/{page_id}/footer-comments.
type addCommentAction struct {
	conn *ConfluenceConnector
}

type addCommentParams struct {
	PageID string `json:"page_id"`
	Body   string `json:"body"`
}

func (p *addCommentParams) validate() error {
	p.PageID = strings.TrimSpace(p.PageID)
	if p.PageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: page_id"}
	}
	if strings.TrimSpace(p.Body) == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

func (a *addCommentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addCommentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	reqBody := map[string]interface{}{
		"body": map[string]interface{}{
			"representation": "storage",
			"value":          params.Body,
		},
	}

	var resp struct {
		ID string `json:"id"`
	}

	path := "/pages/" + url.PathEscape(params.PageID) + "/footer-comments"
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
