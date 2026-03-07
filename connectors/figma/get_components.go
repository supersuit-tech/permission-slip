package figma

import (
	"context"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getComponentsAction implements connectors.Action for figma.get_components.
// It retrieves design system components from a file via GET /v1/files/:file_key/components.
type getComponentsAction struct {
	conn *FigmaConnector
}

type getComponentsParams struct {
	FileKey string `json:"file_key"`
}

func (p *getComponentsParams) validate() error {
	p.FileKey = extractFileKey(p.FileKey)
	return validateFileKey(p.FileKey)
}

// Execute retrieves the components from a Figma file.
func (a *getComponentsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getComponentsParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/files/%s/components", url.PathEscape(params.FileKey))

	var resp map[string]any
	if err := a.conn.doGet(ctx, path, req.Credentials, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
