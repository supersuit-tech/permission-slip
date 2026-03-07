package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updatePageAction implements connectors.Action for confluence.update_page.
// It updates a page via PUT /wiki/api/v2/pages/{page_id}.
type updatePageAction struct {
	conn *ConfluenceConnector
}

type updatePageParams struct {
	PageID         string `json:"page_id"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	VersionNumber  int    `json:"version_number"`
	VersionMessage string `json:"version_message"`
	Status         string `json:"status"`
}

func (p *updatePageParams) validate() error {
	p.PageID = strings.TrimSpace(p.PageID)
	if p.PageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: page_id"}
	}
	if p.VersionNumber <= 0 {
		return &connectors.ValidationError{Message: "missing required parameter: version_number (must be current version + 1)"}
	}
	if p.Title == "" && p.Body == "" {
		return &connectors.ValidationError{Message: "at least one of title or body is required"}
	}
	if p.Status != "" && p.Status != "current" && p.Status != "draft" {
		return &connectors.ValidationError{Message: "status must be \"current\" or \"draft\""}
	}
	return nil
}

func (a *updatePageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updatePageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	status := params.Status
	if status == "" {
		status = "current"
	}

	version := map[string]interface{}{
		"number": params.VersionNumber,
	}
	if params.VersionMessage != "" {
		version["message"] = params.VersionMessage
	}

	reqBody := map[string]interface{}{
		"id":      params.PageID,
		"status":  status,
		"version": version,
	}
	if params.Title != "" {
		reqBody["title"] = params.Title
	}
	if params.Body != "" {
		reqBody["body"] = map[string]interface{}{
			"representation": "storage",
			"value":          params.Body,
		}
	}

	var resp struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Status  string `json:"status"`
		Version struct {
			Number  int    `json:"number"`
			Message string `json:"message"`
		} `json:"version"`
	}

	path := "/pages/" + url.PathEscape(params.PageID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
