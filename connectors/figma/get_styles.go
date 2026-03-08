package figma

import (
	"context"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getStylesAction implements connectors.Action for figma.get_styles.
// It retrieves design styles/tokens from a file via GET /files/{file_key}/styles.
type getStylesAction struct {
	conn *FigmaConnector
}

type getStylesParams struct {
	FileKey string `json:"file_key"`
}

func (p *getStylesParams) validate() error {
	p.FileKey = extractFileKey(p.FileKey)
	return validateFileKey(p.FileKey)
}

type getStylesResponse struct {
	Meta struct {
		Styles []figmaStyle `json:"styles"`
	} `json:"meta"`
	Error bool `json:"error"`
}

type figmaStyle struct {
	Key         string `json:"key"`
	FileKey     string `json:"file_key"`
	NodeID      string `json:"node_id"`
	StyleType   string `json:"style_type"`
	ThumbnailURL string `json:"thumbnail_url"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (a *getStylesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getStylesParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	var resp getStylesResponse
	path := fmt.Sprintf("/files/%s/styles", params.FileKey)
	if err := a.conn.doGet(ctx, path, req.Credentials, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]interface{}{
		"styles": resp.Meta.Styles,
		"count":  len(resp.Meta.Styles),
	})
}
