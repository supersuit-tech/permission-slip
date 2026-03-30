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

// deletePageAction implements connectors.Action for confluence.delete_page.
// It deletes (moves to trash) a page via DELETE /wiki/api/v2/pages/{page_id}.
type deletePageAction struct {
	conn *ConfluenceConnector
}

type deletePageParams struct {
	PageID string `json:"page_id"`
}

func (p *deletePageParams) validate() error {
	p.PageID = strings.TrimSpace(p.PageID)
	if p.PageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: page_id"}
	}
	return nil
}

func (a *deletePageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deletePageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/pages/" + url.PathEscape(params.PageID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":     params.PageID,
		"status": "deleted",
	})
}
