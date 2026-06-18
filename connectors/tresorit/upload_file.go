package tresorit

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type uploadFileAction struct {
	conn *TresoritConnector
}

type uploadFileParams struct {
	Tresor  string `json:"tresor"`
	Key     string `json:"key"`
	Content string `json:"content"`

	decoded []byte
}

func (p *uploadFileParams) validate() error {
	if err := validateTresorName(p.Tresor); err != nil {
		return err
	}
	if err := validateObjectKey(p.Key, "key"); err != nil {
		return err
	}
	if p.Content == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content"}
	}
	decoded, err := base64.StdEncoding.DecodeString(p.Content)
	if err != nil {
		return &connectors.ValidationError{Message: "content must be valid base64-encoded data"}
	}
	if len(decoded) > maxUploadBytes {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("file content is %d MB, which exceeds the maximum of %d MB", len(decoded)>>20, maxUploadBytes>>20),
		}
	}
	p.decoded = decoded
	return nil
}

func (a *uploadFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[uploadFileParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	_, err = a.conn.do(ctx, req.Credentials, "PUT", objectPath(params.Tresor, params.Key), "", params.decoded, "application/octet-stream")
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"tresor": params.Tresor,
		"key":    params.Key,
		"size":   len(params.decoded),
	})
}
