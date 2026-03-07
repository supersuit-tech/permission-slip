package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listPagePostsAction implements connectors.Action for meta.list_page_posts.
// It lists recent posts on a Facebook Page via GET /{page_id}/posts.
type listPagePostsAction struct {
	conn *MetaConnector
}

type listPagePostsParams struct {
	PageID string `json:"page_id"`
	Limit  int    `json:"limit,omitempty"`
	Since  int64  `json:"since,omitempty"`
	Until  int64  `json:"until,omitempty"`
}

func (p *listPagePostsParams) validate() error {
	if p.PageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: page_id"}
	}
	if !isValidGraphID(p.PageID) {
		return &connectors.ValidationError{Message: "page_id contains invalid characters"}
	}
	if p.Limit < 0 || p.Limit > 100 {
		return &connectors.ValidationError{Message: "limit must be between 1 and 100"}
	}
	return nil
}

type listPagePostsResponse struct {
	Data   []pagePost     `json:"data"`
	Paging *pagingCursors `json:"paging,omitempty"`
}

type pagePost struct {
	ID          string       `json:"id"`
	Message     string       `json:"message,omitempty"`
	CreatedTime string       `json:"created_time"`
	Shares      *shareCount  `json:"shares,omitempty"`
	Likes       *summaryData `json:"likes,omitempty"`
	Comments    *summaryData `json:"comments,omitempty"`
}

type shareCount struct {
	Count int `json:"count"`
}

type summaryData struct {
	Summary struct {
		TotalCount int `json:"total_count"`
	} `json:"summary"`
}

type pagingCursors struct {
	Next string `json:"next,omitempty"`
}

func (a *listPagePostsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listPagePostsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = 10
	}

	reqURL := fmt.Sprintf("%s/%s/posts?fields=id,message,created_time,shares,likes.summary(true),comments.summary(true)&limit=%d",
		a.conn.baseURL, params.PageID, limit)

	if params.Since > 0 {
		reqURL += "&since=" + strconv.FormatInt(params.Since, 10)
	}
	if params.Until > 0 {
		reqURL += "&until=" + strconv.FormatInt(params.Until, 10)
	}

	var resp listPagePostsResponse
	if err := a.conn.doGet(ctx, req.Credentials, reqURL, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
