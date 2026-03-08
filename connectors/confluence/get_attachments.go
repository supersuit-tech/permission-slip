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

// getAttachmentsAction implements connectors.Action for confluence.get_attachments.
// It lists attachments on a page via GET /wiki/api/v2/pages/{page_id}/attachments.
type getAttachmentsAction struct {
	conn *ConfluenceConnector
}

type getAttachmentsParams struct {
	PageID   string `json:"page_id"`
	Limit    int    `json:"limit,omitempty"`
	MediaType string `json:"media_type,omitempty"`
}

func (p *getAttachmentsParams) validate() error {
	p.PageID = strings.TrimSpace(p.PageID)
	if p.PageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: page_id"}
	}
	if p.Limit < 0 || p.Limit > 250 {
		return &connectors.ValidationError{Message: "limit must be between 1 and 250"}
	}
	return nil
}

type getAttachmentsResponse struct {
	Results []attachmentItem `json:"results"`
	Links   struct {
		Next string `json:"next,omitempty"`
	} `json:"_links"`
}

type attachmentItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	MediaType string `json:"mediaType"`
	FileSize  int64  `json:"fileSize"`
	Comment   string `json:"comment,omitempty"`
	Links     struct {
		Download string `json:"download"`
		WebUI    string `json:"webui"`
	} `json:"_links"`
}

func (a *getAttachmentsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getAttachmentsParams
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
	q.Set("limit", fmt.Sprintf("%d", limit))
	if params.MediaType != "" {
		q.Set("mediaType", params.MediaType)
	}

	path := "/pages/" + url.PathEscape(params.PageID) + "/attachments?" + q.Encode()

	var resp getAttachmentsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
