package dropbox

import (
	"context"
	"encoding/base64"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// downloadFileAction implements connectors.Action for dropbox.download_file.
type downloadFileAction struct {
	conn *DropboxConnector
}

type downloadFileParams struct {
	Path string `json:"path"`
}

func (p *downloadFileParams) validate() error {
	return validatePath(p.Path, "path")
}

type downloadAPIArg struct {
	Path string `json:"path"`
}

type downloadResultHeader struct {
	Name           string `json:"name"`
	PathDisplay    string `json:"path_display"`
	ID             string `json:"id"`
	Size           int64  `json:"size"`
	ServerModified string `json:"server_modified"`
	ClientModified string `json:"client_modified"`
	ContentHash    string `json:"content_hash"`
}

func (a *downloadFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params downloadFileParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	apiArg := downloadAPIArg{Path: params.Path}
	var metadata downloadResultHeader
	body, err := a.conn.doContent(ctx, "files/download", req.Credentials, apiArg, nil, &metadata)
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"name":            metadata.Name,
		"path_display":    metadata.PathDisplay,
		"id":              metadata.ID,
		"size":            metadata.Size,
		"content":         base64.StdEncoding.EncodeToString(body),
		"server_modified": metadata.ServerModified,
		"client_modified": metadata.ClientModified,
		"content_hash":    metadata.ContentHash,
	})
}
