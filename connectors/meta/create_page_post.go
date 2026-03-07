package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createPagePostAction implements connectors.Action for meta.create_page_post.
// It creates a post on a Facebook Page via POST /{page_id}/feed.
type createPagePostAction struct {
	conn *MetaConnector
}

type createPagePostParams struct {
	PageID    string `json:"page_id"`
	Message   string `json:"message"`
	Link      string `json:"link,omitempty"`
	Published *bool  `json:"published,omitempty"`
}

func (p *createPagePostParams) validate() error {
	if p.PageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: page_id"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	return nil
}

type createPagePostRequest struct {
	Message   string `json:"message"`
	Link      string `json:"link,omitempty"`
	Published *bool  `json:"published,omitempty"`
}

type createPagePostResponse struct {
	ID string `json:"id"`
}

func (a *createPagePostAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPagePostParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := createPagePostRequest{
		Message:   params.Message,
		Link:      params.Link,
		Published: params.Published,
	}

	var resp createPagePostResponse
	url := fmt.Sprintf("%s/%s/feed", a.conn.baseURL, params.PageID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, url, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id": resp.ID,
	})
}
