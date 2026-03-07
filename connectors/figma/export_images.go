package figma

import (
	"context"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// exportImagesAction implements connectors.Action for figma.export_images.
// It exports PNG, SVG, PDF, or JPG from specific nodes via GET /v1/images/:file_key.
type exportImagesAction struct {
	conn *FigmaConnector
}

// validExportFormats are the allowed image export formats.
var validExportFormats = map[string]bool{
	"png": true,
	"svg": true,
	"pdf": true,
	"jpg": true,
}

type exportImagesParams struct {
	FileKey string   `json:"file_key"`
	NodeIDs string   `json:"node_ids"`
	Format  string   `json:"format"`
	Scale   *float64 `json:"scale,omitempty"`
}

func (p *exportImagesParams) validate() error {
	p.FileKey = extractFileKey(p.FileKey)
	if err := validateFileKey(p.FileKey); err != nil {
		return err
	}
	if err := validateNodeIDs(p.NodeIDs); err != nil {
		return err
	}
	if p.Format == "" {
		return &connectors.ValidationError{Message: "missing required parameter: format"}
	}
	if !validExportFormats[p.Format] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid format %q: must be png, svg, pdf, or jpg", p.Format)}
	}
	if p.Scale != nil && (*p.Scale < 0.01 || *p.Scale > 4) {
		return &connectors.ValidationError{Message: fmt.Sprintf("scale must be between 0.01 and 4, got %g", *p.Scale)}
	}
	return nil
}

// Execute exports images from a Figma file and returns download URLs.
func (a *exportImagesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params exportImagesParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/images/%s", url.PathEscape(params.FileKey))

	query := url.Values{}
	query.Set("ids", params.NodeIDs)
	query.Set("format", params.Format)
	if params.Scale != nil {
		query.Set("scale", fmt.Sprintf("%g", *params.Scale))
	}
	path += "?" + query.Encode()

	var resp map[string]any
	if err := a.conn.doGet(ctx, path, req.Credentials, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
