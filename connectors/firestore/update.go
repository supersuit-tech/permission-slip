package firestore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type updateAction struct {
	conn *FirestoreConnector
}

type updateParams struct {
	Path               string                 `json:"path"`
	Data               map[string]interface{} `json:"data"`
	AllowedPaths       []string               `json:"allowed_paths"`
	AllowedWriteFields []string               `json:"allowed_write_fields"`
}

func (p *updateParams) validate() error {
	if err := validateDocumentPath(p.Path); err != nil {
		return err
	}
	if err := validateAllowedPaths(p.Path, p.AllowedPaths, "document"); err != nil {
		return err
	}
	if p.Data == nil || len(p.Data) == 0 {
		return &connectors.ValidationError{Message: "data must not be empty for update"}
	}
	if len(p.AllowedWriteFields) > 0 {
		if err := validateFieldAllowlist(p.AllowedWriteFields); err != nil {
			return err
		}
		return validateMapKeysSubset(p.Data, buildFieldSet(p.AllowedWriteFields), "data")
	}
	return nil
}

func (a *updateAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateParams
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

	if err := runner.updateDocument(ctx, params.Path, params.Data); err != nil {
		return nil, err
	}
	return connectors.JSONResult(map[string]interface{}{
		"path":   params.Path,
		"status": "ok",
	})
}
