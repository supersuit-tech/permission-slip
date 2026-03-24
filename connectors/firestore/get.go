package firestore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type getAction struct {
	conn *FirestoreConnector
}

type getParams struct {
	Path                string   `json:"path"`
	AllowedPaths        []string `json:"allowed_paths"`
	AllowedReadFields   []string `json:"allowed_read_fields"`
}

func (p *getParams) validate() error {
	if err := validateDocumentPath(p.Path); err != nil {
		return err
	}
	if err := validateAllowedPaths(p.Path, p.AllowedPaths, "document"); err != nil {
		return err
	}
	if len(p.AllowedReadFields) > 0 {
		return validateFieldAllowlist(p.AllowedReadFields)
	}
	return nil
}

func (a *getAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	ctx, cancel := a.conn.withTimeout(ctx)
	defer cancel()

	runner, err := a.conn.openRunner(ctx, req.Credentials)
	if err != nil {
		return nil, err
	}
	defer func() { _ = runner.close() }()

	data, err := runner.getDocument(ctx, params.Path)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return connectors.JSONResult(map[string]interface{}{
			"found": false,
			"data":  nil,
		})
	}
	allowed := buildFieldSet(params.AllowedReadFields)
	data = filterMapKeys(data, allowed)
	return connectors.JSONResult(map[string]interface{}{
		"found": true,
		"data":  data,
	})
}
