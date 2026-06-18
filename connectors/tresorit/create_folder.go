package tresorit

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type createFolderAction struct {
	conn *TresoritConnector
}

type createFolderParams struct {
	Tresor string `json:"tresor"`
	Path   string `json:"path"`
}

func (p *createFolderParams) validate() error {
	if err := validateTresorName(p.Tresor); err != nil {
		return err
	}
	if err := validateObjectKey(p.Path, "path"); err != nil {
		return err
	}
	if !stringsHasSuffix(p.Path, "/") {
		p.Path += "/"
	}
	return nil
}

func (a *createFolderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[createFolderParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	_, err = a.conn.do(ctx, req.Credentials, "PUT", objectPath(params.Tresor, params.Path), "", []byte{}, "application/octet-stream")
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"tresor": params.Tresor,
		"path":   params.Path,
	})
}
