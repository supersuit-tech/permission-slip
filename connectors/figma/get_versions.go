package figma

import (
	"context"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getVersionsAction implements connectors.Action for figma.get_versions.
// It retrieves the version history for a file via GET /v1/files/:file_key/versions.
type getVersionsAction struct {
	conn *FigmaConnector
}

type getVersionsParams struct {
	FileKey string `json:"file_key"`
}

func (p *getVersionsParams) validate() error {
	p.FileKey = extractFileKey(p.FileKey)
	return validateFileKey(p.FileKey)
}

// Execute retrieves the version history of a Figma file.
func (a *getVersionsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getVersionsParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/files/%s/versions", url.PathEscape(params.FileKey))

	var resp map[string]any
	if err := a.conn.doGet(ctx, path, req.Credentials, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
