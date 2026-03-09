package dropbox

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createFolderAction implements connectors.Action for dropbox.create_folder.
type createFolderAction struct {
	conn *DropboxConnector
}

type createFolderParams struct {
	Path       string `json:"path"`
	Autorename bool   `json:"autorename"`
}

func (p *createFolderParams) validate() error {
	return validatePath(p.Path, "path")
}

type createFolderRequest struct {
	Path       string `json:"path"`
	Autorename bool   `json:"autorename"`
}

type createFolderResponse struct {
	Metadata struct {
		Name        string `json:"name"`
		PathDisplay string `json:"path_display"`
		ID          string `json:"id"`
	} `json:"metadata"`
}

func (a *createFolderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createFolderParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := createFolderRequest{
		Path:       params.Path,
		Autorename: params.Autorename,
	}

	var resp createFolderResponse
	if err := a.conn.doRPC(ctx, "files/create_folder_v2", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"name":         resp.Metadata.Name,
		"path_display": resp.Metadata.PathDisplay,
		"id":           resp.Metadata.ID,
	})
}
