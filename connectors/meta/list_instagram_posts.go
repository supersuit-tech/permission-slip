package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listInstagramPostsAction implements connectors.Action for meta.list_instagram_posts.
// It lists recent media posts for an Instagram Business/Creator account
// via GET /{ig_account_id}/media.
type listInstagramPostsAction struct {
	conn *MetaConnector
}

type listInstagramPostsParams struct {
	InstagramAccountID string `json:"instagram_account_id"`
	Limit              int    `json:"limit,omitempty"`
}

func (p *listInstagramPostsParams) validate() error {
	if p.InstagramAccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: instagram_account_id"}
	}
	if !isValidGraphID(p.InstagramAccountID) {
		return &connectors.ValidationError{Message: "instagram_account_id contains invalid characters"}
	}
	if p.Limit < 0 || p.Limit > 100 {
		return &connectors.ValidationError{Message: "limit must be between 1 and 100"}
	}
	return nil
}

type listInstagramPostsResponse struct {
	Data   []instagramPost `json:"data"`
	Paging *pagingCursors  `json:"paging,omitempty"`
}

type instagramPost struct {
	ID           string `json:"id"`
	Caption      string `json:"caption,omitempty"`
	MediaType    string `json:"media_type,omitempty"`
	MediaURL     string `json:"media_url,omitempty"`
	Permalink    string `json:"permalink,omitempty"`
	Timestamp    string `json:"timestamp,omitempty"`
	LikeCount    int    `json:"like_count,omitempty"`
	CommentsCount int   `json:"comments_count,omitempty"`
}

func (a *listInstagramPostsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listInstagramPostsParams
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

	reqURL := fmt.Sprintf("%s/%s/media?fields=id,caption,media_type,media_url,permalink,timestamp,like_count,comments_count&limit=%s",
		a.conn.baseURL, params.InstagramAccountID, strconv.Itoa(limit))

	var resp listInstagramPostsResponse
	if err := a.conn.doGet(ctx, req.Credentials, reqURL, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
