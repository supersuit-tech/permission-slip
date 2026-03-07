package figma

import (
	"context"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listCommentsAction implements connectors.Action for figma.list_comments.
// It lists comments on a file via GET /v1/files/:file_key/comments.
type listCommentsAction struct {
	conn *FigmaConnector
}

type listCommentsParams struct {
	FileKey string `json:"file_key"`
	AsMd    bool   `json:"as_md,omitempty"`
}

func (p *listCommentsParams) validate() error {
	p.FileKey = extractFileKey(p.FileKey)
	return validateFileKey(p.FileKey)
}

// Execute lists comments on a Figma file.
func (a *listCommentsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listCommentsParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/files/%s/comments", url.PathEscape(params.FileKey))

	query := url.Values{}
	if params.AsMd {
		query.Set("as_md", "true")
	}
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	var resp map[string]any
	if err := a.conn.doGet(ctx, path, req.Credentials, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
