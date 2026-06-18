package tresorit

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type deleteFileAction struct {
	conn *TresoritConnector
}

type deleteFileParams struct {
	Tresor string `json:"tresor"`
	Key    string `json:"key"`
}

func (p *deleteFileParams) validate() error {
	if err := validateTresorName(p.Tresor); err != nil {
		return err
	}
	return validateObjectKey(p.Key, "key")
}

func (a *deleteFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[deleteFileParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	_, err = a.conn.do(ctx, req.Credentials, "DELETE", objectPath(params.Tresor, params.Key), "", nil, "")
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"tresor":  params.Tresor,
		"key":     params.Key,
		"deleted": true,
	})
}
