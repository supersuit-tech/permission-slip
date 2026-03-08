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

// listPagesAction implements connectors.Action for confluence.list_pages.
// It lists pages in a space via GET /wiki/api/v2/pages.
type listPagesAction struct {
	conn *ConfluenceConnector
}

type listPagesParams struct {
	SpaceID string `json:"space_id"`
	Limit   int    `json:"limit,omitempty"`
	Status  string `json:"status,omitempty"`
}

func (p *listPagesParams) validate() error {
	p.SpaceID = strings.TrimSpace(p.SpaceID)
	if p.SpaceID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: space_id"}
	}
	if p.Limit < 0 || p.Limit > 250 {
		return &connectors.ValidationError{Message: "limit must be between 1 and 250"}
	}
	validStatuses := map[string]bool{"current": true, "archived": true, "deleted": true, "trashed": true, "": true}
	if !validStatuses[strings.ToLower(p.Status)] {
		return &connectors.ValidationError{Message: "status must be one of: current, archived, deleted, trashed"}
	}
	return nil
}

type listPagesResponse struct {
	Results []pageListItem `json:"results"`
	Links   struct {
		Next string `json:"next,omitempty"`
	} `json:"_links"`
}

type pageListItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	SpaceID string `json:"spaceId"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Links struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
}

func (a *listPagesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listPagesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = 25
	}

	q := url.Values{}
	q.Set("space-id", params.SpaceID)
	q.Set("limit", fmt.Sprintf("%d", limit))
	if params.Status != "" {
		q.Set("status", strings.ToLower(params.Status))
	}

	path := "/pages?" + q.Encode()

	var resp listPagesResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
