package dropbox

import (
	"context"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// moveAction implements connectors.Action for dropbox.move.
type moveAction struct {
	conn *DropboxConnector
}

type moveParams struct {
	FromPath                string `json:"from_path"`
	ToPath                  string `json:"to_path"`
	Autorename              bool   `json:"autorename"`
	AllowOwnershipTransfer  bool   `json:"allow_ownership_transfer"`
}

func (p *moveParams) validate() error {
	if err := validatePath(p.FromPath, "from_path"); err != nil {
		return err
	}
	if err := validatePath(p.ToPath, "to_path"); err != nil {
		return err
	}
	// Dropbox paths are case-insensitive — catch no-op moves early.
	if strings.EqualFold(p.FromPath, p.ToPath) {
		return &connectors.ValidationError{Message: "from_path and to_path are the same (Dropbox paths are case-insensitive)"}
	}
	return nil
}

type moveRequest struct {
	FromPath                string `json:"from_path"`
	ToPath                  string `json:"to_path"`
	Autorename              bool   `json:"autorename"`
	AllowOwnershipTransfer  bool   `json:"allow_ownership_transfer"`
}

type moveResponse struct {
	Metadata struct {
		Name        string `json:"name"`
		PathDisplay string `json:"path_display"`
		ID          string `json:"id"`
	} `json:"metadata"`
}

func (a *moveAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params moveParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := moveRequest{
		FromPath:                params.FromPath,
		ToPath:                  params.ToPath,
		Autorename:              params.Autorename,
		AllowOwnershipTransfer:  params.AllowOwnershipTransfer,
	}

	var resp moveResponse
	if err := a.conn.doRPC(ctx, "files/move_v2", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"name":         resp.Metadata.Name,
		"path_display": resp.Metadata.PathDisplay,
		"id":           resp.Metadata.ID,
	})
}
