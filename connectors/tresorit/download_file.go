package tresorit

import (
	"context"
	"encoding/base64"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type downloadFileAction struct {
	conn *TresoritConnector
}

type downloadFileParams struct {
	Tresor string `json:"tresor"`
	Key    string `json:"key"`
}

func (p *downloadFileParams) validate() error {
	if err := validateTresorName(p.Tresor); err != nil {
		return err
	}
	return validateObjectKey(p.Key, "key")
}

func (a *downloadFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[downloadFileParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	respBody, err := a.conn.do(ctx, req.Credentials, "GET", objectPath(params.Tresor, params.Key), "", nil, "")
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"tresor":   params.Tresor,
		"key":      params.Key,
		"content":  base64.StdEncoding.EncodeToString(respBody),
		"size":     len(respBody),
		"encoding": "base64",
	})
}
