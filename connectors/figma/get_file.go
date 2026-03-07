package figma

import (
	"context"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getFileAction implements connectors.Action for figma.get_file.
// It retrieves a full design file tree with metadata via GET /v1/files/:file_key.
type getFileAction struct {
	conn *FigmaConnector
}

type getFileParams struct {
	FileKey string `json:"file_key"`
	Depth   int    `json:"depth,omitempty"`
	NodeIDs string `json:"node_ids,omitempty"`
}

func (p *getFileParams) validate() error {
	if err := validateFileKey(p.FileKey); err != nil {
		return err
	}
	if p.Depth < 0 {
		return &connectors.ValidationError{Message: "depth must be a positive integer"}
	}
	if p.NodeIDs != "" {
		if err := validateNodeIDs(p.NodeIDs); err != nil {
			return err
		}
	}
	return nil
}

// Execute retrieves a Figma file and returns the design tree.
func (a *getFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getFileParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/files/%s", url.PathEscape(params.FileKey))

	query := url.Values{}
	if params.Depth > 0 {
		query.Set("depth", fmt.Sprintf("%d", params.Depth))
	}
	if params.NodeIDs != "" {
		query.Set("ids", params.NodeIDs)
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
