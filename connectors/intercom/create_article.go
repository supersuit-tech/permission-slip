package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createArticleAction implements connectors.Action for intercom.create_article.
// It creates a help center article via POST /articles.
type createArticleAction struct {
	conn *IntercomConnector
}

type createArticleParams struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	AuthorID   int64  `json:"author_id"`
	State      string `json:"state"`       // "published" or "draft" (default)
	ParentID   int64  `json:"parent_id"`   // optional collection ID
	ParentType string `json:"parent_type"` // "collection" when parent_id is set
}

var validArticleStates = map[string]bool{
	"draft":     true,
	"published": true,
}

func (p *createArticleParams) validate() error {
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	if p.AuthorID == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: author_id (must be a non-zero integer admin ID)"}
	}
	if p.State != "" && !validArticleStates[p.State] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid state %q: must be draft or published", p.State)}
	}
	if p.ParentType != "" && p.ParentID == 0 {
		return &connectors.ValidationError{Message: "parent_type requires parent_id to be set"}
	}
	if p.ParentID != 0 && p.ParentType != "" && p.ParentType != "collection" {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid parent_type %q: must be collection", p.ParentType)}
	}
	return nil
}

func (a *createArticleAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createArticleParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	state := params.State
	if state == "" {
		state = "draft"
	}

	body := map[string]any{
		"title":     params.Title,
		"author_id": params.AuthorID,
		"state":     state,
	}
	if params.Body != "" {
		body["body"] = params.Body
	}
	if params.ParentID != 0 {
		body["parent_id"] = params.ParentID
		parentType := params.ParentType
		if parentType == "" {
			parentType = "collection"
		}
		body["parent_type"] = parentType
	}

	var resp intercomArticle
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/articles", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
