package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type addCommentAction struct {
	conn *AsanaConnector
}

type addCommentParams struct {
	TaskID   string `json:"task_id"`
	Text     string `json:"text"`
	HTMLText string `json:"html_text"`
}

func (p *addCommentParams) validate() error {
	if p.TaskID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: task_id"}
	}
	if p.Text == "" && p.HTMLText == "" {
		return &connectors.ValidationError{Message: "either text or html_text is required"}
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

	body := map[string]any{}
	if params.Text != "" {
		body["text"] = params.Text
	}
	if params.HTMLText != "" {
		body["html_text"] = params.HTMLText
	}

	var resp struct {
		GID  string `json:"gid"`
		Text string `json:"text"`
	}

	path := fmt.Sprintf("/tasks/%s/stories", url.PathEscape(params.TaskID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
